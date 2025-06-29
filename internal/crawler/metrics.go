package crawler

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics holds all Prometheus metrics for the crawler
type Metrics struct {
	TracksProcessed prometheus.Counter
	CrawlErrors     prometheus.Counter
	QueueDepth      prometheus.Gauge
	APICallsTotal   prometheus.Counter
	APICallsErrors  prometheus.Counter
	JobDuration     prometheus.Histogram
	BatchSize       prometheus.Histogram
}

// NewMetrics creates a new metrics instance
func NewMetrics() *Metrics {
	return &Metrics{
		TracksProcessed: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rundj_crawler_tracks_processed_total",
			Help: "The total number of tracks processed by the crawler",
		}),
		CrawlErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rundj_crawler_errors_total",
			Help: "The total number of crawl errors",
		}),
		QueueDepth: promauto.NewGauge(prometheus.GaugeOpts{
			Name: "rundj_crawler_queue_depth",
			Help: "The current depth of the job queue",
		}),
		APICallsTotal: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rundj_crawler_api_calls_total",
			Help: "The total number of Spotify API calls made",
		}),
		APICallsErrors: promauto.NewCounter(prometheus.CounterOpts{
			Name: "rundj_crawler_api_errors_total",
			Help: "The total number of Spotify API errors",
		}),
		JobDuration: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rundj_crawler_job_duration_seconds",
			Help:    "The duration of job processing in seconds",
			Buckets: prometheus.DefBuckets,
		}),
		BatchSize: promauto.NewHistogram(prometheus.HistogramOpts{
			Name:    "rundj_crawler_batch_size",
			Help:    "The size of batches processed",
			Buckets: []float64{1, 5, 10, 25, 50, 75, 100},
		}),
	}
}