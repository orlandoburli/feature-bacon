package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	LabelTenant      = "tenant"
	LabelFlagKey     = "flag_key"
	LabelResult      = "result"
	LabelEnvironment = "environment"
	LabelMethod      = "method"
	LabelPath        = "path"
	LabelStatus      = "status"
)

var (
	EvaluationsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bacon_evaluations_total",
		Help: "Total number of flag evaluations.",
	}, []string{LabelTenant, LabelFlagKey, LabelResult, LabelEnvironment})

	EvaluationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bacon_evaluation_duration_seconds",
		Help:    "Duration of flag evaluations in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{LabelTenant, LabelEnvironment})

	HTTPRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bacon_http_requests_total",
		Help: "Total HTTP requests.",
	}, []string{LabelMethod, LabelPath, LabelStatus})

	HTTPRequestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "bacon_http_request_duration_seconds",
		Help:    "HTTP request duration in seconds.",
		Buckets: prometheus.DefBuckets,
	}, []string{LabelMethod, LabelPath})

	GRPCRequestsTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bacon_grpc_requests_total",
		Help: "Total gRPC requests to modules.",
	}, []string{LabelMethod, LabelStatus})
)
