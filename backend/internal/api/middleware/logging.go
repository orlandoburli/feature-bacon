package middleware

import (
	"log/slog"
	"net/http"
	"time"
)

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(sw, r)

		attrs := []slog.Attr{
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.Int("status", sw.status),
			slog.Duration("duration", time.Since(start)),
		}

		if cid := w.Header().Get("X-Request-Id"); cid != "" {
			attrs = append(attrs, slog.String("correlation_id", cid))
		}

		if tid := TenantIDFromRequest(r); tid != "" {
			attrs = append(attrs, slog.String("tenant_id", tid))
		}

		slog.LogAttrs(r.Context(), slog.LevelInfo, "http request", attrs...)
	})
}
