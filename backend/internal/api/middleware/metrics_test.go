package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"

	"github.com/orlandoburli/feature-bacon/internal/metrics"
)

func TestMetrics_RecordsRequestTotal(t *testing.T) {
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	metrics.HTTPRequestsTotal.Reset()
	req := httptest.NewRequest(http.MethodGet, "/test-metrics", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	counter := metrics.HTTPRequestsTotal.WithLabelValues("GET", "/test-metrics", "200")
	val := testutil.ToFloat64(counter)
	if val != 1 {
		t.Errorf("expected counter = 1, got %v", val)
	}
}

func TestMetrics_RecordsRequestDuration(t *testing.T) {
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	metrics.HTTPRequestDuration.Reset()
	req := httptest.NewRequest(http.MethodGet, "/test-duration", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	count := testutil.CollectAndCount(metrics.HTTPRequestDuration)
	if count == 0 {
		t.Error("expected histogram to have observations")
	}
}

func TestMetrics_CapturesStatusCode(t *testing.T) {
	handler := Metrics(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))

	metrics.HTTPRequestsTotal.Reset()
	req := httptest.NewRequest(http.MethodPost, "/not-found", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	counter := metrics.HTTPRequestsTotal.WithLabelValues("POST", "/not-found", "404")
	val := testutil.ToFloat64(counter)
	if val != 1 {
		t.Errorf("expected counter = 1, got %v", val)
	}
}
