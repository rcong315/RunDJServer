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
	case JobTypeTrackRelationships:
		err = w.processTrackRelationshipsJob(ctx, job)
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

	w.logger.Debug("Processing audio features job", zap.String("track_id", job.ID))

	// Create a track object with just the ID to fetch audio features
	track := &spotify.Track{
		Id: job.ID,
	}

	// Use the existing getAudioFeatures function to fetch audio features
	enrichedTracks, err := spotify.GetAudioFeatures([]*spotify.Track{track})
	if err != nil {
		w.metrics.APICallsErrors.Inc()
		return fmt.Errorf("failed to get audio features for track %s: %w", job.ID, err)
	}

	// Check if we got audio features for our track
	if len(enrichedTracks) == 0 || enrichedTracks[0].AudioFeatures == nil {
		w.logger.Warn("No audio features returned for track", zap.String("track_id", job.ID))
		return fmt.Errorf("no audio features returned for track %s", job.ID)
	}

	enrichedTrack := enrichedTracks[0]
	audioFeatures := enrichedTrack.AudioFeatures

	// Convert to the format expected by UpdateTrackAudioFeatures
	featuresMap := map[string]interface{}{
		"id":               enrichedTrack.Id,
		"danceability":     audioFeatures.Danceability,
		"energy":           audioFeatures.Energy,
		"key":              audioFeatures.Key,
		"loudness":         audioFeatures.Loudness,
		"mode":             audioFeatures.Mode,
		"speechiness":      audioFeatures.Speechiness,
		"acousticness":     audioFeatures.Acousticness,
		"instrumentalness": audioFeatures.Instrumentallness,
		"liveness":         audioFeatures.Liveness,
		"valence":          audioFeatures.Valence,
		"tempo":            audioFeatures.Tempo,
		"duration_ms":      audioFeatures.Duration,
		"time_signature":   audioFeatures.TimeSignature,
	}

	// Update the track in the database with the audio features
	err = db.UpdateTrackAudioFeatures(ctx, job.ID, featuresMap)
	if err != nil {
		return fmt.Errorf("failed to update track audio features in database: %w", err)
	}

	w.logger.Debug("Successfully processed audio features job",
		zap.String("track_id", job.ID),
		zap.Float64("tempo", audioFeatures.Tempo),
		zap.Int("time_signature", audioFeatures.TimeSignature))

	return nil
}

// processArtistDiscoveryJob processes artist discovery jobs
func (w *Worker) processArtistDiscoveryJob(ctx context.Context, job CrawlJob) error {
	w.metrics.APICallsTotal.Inc()

	w.logger.Debug("Processing artist discovery job", zap.String("artist_id", job.ID))

	// Use existing spotify package to get artist albums
	err := spotify.GetArtistsAllAlbumTypes(job.ID, func(albums []*spotify.Album) error {
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

// processTrackRelationshipsJob processes track relationships jobs
func (w *Worker) processTrackRelationshipsJob(ctx context.Context, job CrawlJob) error {
	w.logger.Debug("Processing track relationships job", zap.String("track_id", job.ID))

	// Get the track's artist IDs and album ID
	artistIDs, albumID, err := db.GetTrackRelationships(ctx, job.ID)
	if err != nil {
		return fmt.Errorf("failed to get track relationships: %w", err)
	}

	w.logger.Debug("Retrieved track relationships",
		zap.String("track_id", job.ID),
		zap.Strings("artist_ids", artistIDs),
		zap.String("album_id", albumID))

	jobsCreated := 0

	// Check each artist to see if it needs crawling
	crawlThreshold := time.Now().Add(-ArtistCrawlInterval)
	for _, artistID := range artistIDs {
		needsCrawling, err := db.ArtistNeedsCrawling(ctx, artistID, crawlThreshold)
		if err != nil {
			w.logger.Error("Failed to check if artist needs crawling",
				zap.String("artist_id", artistID),
				zap.Error(err))
			continue
		}

		if needsCrawling {
			artistJob := CrawlJob{
				Type:     JobTypeDiscoveryArtists,
				ID:       artistID,
				Priority: PriorityLow,
				Retries:  0,
			}

			select {
			case w.jobQueue <- artistJob:
				jobsCreated++
				w.logger.Debug("Created artist discovery job", zap.String("artist_id", artistID))
			default:
				w.logger.Warn("Job queue full, skipping artist job", zap.String("artist_id", artistID))
			}
		}
	}

	// Check if the album needs track discovery
	if albumID != "" {
		needsTrackDiscovery, err := db.AlbumNeedsTrackDiscovery(ctx, albumID, crawlThreshold)
		if err != nil {
			w.logger.Error("Failed to check if album needs track discovery",
				zap.String("album_id", albumID),
				zap.Error(err))
		} else if needsTrackDiscovery {
			albumJob := CrawlJob{
				Type:     JobTypeDiscoveryAlbums,
				ID:       albumID,
				Priority: PriorityLow,
				Retries:  0,
			}

			select {
			case w.jobQueue <- albumJob:
				jobsCreated++
				w.logger.Debug("Created album discovery job", zap.String("album_id", albumID))
			default:
				w.logger.Warn("Job queue full, skipping album job", zap.String("album_id", albumID))
			}
		}
	}

	w.logger.Info("Track relationships job completed",
		zap.String("track_id", job.ID),
		zap.Int("jobs_created", jobsCreated))
	return nil
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
