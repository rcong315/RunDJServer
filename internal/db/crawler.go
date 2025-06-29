package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/spotify"
)

// GetTracksWithoutAudioFeatures returns track IDs that don't have audio features
func GetTracksWithoutAudioFeatures(ctx context.Context, limit int) ([]string, error) {
	query := `
		SELECT track_id 
		FROM track 
		WHERE (audio_features IS NULL OR audio_features = '{}' OR audio_features = 'null')
		   AND updated_at < NOW() - INTERVAL '1 hour'
		   AND (bpm IS NULL OR bpm = 0)
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
		SELECT artist_id 
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
	// Extract BPM and time signature from features
	var bpm float64
	var timeSignature int
	var audioFeaturesParam interface{}

	if featuresMap, ok := features.(map[string]interface{}); ok {
		// Check if this is just a "processed" marker (no actual audio features)
		if _, hasProcessedAt := featuresMap["processed_at"]; hasProcessedAt && len(featuresMap) <= 2 {
			// This is just marking the track as processed, set audio_features to NULL
			audioFeaturesParam = nil
		} else {
			// This has actual audio features, marshal to JSON
			audioFeaturesJSON, err := json.Marshal(features)
			if err != nil {
				return fmt.Errorf("failed to marshal audio features: %w", err)
			}
			audioFeaturesParam = string(audioFeaturesJSON)
		}

		if tempo, exists := featuresMap["tempo"]; exists {
			if tempoFloat, ok := tempo.(float64); ok {
				bpm = tempoFloat
			}
		}
		if ts, exists := featuresMap["time_signature"]; exists {
			if tsInt, ok := ts.(float64); ok {
				timeSignature = int(tsInt)
			} else if tsInt, ok := ts.(int); ok {
				timeSignature = tsInt
			}
		}
	} else {
		// Not a map, try to marshal directly
		audioFeaturesJSON, err := json.Marshal(features)
		if err != nil {
			return fmt.Errorf("failed to marshal audio features: %w", err)
		}
		audioFeaturesParam = string(audioFeaturesJSON)
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

	_, err = db.Exec(ctx, query, audioFeaturesParam, bpm, timeSignature, trackID)
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
	spotifyAlbum, ok := album.(*spotify.Album)
	if !ok {
		return fmt.Errorf("expected *spotify.Album, got %T", album)
	}

	// Convert Spotify album to DB album
	artistIds := make([]string, len(spotifyAlbum.Artists))
	for i, artist := range spotifyAlbum.Artists {
		artistIds[i] = artist.Id
	}

	imageURLs := make([]string, len(spotifyAlbum.Images))
	for i, img := range spotifyAlbum.Images {
		imageURLs[i] = img.URL
	}

	dbAlbum := &Album{
		AlbumId:          spotifyAlbum.Id,
		Name:             spotifyAlbum.Name,
		ArtistIds:        artistIds,
		Genres:           spotifyAlbum.Genres,
		Popularity:       spotifyAlbum.Popularity,
		AlbumType:        spotifyAlbum.AlbumType,
		TotalTracks:      spotifyAlbum.TotalTracks,
		ReleaseDate:      spotifyAlbum.ReleaseDate,
		AvailableMarkets: spotifyAlbum.AvailableMarkets,
		ImageURLs:        imageURLs,
	}

	// Save the album using the existing SaveAlbums function
	if err := SaveAlbums([]*Album{dbAlbum}); err != nil {
		return fmt.Errorf("failed to save album %s: %w", dbAlbum.AlbumId, err)
	}

	logger.Debug("Successfully upserted album",
		zap.String("album_id", dbAlbum.AlbumId),
		zap.String("artist_id", artistID))

	return nil
}

// UpsertTrack inserts or updates a track
func UpsertTrack(ctx context.Context, track interface{}, albumID string) error {
	spotifyTrack, ok := track.(*spotify.Track)
	if !ok {
		return fmt.Errorf("expected *spotify.Track, got %T", track)
	}

	// Convert Spotify track to DB track
	artistIds := make([]string, len(spotifyTrack.Artists))
	for i, artist := range spotifyTrack.Artists {
		artistIds[i] = artist.Id
	}

	var albumId string
	if spotifyTrack.Album != nil {
		albumId = spotifyTrack.Album.Id
	}

	var dbAudioFeatures *AudioFeatures
	if spotifyTrack.AudioFeatures != nil {
		dbAudioFeatures = &AudioFeatures{
			Danceability:      spotifyTrack.AudioFeatures.Danceability,
			Energy:            spotifyTrack.AudioFeatures.Energy,
			Key:               spotifyTrack.AudioFeatures.Key,
			Loudness:          spotifyTrack.AudioFeatures.Loudness,
			Mode:              spotifyTrack.AudioFeatures.Mode,
			Speechiness:       spotifyTrack.AudioFeatures.Speechiness,
			Acousticness:      spotifyTrack.AudioFeatures.Acousticness,
			Instrumentallness: spotifyTrack.AudioFeatures.Instrumentallness,
			Liveness:          spotifyTrack.AudioFeatures.Liveness,
			Valence:           spotifyTrack.AudioFeatures.Valence,
			Tempo:             spotifyTrack.AudioFeatures.Tempo,
			Duration:          spotifyTrack.AudioFeatures.Duration,
			TimeSignature:     spotifyTrack.AudioFeatures.TimeSignature,
		}
	}

	var bpm float64
	var timeSignature int
	if dbAudioFeatures != nil {
		bpm = dbAudioFeatures.Tempo
		timeSignature = dbAudioFeatures.TimeSignature
	}

	dbTrack := &Track{
		TrackId:          spotifyTrack.Id,
		Name:             spotifyTrack.Name,
		ArtistIds:        artistIds,
		AlbumId:          albumId,
		Popularity:       spotifyTrack.Popularity,
		DurationMS:       spotifyTrack.DurationMS,
		AvailableMarkets: spotifyTrack.AvailableMarkets,
		AudioFeatures:    dbAudioFeatures,
		BPM:              bpm,
		TimeSignature:    timeSignature,
	}

	// Save the track using the existing SaveTracks function
	if err := SaveTracks([]*Track{dbTrack}); err != nil {
		return fmt.Errorf("failed to save track %s: %w", dbTrack.TrackId, err)
	}

	logger.Debug("Successfully upserted track",
		zap.String("track_id", dbTrack.TrackId),
		zap.String("album_id", albumID))

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
		SELECT t.track_id, t.created_at
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
		var createdAt time.Time
		if err := rows.Scan(&trackID, &createdAt); err != nil {
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
