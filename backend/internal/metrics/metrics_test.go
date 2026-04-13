package metrics

import (
	"testing"

	"github.com/prometheus/client_golang/prometheus/testutil"
)

func TestEvaluationsTotal_Registered(t *testing.T) {
	EvaluationsTotal.Reset()
	EvaluationsTotal.WithLabelValues("acme", "dark-mode", "rule_match", "production").Inc()

	val := testutil.ToFloat64(EvaluationsTotal.WithLabelValues("acme", "dark-mode", "rule_match", "production"))
	if val != 1 {
		t.Errorf("expected counter = 1, got %v", val)
	}
}

func TestEvaluationDuration_Registered(t *testing.T) {
	EvaluationDuration.WithLabelValues("acme", "production").Observe(0.005)
}

func TestHTTPRequestsTotal_Registered(t *testing.T) {
	HTTPRequestsTotal.WithLabelValues("GET", "/healthz", "200").Inc()
}

func TestHTTPRequestDuration_Registered(t *testing.T) {
	HTTPRequestDuration.WithLabelValues("GET", "/healthz").Observe(0.01)
}

func TestGRPCRequestsTotal_Registered(t *testing.T) {
	GRPCRequestsTotal.WithLabelValues("GetFlag", "ok").Inc()
}
