package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

const pathReadyz = "/readyz"

type stubChecker struct {
	name   string
	health ModuleHealth
}

func (s *stubChecker) CheckHealth(_ context.Context) (string, ModuleHealth) {
	return s.name, s.health
}

func TestHandleHealthz(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	w := httptest.NewRecorder()

	HandleHealthz().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf(fmtWantStatus, w.Code, http.StatusOK)
	}

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}

	var body map[string]string
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if body["status"] != "ok" {
		t.Errorf(fmtStatusFieldWant, body["status"], "ok")
	}
}

func TestHandleReadyz_NoCheckers(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, pathReadyz, nil)
	w := httptest.NewRecorder()

	HandleReadyz().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if body["status"] != "ready" {
		t.Errorf(fmtStatusFieldWant, body["status"], "ready")
	}
}

func TestHandleReadyz_AllHealthy(t *testing.T) {
	checkers := []HealthChecker{
		&stubChecker{name: "persistence", health: ModuleHealth{Status: "ok", LatencyMs: 5}},
		&stubChecker{name: "publisher", health: ModuleHealth{Status: "ok", LatencyMs: 3}},
	}

	req := httptest.NewRequest(http.MethodGet, pathReadyz, nil)
	w := httptest.NewRecorder()

	HandleReadyz(checkers...).ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf(fmtWantStatus, w.Code, http.StatusOK)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if body["status"] != "ready" {
		t.Errorf(fmtStatusFieldWant, body["status"], "ready")
	}

	modules, ok := body["modules"].(map[string]any)
	if !ok {
		t.Fatal("expected modules map in response")
	}
	if len(modules) != 2 {
		t.Errorf("expected 2 modules, got %d", len(modules))
	}
}

func TestHandleReadyz_OneUnhealthy(t *testing.T) {
	checkers := []HealthChecker{
		&stubChecker{name: "persistence", health: ModuleHealth{Status: "ok", LatencyMs: 5}},
		&stubChecker{name: "publisher", health: ModuleHealth{Status: "error", Message: "connection refused"}},
	}

	req := httptest.NewRequest(http.MethodGet, pathReadyz, nil)
	w := httptest.NewRecorder()

	HandleReadyz(checkers...).ServeHTTP(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf(fmtWantStatus, w.Code, http.StatusServiceUnavailable)
	}

	var body map[string]any
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf(fmtDecode, err)
	}
	if body["status"] != "not_ready" {
		t.Errorf(fmtStatusFieldWant, body["status"], "not_ready")
	}
}
