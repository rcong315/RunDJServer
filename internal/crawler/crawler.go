package crawler

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

// Crawler is the main crawler service
type Crawler struct {
	config      *Config
	scheduler   *Scheduler
	workers     []*Worker
	rateLimiter *rate.Limiter
	metrics     *Metrics
	jobs        chan CrawlJob
	quit        chan struct{}
	wg          sync.WaitGroup
}

// New creates a new crawler instance
func New(config *Config) (*Crawler, error) {
	// Create rate limiter (100 requests per minute)
	rateLimiter := rate.NewLimiter(rate.Every(time.Minute/DefaultRateLimit), DefaultRateLimit)

	// Create metrics
	metrics := NewMetrics()

	// Create job queue
	jobs := make(chan CrawlJob, JobQueueSize)

	// Create workers
	workers := make([]*Worker, config.Workers)
	for i := 0; i < config.Workers; i++ {
		workers[i] = &Worker{
			id:          i,
			jobs:        jobs,
			jobQueue:    jobs, // Workers can send new jobs to the same queue
			rateLimiter: rateLimiter,
			metrics:     metrics,
			logger:      config.Logger.With(zap.Int("worker_id", i)),
		}
	}

	// Create scheduler
	scheduler := NewScheduler(jobs, config.Logger)

	return &Crawler{
		config:      config,
		scheduler:   scheduler,
		workers:     workers,
		rateLimiter: rateLimiter,
		metrics:     metrics,
		jobs:        jobs,
		quit:        make(chan struct{}),
	}, nil
}

// Start starts the crawler service
func (c *Crawler) Start(ctx context.Context) error {
	c.config.Logger.Info("Starting crawler service")

	// Start metrics server
	go c.startMetricsServer()

	// Start workers
	for _, worker := range c.workers {
		c.wg.Add(1)
		go func(w *Worker) {
			defer c.wg.Done()
			w.Start(ctx)
		}(worker)
	}

	// Start scheduler
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.scheduler.Start(ctx)
	}()

	// Start monitoring goroutine
	c.wg.Add(1)
	go func() {
		defer c.wg.Done()
		c.monitor(ctx)
	}()

	c.config.Logger.Info("Crawler service started successfully")

	// Wait for context cancellation
	<-ctx.Done()

	c.config.Logger.Info("Shutting down crawler service...")

	// Close job queue
	close(c.jobs)

	// Wait for all workers to finish
	c.wg.Wait()

	c.config.Logger.Info("Crawler service shutdown complete")
	return nil
}

// startMetricsServer starts the Prometheus metrics HTTP server
func (c *Crawler) startMetricsServer() {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    ":" + c.config.MetricsPort,
		Handler: mux,
	}

	c.config.Logger.Info("Starting metrics server", zap.String("port", c.config.MetricsPort))

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		c.config.Logger.Error("Metrics server failed", zap.Error(err))
	}
}

// monitor runs periodic monitoring and logging
func (c *Crawler) monitor(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			queueDepth := len(c.jobs)
			c.metrics.QueueDepth.Set(float64(queueDepth))

			c.config.Logger.Debug("Crawler status",
				zap.Int("queue_depth", queueDepth),
				zap.Int("workers", len(c.workers)))
		}
	}
}
