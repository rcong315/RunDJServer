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

// GetTracksWithUnprocessedRelationships returns track IDs that have artists or albums that haven't been processed
func GetTracksWithUnprocessedRelationships(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT DISTINCT t.track_id 
		FROM track t
		WHERE EXISTS (
			-- Check for artists that haven't been crawled recently or at all
			SELECT 1 FROM unnest(t.artist_ids) AS artist_id
			WHERE NOT EXISTS (
				SELECT 1 FROM artist a 
				WHERE a.artist_id = artist_id 
				AND a.last_crawled_at IS NOT NULL 
				AND a.last_crawled_at > NOW() - INTERVAL '7 days'
			)
		)
		OR EXISTS (
			-- Check for albums that haven't had tracks fetched recently or at all
			SELECT 1 FROM album al
			WHERE al.album_id = t.album_id
			AND (al.tracks_fetched_at IS NULL 
				 OR al.tracks_fetched_at < NOW() - INTERVAL '7 days')
		)
		ORDER BY t.created_at DESC
		LIMIT $1`

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("failed to get database connection: %w", err)
	}

	rows, err := db.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query tracks with unprocessed relationships: %w", err)
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

// GetTrackRelationships returns the artist IDs and album ID for a given track
func GetTrackRelationships(ctx context.Context, trackID string) (artistIDs []string, albumID string, err error) {
	query := `SELECT artist_ids, album_id FROM track WHERE track_id = $1`
	
	db, err := getDB()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get database connection: %w", err)
	}

	row := db.QueryRow(ctx, query, trackID)
	
	err = row.Scan(&artistIDs, &albumID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to get track relationships for track %s: %w", trackID, err)
	}

	return artistIDs, albumID, nil
}

// ArtistNeedsCrawling checks if an artist needs to be crawled based on last_crawled_at timestamp
func ArtistNeedsCrawling(ctx context.Context, artistID string, crawlThreshold time.Time) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM artist 
		WHERE artist_id = $1 
		AND (last_crawled_at IS NULL OR last_crawled_at < $2)`
	
	db, err := getDB()
	if err != nil {
		return false, fmt.Errorf("failed to get database connection: %w", err)
	}

	var count int
	err = db.QueryRow(ctx, query, artistID, crawlThreshold).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if artist needs crawling: %w", err)
	}

	return count > 0, nil
}

// AlbumNeedsTrackDiscovery checks if an album needs track discovery based on tracks_fetched_at timestamp
func AlbumNeedsTrackDiscovery(ctx context.Context, albumID string, crawlThreshold time.Time) (bool, error) {
	query := `
		SELECT COUNT(*) 
		FROM album 
		WHERE album_id = $1 
		AND (tracks_fetched_at IS NULL OR tracks_fetched_at < $2)`
	
	db, err := getDB()
	if err != nil {
		return false, fmt.Errorf("failed to get database connection: %w", err)
	}

	var count int
	err = db.QueryRow(ctx, query, albumID, crawlThreshold).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("failed to check if album needs track discovery: %w", err)
	}

	return count > 0, nil
}
