package crawler

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
)

// Scheduler manages the scheduling of crawl jobs
type Scheduler struct {
	jobs   chan<- CrawlJob
	logger *zap.Logger
}

// NewScheduler creates a new scheduler
func NewScheduler(jobs chan<- CrawlJob, logger *zap.Logger) *Scheduler {
	return &Scheduler{
		jobs:   jobs,
		logger: logger,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start(ctx context.Context) {
	s.logger.Info("Starting scheduler")

	// Schedule different types of jobs at different intervals
	go s.scheduleJob(ctx, JobTypeMissingAudioFeatures, 24*time.Hour, PriorityHigh)
	go s.scheduleJob(ctx, JobTypeTrackRelationships, 24*time.Hour, PriorityMedium)
	go s.scheduleJob(ctx, JobTypeDiscoveryArtists, 3.5*24*time.Hour, PriorityMedium)
	go s.scheduleJob(ctx, JobTypeDiscoveryAlbums, 7*24*time.Hour, PriorityMedium)
	go s.scheduleJob(ctx, JobTypeStaleRefresh, 7*24*time.Hour, PriorityLow)

	// Run initial job discovery
	s.discoverJobs(ctx)

	<-ctx.Done()
	s.logger.Info("Scheduler stopped")
}

// scheduleJob schedules a specific type of job at regular intervals
func (s *Scheduler) scheduleJob(ctx context.Context, jobType CrawlJobType, interval time.Duration, priority int) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	s.logger.Info("Scheduling job type",
		zap.String("job_type", string(jobType)),
		zap.Duration("interval", interval),
		zap.Int("priority", priority))

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.discoverJobsOfType(ctx, jobType, priority)
		}
	}
}

// discoverJobs discovers all types of jobs that need to be processed
func (s *Scheduler) discoverJobs(ctx context.Context) {
	s.logger.Info("Running initial job discovery")

	s.discoverJobsOfType(ctx, JobTypeMissingAudioFeatures, PriorityHigh)
	s.discoverJobsOfType(ctx, JobTypeStaleRefresh, PriorityMedium)
	s.discoverJobsOfType(ctx, JobTypeDiscoveryArtists, PriorityLow)
	s.discoverJobsOfType(ctx, JobTypeDiscoveryAlbums, PriorityLow)
	s.discoverJobsOfType(ctx, JobTypeTrackRelationships, PriorityMedium)
}

// discoverJobsOfType discovers jobs of a specific type
func (s *Scheduler) discoverJobsOfType(ctx context.Context, jobType CrawlJobType, priority int) {
	start := time.Now()

	switch jobType {
	case JobTypeMissingAudioFeatures:
		s.discoverMissingAudioFeatures(ctx, priority)
	case JobTypeStaleRefresh:
		s.discoverStaleData(ctx, priority)
	case JobTypeDiscoveryArtists:
		s.discoverArtistsToRefresh(ctx, priority)
	case JobTypeDiscoveryAlbums:
		s.discoverAlbumsToRefresh(ctx, priority)
	case JobTypeTrackRelationships:
		s.discoverTrackRelationships(ctx, priority)
	}

	s.logger.Debug("Job discovery completed",
		zap.String("job_type", string(jobType)),
		zap.Duration("duration", time.Since(start)))
}

// discoverMissingAudioFeatures finds tracks without audio features
func (s *Scheduler) discoverMissingAudioFeatures(ctx context.Context, priority int) {
	trackIDs, err := db.GetTracksWithoutAudioFeatures(ctx, 1000)
	if err != nil {
		s.logger.Error("Failed to get tracks without audio features", zap.Error(err))
		return
	}

	jobCount := 0
	for _, trackID := range trackIDs {
		job := CrawlJob{
			Type:     JobTypeMissingAudioFeatures,
			ID:       trackID,
			Priority: priority,
			Retries:  0,
		}

		select {
		case s.jobs <- job:
			jobCount++
			// Add small delay every 50 jobs to avoid overwhelming the queue
			if jobCount%50 == 0 {
				time.Sleep(10 * time.Millisecond)
			}
		case <-ctx.Done():
			return
		default:
			s.logger.Warn("Job queue full, skipping remaining audio features jobs", 
				zap.Int("jobs_created", jobCount),
				zap.Int("total_tracks", len(trackIDs)))
			return
		}
	}

	if jobCount > 0 {
		s.logger.Info("Discovered missing audio features jobs", zap.Int("count", jobCount))
	}
}

