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
	
	// Batching for audio features
	audioFeaturesBatch []CrawlJob
	batchTimer         *time.Timer
	
	// Job deduplication (track recently processed jobs)
	recentJobs map[string]time.Time
	
	// Batching for albums
	albumsBatch []string
	albumsBatchTimer *time.Timer
}

// Start starts the worker
func (w *Worker) Start(ctx context.Context) {
	w.logger.Info("Starting worker")
	
	// Initialize batching and deduplication
	w.audioFeaturesBatch = make([]CrawlJob, 0, AudioFeaturesBatchSize)
	w.albumsBatch = make([]string, 0, AlbumsBatchSize)
	w.recentJobs = make(map[string]time.Time)

	for {
		select {
		case <-ctx.Done():
			// Process any remaining batched jobs before stopping
			w.flushAudioFeaturesBatch(ctx)
			w.flushAlbumsBatch(ctx)
			w.logger.Info("Worker stopped")
			return
		case job, ok := <-w.jobs:
			if !ok {
				// Process any remaining batched jobs before stopping
				w.flushAudioFeaturesBatch(ctx)
				w.flushAlbumsBatch(ctx)
				w.logger.Info("Job channel closed, worker stopping")
				return
			}
			w.processJob(ctx, job)
		case <-w.getBatchTimerChannel():
			// Batch timeout - process accumulated audio features jobs
			w.flushAudioFeaturesBatch(ctx)
		case <-w.getAlbumsBatchTimerChannel():
			// Albums batch timeout - process accumulated album jobs
			w.flushAlbumsBatch(ctx)
		}
	}
}

