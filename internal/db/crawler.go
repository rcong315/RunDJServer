package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// GetTracksWithoutAudioFeatures returns track IDs that don't have audio features
func GetTracksWithoutAudioFeatures(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT track_id 
		FROM track 
		WHERE audio_features IS NULL 
		   OR bpm = 0 
		   OR time_signature = 0
		ORDER BY created_at DESC
		LIMIT $1`

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks without audio features: %w", err)
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil {
			logger.Error("Failed to scan track ID", zap.Error(err))
			continue
		}
		trackIDs = append(trackIDs, trackID)
	}

	return trackIDs, rows.Err()
}

// GetStaleAudioFeatures returns track IDs with outdated audio features
func GetStaleAudioFeatures(ctx context.Context, staleThreshold time.Time, limit int) ([]string, error) {
	query := `
		SELECT track_id 
		FROM track 
		WHERE updated_at < $1
		  AND audio_features IS NOT NULL
		ORDER BY updated_at ASC
		LIMIT $2`

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.Query(ctx, query, staleThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query stale audio features: %w", err)
	}
	defer rows.Close()

	var trackIDs []string
	for rows.Next() {
		var trackID string
		if err := rows.Scan(&trackID); err != nil {
			logger.Error("Failed to scan track ID", zap.Error(err))
			continue
		}
		trackIDs = append(trackIDs, trackID)
	}

	return trackIDs, rows.Err()
}

// GetArtistsToRefresh returns artist IDs that haven't been crawled recently
func GetArtistsToRefresh(ctx context.Context, crawlThreshold time.Time, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT artist_id 
		FROM artist 
		WHERE last_crawled_at < $1 
		   OR last_crawled_at IS NULL
		ORDER BY last_crawled_at ASC NULLS FIRST
		LIMIT $2`

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.Query(ctx, query, crawlThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query artists to refresh: %w", err)
	}
	defer rows.Close()

	var artistIDs []string
	for rows.Next() {
		var artistID string
		if err := rows.Scan(&artistID); err != nil {
			logger.Error("Failed to scan artist ID", zap.Error(err))
			continue
		}
		artistIDs = append(artistIDs, artistID)
	}

	return artistIDs, rows.Err()
}

// GetAlbumsToRefresh returns album IDs that need track discovery
func GetAlbumsToRefresh(ctx context.Context, crawlThreshold time.Time, limit int) ([]string, error) {
	query := `
		SELECT album_id 
		FROM album 
		WHERE tracks_fetched_at < $1 
		   OR tracks_fetched_at IS NULL
		ORDER BY tracks_fetched_at ASC NULLS FIRST
		LIMIT $2`

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.Query(ctx, query, crawlThreshold, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query albums to refresh: %w", err)
	}
	defer rows.Close()

	var albumIDs []string
	for rows.Next() {
		var albumID string
		if err := rows.Scan(&albumID); err != nil {
			logger.Error("Failed to scan album ID", zap.Error(err))
			continue
		}
		albumIDs = append(albumIDs, albumID)
	}

	return albumIDs, rows.Err()
}

// UpdateTrackAudioFeatures updates a track's audio features
func UpdateTrackAudioFeatures(ctx context.Context, trackID string, features interface{}) error {
	// Convert features to JSON
	audioFeaturesJSON, err := json.Marshal(features)
	if err != nil {
		return fmt.Errorf("failed to marshal audio features: %w", err)
	}

	// Extract BPM and time signature from features
	var bpm float64
	var timeSignature int

	if featuresMap, ok := features.(map[string]interface{}); ok {
		if tempo, exists := featuresMap["tempo"]; exists {
			if tempoFloat, ok := tempo.(float64); ok {
				bpm = tempoFloat
			}
		}
		if ts, exists := featuresMap["time_signature"]; exists {
			if tsInt, ok := ts.(float64); ok {
				timeSignature = int(tsInt)
			}
		}
	}

	query := `
		UPDATE track 
		SET audio_features = $1,
		    bpm = $2,
		    time_signature = $3,
		    updated_at = NOW()
		WHERE track_id = $4`

	db, err := getDB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	_, err = db.Exec(ctx, query, string(audioFeaturesJSON), bpm, timeSignature, trackID)
	if err != nil {
		return fmt.Errorf("failed to update track audio features: %w", err)
	}

	logger.Debug("Updated track audio features",
		zap.String("track_id", trackID),
		zap.Float64("bpm", bpm),
		zap.Int("time_signature", timeSignature))

	return nil
}

// UpsertAlbum inserts or updates an album
func UpsertAlbum(ctx context.Context, album interface{}, artistID string) error {
	// This would need to be implemented based on the album structure from spotify package
	// For now, return a placeholder implementation
	logger.Debug("UpsertAlbum called", zap.String("artist_id", artistID))
	return nil
}

// UpsertTrack inserts or updates a track
func UpsertTrack(ctx context.Context, track interface{}, albumID string) error {
	// This would need to be implemented based on the track structure from spotify package
	// For now, return a placeholder implementation
	logger.Debug("UpsertTrack called", zap.String("album_id", albumID))
	return nil
}

// UpdateArtistCrawlTime updates the last crawled timestamp for an artist
func UpdateArtistCrawlTime(ctx context.Context, artistID string) error {
	query := `UPDATE artist SET last_crawled_at = NOW() WHERE artist_id = $1`
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	_, err = db.Exec(ctx, query, artistID)
	if err != nil {
		return fmt.Errorf("failed to update artist crawl time: %w", err)
	}

	logger.Debug("Updated artist crawl time", zap.String("artist_id", artistID))
	return nil
}

// UpdateAlbumTracksTime updates the tracks fetched timestamp for an album
func UpdateAlbumTracksTime(ctx context.Context, albumID string) error {
	query := `UPDATE album SET tracks_fetched_at = NOW() WHERE album_id = $1`
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	_, err = db.Exec(ctx, query, albumID)
	if err != nil {
		return fmt.Errorf("failed to update album tracks time: %w", err)
	}

	logger.Debug("Updated album tracks time", zap.String("album_id", albumID))
	return nil
}
