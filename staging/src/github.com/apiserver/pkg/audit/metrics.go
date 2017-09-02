package audit

import (
	auditinternal "github.com/HuZhou/apiserver/pkg/apis/audit"
	"github.com/prometheus/client_golang/prometheus"
)

const (
	subsystem = "apiserver_audit"
)

var (
	eventCounter = prometheus.NewCounter(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "event_total",
			Help:      "Counter of audit events generated and sent to the audit backend.",
		})
	errorCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "error_total",
			Help: "Counter of audit events that failed to be audited properly. " +
				"Plugin identifies the plugin affected by the error.",
		},
		[]string{"plugin"},
	)
	levelCounter = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Subsystem: subsystem,
			Name:      "level_total",
			Help:      "Counter of policy levels for audit events (1 per request).",
		},
		[]string{"level"},
	)
)

// ObservePolicyLevel updates the relevant prometheus metrics with the audit level for a request.
func ObservePolicyLevel(level auditinternal.Level) {
	levelCounter.WithLabelValues(string(level)).Inc()
}

// ObserveEvent updates the relevant prometheus metrics for the generated audit event.
func ObserveEvent() {
	eventCounter.Inc()
}