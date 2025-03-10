package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// EventsProcessed counts the total number of file events processed
	EventsProcessed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blink_events_processed_total",
		Help: "The total number of file events processed",
	})

	// EventsFiltered counts the total number of file events filtered out
	EventsFiltered = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blink_events_filtered_total",
		Help: "The total number of file events filtered out",
	})

	// ActiveWatchers tracks the number of active file watchers
	ActiveWatchers = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "blink_active_watchers",
		Help: "The number of active file watchers",
	})

	// WebhookLatency tracks webhook request latency
	WebhookLatency = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "blink_webhook_latency_seconds",
		Help:    "The latency of webhook requests",
		Buckets: prometheus.DefBuckets,
	})

	// WebhookErrors counts the total number of webhook errors
	WebhookErrors = promauto.NewCounter(prometheus.CounterOpts{
		Name: "blink_webhook_errors_total",
		Help: "The total number of webhook errors",
	})

	// MemoryUsage tracks the current memory usage
	MemoryUsage = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "blink_memory_bytes",
		Help: "Current memory usage in bytes",
	})
)
