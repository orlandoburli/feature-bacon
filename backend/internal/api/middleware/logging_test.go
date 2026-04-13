package middleware

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRequestLogger_LogsFields(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-log", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}

	if entry["method"] != "GET" {
		t.Errorf("method = %v, want GET", entry["method"])
	}
	if entry["path"] != "/test-log" {
		t.Errorf("path = %v, want /test-log", entry["path"])
	}
	if entry["status"] != float64(200) {
		t.Errorf("status = %v, want 200", entry["status"])
	}
	if entry["msg"] != "http request" {
		t.Errorf("msg = %v, want 'http request'", entry["msg"])
	}
}

func TestRequestLogger_IncludesCorrelationID(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)

	handler := Correlation(RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})))

	req := httptest.NewRequest(http.MethodGet, "/corr-test", nil)
	req.Header.Set("X-Request-Id", "test-corr-123")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}

	if entry["correlation_id"] != "test-corr-123" {
		t.Errorf("correlation_id = %v, want test-corr-123", entry["correlation_id"])
	}
}

func TestRequestLogger_CapturesNon200Status(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	slog.SetDefault(logger)

	handler := RequestLogger(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
	}))

	req := httptest.NewRequest(http.MethodPost, "/bad", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	var entry map[string]any
	if err := json.Unmarshal(buf.Bytes(), &entry); err != nil {
		t.Fatalf("unmarshal log: %v", err)
	}
	if entry["status"] != float64(400) {
		t.Errorf("status = %v, want 400", entry["status"])
	}
}