// processJob processes a single crawl job
func (w *Worker) processJob(ctx context.Context, job CrawlJob) {
	start := time.Now()

	// Check for recent duplicate jobs (deduplication)
	jobKey := string(job.Type) + ":" + job.ID
	if lastProcessed, exists := w.recentJobs[jobKey]; exists {
		if time.Since(lastProcessed) < 5*time.Minute {
			w.logger.Debug("Skipping duplicate job", 
				zap.String("type", string(job.Type)),
				zap.String("id", job.ID),
				zap.Duration("since_last", time.Since(lastProcessed)))
			return
		}
	}
	w.recentJobs[jobKey] = time.Now()
	
	// Clean up old entries periodically
	if len(w.recentJobs) > 10000 {
		w.cleanupRecentJobs()
	}

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
		// Add to batch instead of processing immediately
		w.addToAudioFeaturesBatch(ctx, job)
		return // Don't process metrics/retry logic here since we're batching
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


// processArtistDiscoveryJob processes artist discovery jobs
func (w *Worker) processArtistDiscoveryJob(ctx context.Context, job CrawlJob) error {
	w.metrics.APICallsTotal.Inc()

	w.logger.Debug("Processing artist discovery job", zap.String("artist_id", job.ID))

	// Use existing spotify package to get artist albums
	err := spotify.GetArtistsAllAlbumTypes(job.ID, func(albums []*spotify.Album) error {
		w.logger.Debug("Retrieved artist albums batch",
			zap.String("artist_id", job.ID),
			zap.Int("album_count", len(albums)))

		// Save albums and batch album IDs for track discovery
		albumIds := make([]string, 0, len(albums))
		for _, album := range albums {
			if err := db.UpsertAlbum(ctx, album, job.ID); err != nil {
				w.logger.Error("Failed to save album",
					zap.String("album_id", album.Id),
					zap.Error(err))
				continue
			}
			albumIds = append(albumIds, album.Id)
		}
		
		// Add album IDs to batch instead of creating individual jobs
		w.addToAlbumsBatch(ctx, albumIds)
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

		// Save tracks and create jobs for their audio features (with rate limiting)
		jobsCreated := 0
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
				jobsCreated++
			default:
				w.logger.Warn("Job queue full, stopping track job creation for album", 
					zap.String("album_id", job.ID),
					zap.Int("jobs_created", jobsCreated))
				return nil // Stop creating more jobs if queue is full
			}
			
			// Add small delay every 10 jobs to avoid overwhelming the queue
			if jobsCreated%10 == 0 {
				time.Sleep(5 * time.Millisecond)
			}
		}
		
		w.logger.Debug("Created audio features jobs for album tracks",
			zap.String("album_id", job.ID),
			zap.Int("total_tracks", len(tracks)),
			zap.Int("jobs_created", jobsCreated))
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

// getBatchTimerChannel returns the timer channel or nil if no timer is active
func (w *Worker) getBatchTimerChannel() <-chan time.Time {
	if w.batchTimer != nil {
		return w.batchTimer.C
	}
	return nil
}

// getAlbumsBatchTimerChannel returns the albums batch timer channel or nil if no timer is active
func (w *Worker) getAlbumsBatchTimerChannel() <-chan time.Time {
	if w.albumsBatchTimer != nil {
		return w.albumsBatchTimer.C
	}
	return nil
}

// cleanupRecentJobs removes old entries from the deduplication map
func (w *Worker) cleanupRecentJobs() {
	cutoff := time.Now().Add(-10 * time.Minute)
	for key, timestamp := range w.recentJobs {
		if timestamp.Before(cutoff) {
			delete(w.recentJobs, key)
		}
	}
	w.logger.Debug("Cleaned up recent jobs cache", zap.Int("remaining", len(w.recentJobs)))
}

// addToAudioFeaturesBatch adds a job to the audio features batch
func (w *Worker) addToAudioFeaturesBatch(ctx context.Context, job CrawlJob) {
	w.audioFeaturesBatch = append(w.audioFeaturesBatch, job)
	
	// Start timer if this is the first job in the batch
	if len(w.audioFeaturesBatch) == 1 {
		w.batchTimer = time.NewTimer(AudioFeaturesBatchTimeout)
	}
	
	// Process batch if it's full
	if len(w.audioFeaturesBatch) >= AudioFeaturesBatchSize {
		w.flushAudioFeaturesBatch(ctx)
	}
}

// flushAudioFeaturesBatch processes all jobs in the current audio features batch
func (w *Worker) flushAudioFeaturesBatch(ctx context.Context) {
	if len(w.audioFeaturesBatch) == 0 {
		return
	}
	
	// Stop and drain the timer
	if w.batchTimer != nil {
		if !w.batchTimer.Stop() {
			select {
			case <-w.batchTimer.C:
			default:
			}
		}
		w.batchTimer = nil
	}
	
	batch := w.audioFeaturesBatch
	w.audioFeaturesBatch = make([]CrawlJob, 0, AudioFeaturesBatchSize)
	
	w.logger.Debug("Processing audio features batch", 
		zap.Int("batch_size", len(batch)))
	
	err := w.processBatchedAudioFeatures(ctx, batch)
	if err != nil {
		w.logger.Error("Failed to process audio features batch", 
			zap.Error(err), 
			zap.Int("batch_size", len(batch)))
		
		// Retry individual jobs on batch failure
		for _, job := range batch {
			if job.Retries < MaxRetries {
				w.retryJob(job)
			}
		}
	}
}

// processBatchedAudioFeatures processes a batch of audio features jobs
func (w *Worker) processBatchedAudioFeatures(ctx context.Context, jobs []CrawlJob) error {
	if len(jobs) == 0 {
		return nil
	}
	
	start := time.Now()
	
	// Wait for rate limiter (one call for the entire batch)
	if err := w.rateLimiter.Wait(ctx); err != nil {
		return fmt.Errorf("rate limiter wait failed: %w", err)
	}
	
	w.metrics.APICallsTotal.Inc()
	
	// Create track objects for all jobs
	tracks := make([]*spotify.Track, len(jobs))
	jobMap := make(map[string]CrawlJob) // Map track ID to original job
	
	for i, job := range jobs {
		tracks[i] = &spotify.Track{Id: job.ID}
		jobMap[job.ID] = job
	}
	
	w.logger.Debug("Fetching audio features for batch", 
		zap.Int("track_count", len(tracks)))
	
	// Make single API call for all tracks
	enrichedTracks, err := spotify.GetAudioFeatures(tracks)
	if err != nil {
		w.metrics.APICallsErrors.Inc()
		return fmt.Errorf("failed to get audio features for batch: %w", err)
	}
	
	// Process results and update database
	successCount := 0
	for _, enrichedTrack := range enrichedTracks {
		if enrichedTrack.AudioFeatures == nil {
			w.logger.Debug("No audio features available for track", 
				zap.String("track_id", enrichedTrack.Id))
			
			// Mark as processed with empty audio features
			err := db.UpdateTrackAudioFeatures(ctx, enrichedTrack.Id, map[string]interface{}{
				"id": enrichedTrack.Id,
				"processed_at": time.Now(),
			})
			if err != nil {
				w.logger.Error("Failed to mark track as processed", 
					zap.String("track_id", enrichedTrack.Id), 
					zap.Error(err))
				continue
			}
		} else {
			// Update with actual audio features
			audioFeatures := enrichedTrack.AudioFeatures
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
			
			err := db.UpdateTrackAudioFeatures(ctx, enrichedTrack.Id, featuresMap)
			if err != nil {
				w.logger.Error("Failed to update track audio features", 
					zap.String("track_id", enrichedTrack.Id), 
					zap.Error(err))
				continue
			}
			
			w.logger.Debug("Successfully updated audio features",
				zap.String("track_id", enrichedTrack.Id),
				zap.Float64("tempo", audioFeatures.Tempo),
				zap.Int("time_signature", audioFeatures.TimeSignature))
		}
		
		successCount++
		w.metrics.TracksProcessed.Inc()
	}
	
	duration := time.Since(start)
	w.metrics.JobDuration.Observe(duration.Seconds())
	w.metrics.BatchSize.Observe(float64(len(jobs)))
	
	w.logger.Info("Completed audio features batch",
		zap.Int("total_tracks", len(jobs)),
		zap.Int("successful", successCount),
		zap.Duration("duration", duration),
		zap.Float64("tracks_per_second", float64(len(jobs))/duration.Seconds()))
	
	return nil
}

// addToAlbumsBatch adds album IDs to the albums batch for track discovery
func (w *Worker) addToAlbumsBatch(ctx context.Context, albumIds []string) {
	for _, albumId := range albumIds {
		// Check for duplicates in current batch
		duplicate := false
		for _, existingId := range w.albumsBatch {
			if existingId == albumId {
				duplicate = true
				break
			}
		}
		if duplicate {
			continue
		}
		
		w.albumsBatch = append(w.albumsBatch, albumId)
		
		// Start timer if this is the first album in the batch
		if len(w.albumsBatch) == 1 {
			w.albumsBatchTimer = time.NewTimer(AlbumsBatchTimeout)
		}
		
		// Process batch if it's full
		if len(w.albumsBatch) >= AlbumsBatchSize {
			w.flushAlbumsBatch(ctx)
			return
		}
	}
}

// flushAlbumsBatch processes all album IDs in the current batch
func (w *Worker) flushAlbumsBatch(ctx context.Context) {
	if len(w.albumsBatch) == 0 {
		return
	}
	
	// Stop and drain the timer
	if w.albumsBatchTimer != nil {
		if !w.albumsBatchTimer.Stop() {
			select {
			case <-w.albumsBatchTimer.C:
			default:
			}
		}
		w.albumsBatchTimer = nil
	}
	
	batch := w.albumsBatch
	w.albumsBatch = make([]string, 0, AlbumsBatchSize)
	
	w.logger.Debug("Processing albums batch for track discovery", 
		zap.Int("batch_size", len(batch)))
	
	// Create individual album discovery jobs (we can't batch the actual Spotify API call for album tracks)
	// But we can at least batch the job creation and add rate limiting
	jobsCreated := 0
	for _, albumId := range batch {
		albumJob := CrawlJob{
			Type:     JobTypeDiscoveryAlbums,
			ID:       albumId,
			Priority: PriorityLow,
			Retries:  0,
		}

		select {
		case w.jobQueue <- albumJob:
			jobsCreated++
		default:
			w.logger.Warn("Job queue full, skipping album job", zap.String("album_id", albumId))
			break // Stop trying if queue is full
		}
		
		// Add small delay to avoid overwhelming the queue
		if jobsCreated%5 == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}
	
	w.logger.Info("Created album discovery jobs from batch",
		zap.Int("total_albums", len(batch)),
		zap.Int("jobs_created", jobsCreated))
}
