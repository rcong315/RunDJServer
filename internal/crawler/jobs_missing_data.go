package crawler

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/service"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type RefetchMissingDataJob struct {
	logger *zap.Logger
}

func (j *RefetchMissingDataJob) Execute(pool *service.WorkerPool, jobWg *sync.WaitGroup, tracker *service.ProcessedTracker, stage *service.StageContext) error {
	defer jobWg.Done()
	defer stage.Wg.Done()

	j.logger.Info("Starting RefetchMissingDataJob")

	// Get tracks with missing BPM or time signature
	tracks, err := j.getTracksWithMissingData()
	if err != nil {
		return fmt.Errorf("getting tracks with missing data: %w", err)
	}

	if len(tracks) == 0 {
		j.logger.Info("No tracks with missing data found")
		return nil
	}

	j.logger.Info("Found tracks with missing data", zap.Int("count", len(tracks)))

	// Process in batches
	batchProcessor := spotify.NewBatchProcessor(100, func(trackBatch []*spotify.Track) error {
		// Get audio features for the batch
		enrichedTracks, err := j.getAudioFeaturesForTracks(trackBatch)
		if err != nil {
			j.logger.Error("Failed to get audio features for batch", zap.Error(err))
			// Continue processing other batches
			return nil
		}

		// Convert and save
		dbTracks := service.ConvertSpotifyTracksToDBTracks(enrichedTracks)
		if err := db.SaveTracks(dbTracks); err != nil {
			return fmt.Errorf("saving tracks: %w", err)
		}

		j.logger.Debug("Updated tracks with audio features", zap.Int("count", len(dbTracks)))
		return nil
	})

	// Add all tracks to batch processor
	for _, track := range tracks {
		if err := batchProcessor.Add(track); err != nil {
			j.logger.Error("Failed to add track to batch", zap.Error(err))
		}
	}

	// Process remaining tracks
	if err := batchProcessor.Flush(); err != nil {
		return fmt.Errorf("flushing batch processor: %w", err)
	}

	j.logger.Info("Completed RefetchMissingDataJob", zap.Int("processedCount", len(tracks)))
	return nil
}

func (j *RefetchMissingDataJob) getTracksWithMissingData() ([]*spotify.Track, error) {
	query := `
		SELECT track_id, name, artist_ids, album_id, popularity, duration_ms, available_markets
		FROM track
		WHERE bpm = 0 OR time_signature = 0 OR bpm IS NULL OR time_signature IS NULL
		LIMIT 1000
	`

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("getting database connection: %w", err)
	}

	rows, err := database.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("querying tracks: %w", err)
	}
	defer rows.Close()

	var tracks []*spotify.Track
	for rows.Next() {
		var trackID, name, albumID string
		var artistIDs, availableMarkets []string
		var popularity, durationMS int

		err := rows.Scan(&trackID, &name, &artistIDs, &albumID, &popularity, &durationMS, &availableMarkets)
		if err != nil {
			j.logger.Error("Failed to scan track row", zap.Error(err))
			continue
		}

		// Convert to Spotify track format
		track := &spotify.Track{
			Id:               trackID,
			Name:             name,
			Popularity:       popularity,
			DurationMS:       durationMS,
			AvailableMarkets: availableMarkets,
		}

		// Add album if present
		if albumID != "" {
			track.Album = &spotify.Album{Id: albumID}
		}

		// Add artists
		for _, artistID := range artistIDs {
			track.Artists = append(track.Artists, &spotify.Artist{Id: artistID})
		}

		tracks = append(tracks, track)
	}

	return tracks, nil
}

func (j *RefetchMissingDataJob) getAudioFeaturesForTracks(tracks []*spotify.Track) ([]*spotify.Track, error) {
	// Get secret token for Spotify API
	token, err := spotify.GetSecretToken()
	if err != nil {
		return nil, fmt.Errorf("getting Spotify token: %w", err)
	}

	// Use the existing audio features fetcher
	enrichedTracks, err := spotify.GetAudioFeatures(token, tracks)
	if err != nil {
		return nil, fmt.Errorf("getting audio features: %w", err)
	}

	return enrichedTracks, nil
}