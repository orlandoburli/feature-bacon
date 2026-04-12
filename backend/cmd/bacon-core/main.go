package main

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/orlandoburli/feature-bacon/internal/api"
	"github.com/orlandoburli/feature-bacon/internal/config"
	"github.com/orlandoburli/feature-bacon/internal/configfile"
	"github.com/orlandoburli/feature-bacon/internal/engine"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	cfg := config.Load()

	var store engine.FlagStore

	switch cfg.Persistence {
	case "file":
		s, err := configfile.New(cfg.ConfigFile)
		if err != nil {
			slog.Error("failed to load config file", "path", cfg.ConfigFile, "error", err)
			os.Exit(1)
		}
		store = s
	default:
		slog.Error("unsupported persistence type", "persistence", cfg.Persistence)
		os.Exit(1)
	}

	eng := engine.New(store)
	router := api.NewRouter(eng)

	srv := &http.Server{
		Addr:              cfg.HTTPAddr,
		Handler:           router,
		ReadHeaderTimeout: 10 * time.Second,
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	if fs, ok := store.(*configfile.Store); ok {
		go fs.WatchSignal(ctx)
	}

	go func() {
		slog.Info("starting server", "addr", cfg.HTTPAddr)
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
