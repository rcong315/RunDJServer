package crawler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/service"
)

type RefreshStaleDataJob struct {
	logger    *zap.Logger
	threshold time.Duration
}

func (j *RefreshStaleDataJob) Execute(pool *service.WorkerPool, jobWg *sync.WaitGroup, tracker *service.ProcessedTracker, stage *service.StageContext) error {
	defer jobWg.Done()
	defer stage.Wg.Done()

	j.logger.Info("Starting RefreshStaleDataJob", zap.Duration("threshold", j.threshold))

	// Find stale artists
	staleArtists, err := j.getStaleArtists()
	if err != nil {
		return fmt.Errorf("getting stale artists: %w", err)
	}

	// Find stale albums
	staleAlbums, err := j.getStaleAlbums()
	if err != nil {
		return fmt.Errorf("getting stale albums: %w", err)
	}

	j.logger.Info("Found stale entities",
		zap.Int("staleArtists", len(staleArtists)),
		zap.Int("staleAlbums", len(staleAlbums)))

	// Queue jobs to refresh stale artists
	for _, artistID := range staleArtists {
		// Queue top tracks refresh
		job := &service.SaveArtistTopTracksJob{
			ArtistId: artistID,
			Type:     service.TopArtists,
		}
		jobWg.Add(1)
		stage.Wg.Add(1)
		pool.SubmitWithStage(job, jobWg, stage)

		// Queue albums refresh
		albumJob := &service.SaveArtistAlbumsJob{
			ArtistId: artistID,
		}
		jobWg.Add(1)
		stage.Wg.Add(1)
		pool.SubmitWithStage(albumJob, jobWg, stage)
	}

	// Queue jobs to refresh stale albums
	for _, albumID := range staleAlbums {
		job := &ProcessAlbumJob{
			logger:  j.logger,
			albumID: albumID,
		}
		jobWg.Add(1)
		stage.Wg.Add(1)
		pool.SubmitWithStage(job, jobWg, stage)
	}

	j.logger.Info("Completed RefreshStaleDataJob",
		zap.Int("queuedArtists", len(staleArtists)),
		zap.Int("queuedAlbums", len(staleAlbums)))
	return nil
}

func (j *RefreshStaleDataJob) getStaleArtists() ([]string, error) {
	cutoffTime := time.Now().Add(-j.threshold)
	
	query := `
		SELECT artist_id
		FROM artist
		WHERE updated_at < $1
		ORDER BY updated_at ASC
		LIMIT 100
	`

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("getting database connection: %w", err)
	}

	rows, err := database.Query(context.Background(), query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("querying stale artists: %w", err)
	}
	defer rows.Close()

	var artistIDs []string
	for rows.Next() {
		var artistID string
		if err := rows.Scan(&artistID); err != nil {
			j.logger.Error("Failed to scan artist ID", zap.Error(err))
			continue
		}
		artistIDs = append(artistIDs, artistID)
	}

	return artistIDs, nil
}

func (j *RefreshStaleDataJob) getStaleAlbums() ([]string, error) {
	cutoffTime := time.Now().Add(-j.threshold)
	
	query := `
		SELECT album_id
		FROM album
		WHERE updated_at < $1
		ORDER BY updated_at ASC
		LIMIT 100
	`

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("getting database connection: %w", err)
	}

	rows, err := database.Query(context.Background(), query, cutoffTime)
	if err != nil {
		return nil, fmt.Errorf("querying stale albums: %w", err)
	}
	defer rows.Close()

	var albumIDs []string
	for rows.Next() {
		var albumID string
		if err := rows.Scan(&albumID); err != nil {
			j.logger.Error("Failed to scan album ID", zap.Error(err))
			continue
		}
		albumIDs = append(albumIDs, albumID)
	}

	return albumIDs, nil
}