// discoverStaleData finds tracks with outdated audio features
func (s *Scheduler) discoverStaleData(ctx context.Context, priority int) {
	staleThreshold := time.Now().Add(-StaleDataThreshold)
	trackIDs, err := db.GetStaleAudioFeatures(ctx, staleThreshold, 500)
	if err != nil {
		s.logger.Error("Failed to get stale audio features", zap.Error(err))
		return
	}

	jobCount := 0
	for _, trackID := range trackIDs {
		job := CrawlJob{
			Type:     JobTypeStaleRefresh,
			ID:       trackID,
			Priority: priority,
			Retries:  0,
		}

		select {
		case s.jobs <- job:
			jobCount++
		case <-ctx.Done():
			return
		default:
			s.logger.Warn("Job queue full, skipping job", zap.String("track_id", trackID))
		}
	}

	if jobCount > 0 {
		s.logger.Info("Discovered stale data refresh jobs", zap.Int("count", jobCount))
	}
}

// discoverArtistsToRefresh finds artists that haven't been crawled recently
func (s *Scheduler) discoverArtistsToRefresh(ctx context.Context, priority int) {
	crawlThreshold := time.Now().Add(-ArtistCrawlInterval)
	artistIDs, err := db.GetArtistsToRefresh(ctx, crawlThreshold, 100)
	if err != nil {
		s.logger.Error("Failed to get artists to refresh", zap.Error(err))
		return
	}

	jobCount := 0
	for _, artistID := range artistIDs {
		job := CrawlJob{
			Type:     JobTypeDiscoveryArtists,
			ID:       artistID,
			Priority: priority,
			Retries:  0,
		}

		select {
		case s.jobs <- job:
			jobCount++
		case <-ctx.Done():
			return
		default:
			s.logger.Warn("Job queue full, skipping job", zap.String("artist_id", artistID))
		}
	}

	if jobCount > 0 {
		s.logger.Info("Discovered artist discovery jobs", zap.Int("count", jobCount))
	}
}

// discoverAlbumsToRefresh finds albums that need track discovery
func (s *Scheduler) discoverAlbumsToRefresh(ctx context.Context, priority int) {
	crawlThreshold := time.Now().Add(-ArtistCrawlInterval)
	albumIDs, err := db.GetAlbumsToRefresh(ctx, crawlThreshold, 200)
	if err != nil {
		s.logger.Error("Failed to get albums to refresh", zap.Error(err))
		return
	}

	jobCount := 0
	for _, albumID := range albumIDs {
		job := CrawlJob{
			Type:     JobTypeDiscoveryAlbums,
			ID:       albumID,
			Priority: priority,
			Retries:  0,
		}

		select {
		case s.jobs <- job:
			jobCount++
		case <-ctx.Done():
			return
		default:
			s.logger.Warn("Job queue full, skipping job", zap.String("album_id", albumID))
		}
	}

	if jobCount > 0 {
		s.logger.Info("Discovered album discovery jobs", zap.Int("count", jobCount))
	}
}

// discoverTrackRelationships finds tracks with unprocessed artists or albums
func (s *Scheduler) discoverTrackRelationships(ctx context.Context, priority int) {
	trackIDs, err := db.GetTracksWithUnprocessedRelationships(ctx, 500)
	if err != nil {
		s.logger.Error("Failed to get tracks with unprocessed relationships", zap.Error(err))
		return
	}

	jobCount := 0
	for _, trackID := range trackIDs {
		job := CrawlJob{
			Type:     JobTypeTrackRelationships,
			ID:       trackID,
			Priority: priority,
			Retries:  0,
		}

		select {
		case s.jobs <- job:
			jobCount++
		case <-ctx.Done():
			return
		default:
			s.logger.Warn("Job queue full, skipping job", zap.String("track_id", trackID))
		}
	}

	if jobCount > 0 {
		s.logger.Info("Discovered track relationships jobs", zap.Int("count", jobCount))
	}
}
