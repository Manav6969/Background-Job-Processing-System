package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	JobsEnqueuedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "jobs_enqueued_total",
			Help: "Total number of jobs enqueued by priority",
		},
		[]string{"priority"},
	)

	JobsCompletedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jobs_completed_total",
			Help: "Total number of successfully completed jobs",
		},
	)

	JobsFailedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jobs_failed_total",
			Help: "Total number of jobs that failed and will be retried",
		},
	)

	JobsDeadTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "jobs_dead_total",
			Help: "Total number of jobs moved to the dead letter queue",
		},
	)

	JobDurationSeconds = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "job_duration_seconds",
			Help:    "Execution time distribution of background jobs",
			Buckets: prometheus.DefBuckets,
		},
	)

	WorkerGoroutines = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "worker_goroutines",
			Help: "Current number of active worker goroutines",
		},
	)
)
