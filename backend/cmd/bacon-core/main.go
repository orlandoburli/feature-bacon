package main

import (
	"context"
	crypto_tls "crypto/tls"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	baconapi "github.com/orlandoburli/feature-bacon/internal/api"
	"github.com/orlandoburli/feature-bacon/internal/api/handlers"
	"github.com/orlandoburli/feature-bacon/internal/auth"
	"github.com/orlandoburli/feature-bacon/internal/config"
	"github.com/orlandoburli/feature-bacon/internal/configfile"
	"github.com/orlandoburli/feature-bacon/internal/engine"
	"github.com/orlandoburli/feature-bacon/internal/grpcclient"
	"github.com/orlandoburli/feature-bacon/internal/publisher"
	"github.com/orlandoburli/feature-bacon/internal/tlsutil"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	var (
		store       engine.FlagStore
		assignments engine.AssignmentStore
		fileStore   *configfile.Store
		grpcPersist *grpcclient.PersistenceClient
		flagMgr     handlers.FlagManager
		expMgr      handlers.ExperimentManager
		apiKeyMgr   handlers.APIKeyManager
	)

	switch cfg.Persistence {
	case "file":
		s, err := configfile.New(cfg.ConfigFile)
		if err != nil {
			slog.Error("failed to load config file", "path", cfg.ConfigFile, "error", err)
			os.Exit(1)
		}
		store = s
		fileStore = s
	case "grpc":
		tlsCfg := tlsutil.Config{
			CAFile:   cfg.TLSCA,
			CertFile: cfg.TLSCert,
			KeyFile:  cfg.TLSKey,
		}
		var tc *crypto_tls.Config
		if tlsCfg.Enabled() {
			var err error
			tc, err = tlsutil.ClientTLSConfig(tlsCfg)
			if err != nil {
				slog.Error("failed to load TLS config", "error", err)
				os.Exit(1)
			}
			slog.Info("mTLS enabled for persistence module")
		}
		conn, err := grpcclient.Dial(cfg.PersistenceAddr, tc)
		if err != nil {
			slog.Error("failed to connect to persistence module", "addr", cfg.PersistenceAddr, "error", err)
			os.Exit(1)
		}
		grpcPersist = grpcclient.NewPersistenceClient(conn)
		store = grpcPersist
		assignments = grpcPersist
		slog.Info("connected to persistence module", "addr", cfg.PersistenceAddr)

		var publishers []publisher.Publisher
		for _, addr := range strings.Split(cfg.PublisherAddrs, ",") {
			addr = strings.TrimSpace(addr)
			if addr == "" {
				continue
			}
			pConn, pErr := grpcclient.Dial(addr, tc)
			if pErr != nil {
				slog.Error("failed to connect to publisher", "addr", addr, "error", pErr)
				os.Exit(1)
			}
			publishers = append(publishers, grpcclient.NewPublisherClient(pConn))
			slog.Info("connected to publisher module", "addr", addr)
		}
		fanout := publisher.NewFanout(publishers...)

		flagMgr = handlers.NewPublishingFlagManager(grpcclient.NewFlagManagerAdapter(grpcPersist), fanout)
		expMgr = handlers.NewPublishingExperimentManager(grpcclient.NewExperimentManagerAdapter(grpcPersist), fanout)
		apiKeyMgr = grpcclient.NewAPIKeyManagerAdapter(grpcPersist)
	default:
		slog.Error("unsupported persistence type", "persistence", cfg.Persistence)
		os.Exit(1)
	}

	keyStore := auth.NewMemKeyStore()
	if err := loadAPIKeys(cfg, keyStore, fileStore); err != nil {
		slog.Error("failed to load API keys", "error", err)
		os.Exit(1)
	}

	var jwtValidator *auth.JWTValidator
	jwtEnabled := cfg.JWTJWKSURL != ""
	if jwtEnabled {
		jwtValidator = auth.NewJWTValidator(auth.JWTConfig{
			Issuer:      cfg.JWTIssuer,
			Audience:    cfg.JWTAudience,
			JWKSURL:     cfg.JWTJWKSURL,
			TenantClaim: cfg.JWTTenantClaim,
			ScopeClaim:  cfg.JWTScopeClaim,
		})
		slog.Info("JWT authentication enabled", "issuer", cfg.JWTIssuer)
	}

	if grpcPersist != nil {
		defer func() { _ = grpcPersist.Close() }()
	}

	eng := engine.New(store, assignments)
	router := baconapi.NewRouter(baconapi.RouterConfig{
		Engine:            eng,
		AuthDisabled:      !cfg.AuthEnabled,
		KeyStore:          keyStore,
		JWTValidator:      jwtValidator,
		JWTEnabled:        jwtEnabled,
		FlagManager:       flagMgr,
		ExperimentManager: expMgr,
		APIKeyManager:     apiKeyMgr,
	})

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if fileStore != nil {
		go fileStore.WatchSignal(ctx)
	}

	go func() {
		slog.Info("starting server", "addr", cfg.HTTPAddr, "auth_enabled", cfg.AuthEnabled)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			slog.Error("server listen error", "error", err)
			os.Exit(1)
		}
	}()

	<-ctx.Done()
	slog.Info("shutting down")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		slog.Error("shutdown error", "error", err)
	}
}

func loadAPIKeys(cfg config.Config, keyStore *auth.MemKeyStore, fileStore *configfile.Store) error {
	if cfg.APIKeys != "" {
		if err := auth.LoadKeysFromEnv(keyStore, cfg.APIKeys, configfile.DefaultTenant); err != nil {
			return err
		}
		slog.Info("loaded API keys from environment")
	}

	if fileStore != nil {
		for tid, entries := range fileStore.APIKeys() {
			cfgKeys := make([]auth.ConfigFileKey, len(entries))
			for i, e := range entries {
				cfgKeys[i] = auth.ConfigFileKey{Key: e.Key, Scope: e.Scope, Name: e.Name}
			}
			if err := auth.LoadKeysFromConfig(keyStore, cfgKeys, tid); err != nil {
				return err
			}
		}
		slog.Info("loaded API keys from config file")
	}
	return nil
}
