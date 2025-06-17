package crawler

import (
	"context"
	"fmt"
	"math"
	"time"

	"go.uber.org/zap"
	"golang.org/x/time/rate"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

// Worker processes crawl jobs
type Worker struct {
	id          int
	jobs        <-chan CrawlJob
	jobQueue    chan<- CrawlJob // For creating new jobs
	rateLimiter *rate.Limiter
	metrics     *Metrics
	logger      *zap.Logger
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Starting worker")

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("Worker stopped")
			return
		case job, ok := <-w.jobs:
			if !ok {
				w.logger.Info("Job channel closed, worker stopping")
				return
			}
			w.processJob(ctx, job)
		}
	}
}

// processJob processes a single crawl job
func (w *Worker) processJob(ctx context.Context, job CrawlJob) {
	start := time.Now()

	w.logger.Debug("Processing job",
		zap.String("type", string(job.Type)),
		zap.String("id", job.ID),
		zap.Int("priority", job.Priority),
		zap.Int("retries", job.Retries))

	// Wait for rate limiter
	if err := w.rateLimiter.Wait(ctx); err != nil {
		w.logger.Error("Rate limiter wait failed", zap.Error(err))
		return
	}

	var err error
	switch job.Type {
	case JobTypeMissingAudioFeatures, JobTypeStaleRefresh:
		err = w.processAudioFeaturesJob(ctx, job)
	case JobTypeDiscoveryArtists:
		err = w.processArtistDiscoveryJob(ctx, job)
	case JobTypeDiscoveryAlbums:
		err = w.processAlbumDiscoveryJob(ctx, job)
	default:
		w.logger.Error("Unknown job type", zap.String("type", string(job.Type)))
		return
	}

	duration := time.Since(start)
	w.metrics.JobDuration.Observe(duration.Seconds())

	if err != nil {
		w.metrics.CrawlErrors.Inc()
		w.logger.Error("Job processing failed",
			zap.String("type", string(job.Type)),
			zap.String("id", job.ID),
			zap.Error(err),
			zap.Duration("duration", duration))

		// Retry logic
		if job.Retries < MaxRetries {
			w.retryJob(job)
		}
	} else {
		w.metrics.TracksProcessed.Inc()
		w.logger.Debug("Job completed successfully",
			zap.String("type", string(job.Type)),
			zap.String("id", job.ID),
			zap.Duration("duration", duration))
	}
}

// processAudioFeaturesJob processes audio features jobs
func (w *Worker) processAudioFeaturesJob(ctx context.Context, job CrawlJob) error {
	w.metrics.APICallsTotal.Inc()

	// Get audio features for the track using existing spotify package
	// Note: The existing package doesn't have a direct GetAudioFeatures function
	// This would need to be implemented or we could use the existing track fetching
	// For now, we'll create a placeholder that logs the action
	w.logger.Info("Processing audio features job", zap.String("track_id", job.ID))
	
	// TODO: Implement audio features fetching using the existing spotify package pattern
	// This might involve extending the spotify package to support direct audio features fetching
	
	return nil
}

// processArtistDiscoveryJob processes artist discovery jobs
func (w *Worker) processArtistDiscoveryJob(ctx context.Context, job CrawlJob) error {
	w.metrics.APICallsTotal.Inc()

	w.logger.Debug("Processing artist discovery job", zap.String("artist_id", job.ID))

	// Use existing spotify package to get artist albums
	err := spotify.GetArtistsAlbumsAndSingles(job.ID, func(albums []*spotify.Album) error {
		w.logger.Debug("Retrieved artist albums batch",
			zap.String("artist_id", job.ID),
			zap.Int("album_count", len(albums)))

		// Save albums and create jobs for their tracks
		for _, album := range albums {
			if err := db.UpsertAlbum(ctx, album, job.ID); err != nil {
				w.logger.Error("Failed to save album",
					zap.String("album_id", album.Id),
					zap.Error(err))
				continue
			}

			// Create job to crawl album tracks
			albumJob := CrawlJob{
				Type:     JobTypeDiscoveryAlbums,
				ID:       album.Id,
				Priority: PriorityLow,
				Retries:  0,
			}

			select {
			case w.jobQueue <- albumJob:
			default:
				w.logger.Warn("Job queue full, skipping album job", zap.String("album_id", album.Id))
			}
		}
		return nil
	})

	if err != nil {
		w.metrics.APICallsErrors.Inc()
		return fmt.Errorf("failed to get artist albums: %w", err)
	}

	// Update artist's last crawled timestamp
	return db.UpdateArtistCrawlTime(ctx, job.ID)
}

// processAlbumDiscoveryJob processes album discovery jobs
func (w *Worker) processAlbumDiscoveryJob(ctx context.Context, job CrawlJob) error {
	w.metrics.APICallsTotal.Inc()

	w.logger.Debug("Processing album discovery job", zap.String("album_id", job.ID))

	// Use existing spotify package to get album tracks
	err := spotify.GetAlbumsTracks(job.ID, func(tracks []*spotify.Track) error {
		w.logger.Debug("Retrieved album tracks batch",
			zap.String("album_id", job.ID),
			zap.Int("track_count", len(tracks)))

		// Save tracks and create jobs for their audio features
		for _, track := range tracks {
			if err := db.UpsertTrack(ctx, track, job.ID); err != nil {
				w.logger.Error("Failed to save track",
					zap.String("track_id", track.Id),
					zap.Error(err))
				continue
			}

			// Create job to get audio features
			audioJob := CrawlJob{
				Type:     JobTypeMissingAudioFeatures,
				ID:       track.Id,
				Priority: PriorityHigh,
				Retries:  0,
			}

			select {
			case w.jobQueue <- audioJob:
			default:
				w.logger.Warn("Job queue full, skipping audio features job", zap.String("track_id", track.Id))
			}
		}
		return nil
	})

	if err != nil {
		w.metrics.APICallsErrors.Inc()
		return fmt.Errorf("failed to get album tracks: %w", err)
	}

	// Update album's tracks fetched timestamp
	return db.UpdateAlbumTracksTime(ctx, job.ID)
}

// retryJob retries a failed job with exponential backoff
func (w *Worker) retryJob(job CrawlJob) {
	job.Retries++

	// Calculate backoff delay
	backoffSeconds := math.Pow(2, float64(job.Retries))
	delay := time.Duration(backoffSeconds) * time.Second

	if delay > RetryBackoffMax {
		delay = RetryBackoffMax
	}
	if delay < RetryBackoffMin {
		delay = RetryBackoffMin
	}

	w.logger.Info("Retrying job",
		zap.String("type", string(job.Type)),
		zap.String("id", job.ID),
		zap.Int("retry", job.Retries),
		zap.Duration("delay", delay))

	go func() {
		time.Sleep(delay)
		select {
		case w.jobQueue <- job:
		default:
			w.logger.Warn("Job queue full, dropping retry",
				zap.String("type", string(job.Type)),
				zap.String("id", job.ID))
		}
	}()
}
