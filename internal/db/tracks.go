package db

import (
	"context"
	"encoding/json"
	"fmt"

	"go.uber.org/zap"
)

// RankedTrack wraps a track with its ranking
type RankedTrack struct {
	Track *Track
	Rank  int
}

type Track struct {
	TrackId          string         `json:"track_id"`
	Name             string         `json:"name"`
	ArtistIds        []string       `json:"artist_ids"`
	AlbumId          string         `json:"album_id"`
	Popularity       int            `json:"popularity"`
	DurationMS       int            `json:"duration_ms"`
	AvailableMarkets []string       `json:"available_markets"`
	AudioFeatures    *AudioFeatures `json:"audio_features"`
	BPM              float64        `json:"bpm"`
	TimeSignature    int            `json:"time_signature"`
}

type AudioFeatures struct {
	Danceability      float64 `json:"danceability"`
	Energy            float64 `json:"energy"`
	Key               int     `json:"key"`
	Loudness          float64 `json:"loudness"`
	Mode              int     `json:"mode"`
	Speechiness       float64 `json:"speechiness"`
	Acousticness      float64 `json:"acousticness"`
	Instrumentallness float64 `json:"instrumentallness"`
	Liveness          float64 `json:"liveness"`
	Valence           float64 `json:"valence"`
	Tempo             float64 `json:"tempo"`
	Duration          int     `json:"duration_ms"`
	TimeSignature     int     `json:"time_signature"`
}

func SaveTracks(tracks []*Track) error {
	// TODO: remove 0 checks at db level
	if len(tracks) == 0 {
		logger.Debug("SaveTracks: No tracks to save.")
		return nil
	}
	logger.Debug("Attempting to save tracks", zap.Int("count", len(tracks)))

	err := batchAndSave(tracks, "track", func(item any) []any {
		track := item.(*Track)

		var audioFeaturesJSON interface{}
		bpm := 0.0
		timeSignature := 0
		if track.AudioFeatures != nil {
			bpm = track.AudioFeatures.Tempo
			timeSignature = track.AudioFeatures.TimeSignature
			audioFeaturesBytes, errMarshal := json.Marshal(track.AudioFeatures)
			if errMarshal != nil {
				// Log the error but continue, audioFeaturesJSON will be nil
				logger.Warn("SaveTracks: Error marshalling audio features for track",
					zap.String("trackId", track.TrackId),
					zap.Error(errMarshal))
				audioFeaturesJSON = nil
			} else {
				audioFeaturesJSON = string(audioFeaturesBytes)
			}
		} else {
			// Use nil for NULL in database instead of empty string
			audioFeaturesJSON = nil
		}

		return []any{
			track.TrackId,
			track.Name,
			track.ArtistIds,
			track.AlbumId,
			track.Popularity,
			track.DurationMS,
			track.AvailableMarkets,
			audioFeaturesJSON,
			bpm,
			timeSignature,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving tracks: %v", err)
	}

	logger.Debug("Successfully saved tracks batch", zap.Int("count", len(tracks)))
	return nil
}

func SaveUserTopTracks(userId string, rankedTracks []*RankedTrack) error {
	if len(rankedTracks) == 0 {
		logger.Debug("SaveUserTopTracks: No tracks to save for user.", zap.String("userId", userId))
		return nil
	}
	logger.Debug("Attempting to save user top tracks",
		zap.String("userId", userId),
		zap.Int("count", len(rankedTracks)))

	// Create a custom type for the batch save to include ranking
	type userTopTrackWithRank struct {
		userId  string
		trackId string
		rank    int
	}

	items := make([]userTopTrackWithRank, len(rankedTracks))
	for i, rt := range rankedTracks {
		items[i] = userTopTrackWithRank{
			userId:  userId,
			trackId: rt.Track.TrackId,
			rank:    rt.Rank,
		}
	}

	err := batchAndSave(items, "userTopTrack", func(item any) []any {
		track := item.(userTopTrackWithRank)
		return []any{
			track.userId,
			track.trackId,
			track.rank,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user top tracks: %v", err)
	}

	logger.Debug("Successfully saved user top tracks batch",
		zap.String("userId", userId),
		zap.Int("count", len(rankedTracks)))
	return nil
}

func SaveUserSavedTracks(userId string, tracks []*Track) error {
	if len(tracks) == 0 {
		logger.Debug("SaveUserSavedTracks: No tracks to save for user.", zap.String("userId", userId))
		return nil
	}
	logger.Debug("Attempting to save user saved tracks", zap.String("userId", userId), zap.Int("count", len(tracks)))

	err := batchAndSave(tracks, "userSavedTrack", func(item any) []any {
		track := item.(*Track)
		return []any{
			userId,
			track.TrackId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user saved tracks: %v", err)
	}

	logger.Debug("Successfully saved user saved tracks batch", zap.String("userId", userId), zap.Int("count", len(tracks)))
	return nil
}

func GetTracksByBPM(userId string, min float64, max float64, sources []string) (map[string]float64, error) {
	logger.Debug("Getting tracks by BPM for user",
		zap.String("userId", userId),
		zap.Float64("minBPM", min),
		zap.Float64("maxBPM", max),
		zap.Strings("sources", sources))

	tracks := make(map[string]float64)
	sqlFileMap := map[string]string{
		"top_tracks":                  "topTracksByBPM",
		"saved_tracks":                "savedTracksByBPM",
		"playlists":                   "playlistsTracksByBPM",
		"top_artists_top_tracks":      "topArtistsTopTracksByBPM",
		"top_artists_albums":          "topArtistsAlbumsByBPM",
		"top_artists_singles":         "topArtistsSinglesByBPM",
		"followed_artists_top_tracks": "followedArtistsTopTracksByBPM",
		"followed_artists_albums":     "followedArtistsAlbumsByBPM",
		"followed_artists_singles":    "followedArtistsSinglesByBPM",
		"saved_albums":                "savedAlbumsByBPM",
	}

	for _, source := range sources {
		queryName, ok := sqlFileMap[source]
		if !ok {
			logger.Warn("GetTracksByBPM: Unknown source provided", zap.String("userId", userId), zap.String("source", source))
			continue // Or return an error if sources must be valid
		}
		logger.Debug("GetTracksByBPM: Executing select for source",
			zap.String("userId", userId),
			zap.String("source", source),
			zap.String("queryName", queryName))

		rows, err := executeSelect(queryName, userId, min, max)
		if err != nil {
			return nil, fmt.Errorf("error executing select for source %s: %v", source, err)
		}

		processedRows := 0
		for rows.Next() {
			var track string
			var bpm float64
			err := rows.Scan(&track, &bpm)
			if err != nil {
				rows.Close() // Ensure rows is closed on scan error
				return nil, fmt.Errorf("error scanning track for source %s: %v", source, err)
			}
			tracks[track] = bpm
			processedRows++
		}
		rows.Close() // Close rows after successful iteration or if Next returns false
		logger.Debug("GetTracksByBPM: Finished processing source",
			zap.String("userId", userId),
			zap.String("source", source),
			zap.Int("processedRows", processedRows))
	}

	logger.Debug("GetTracksByBPM: Successfully retrieved tracks",
		zap.String("userId", userId),
		zap.Int("trackCount", len(tracks)))
	return tracks, nil
}

func GetTracksByTimeSignature(userId string, timeSignature int, sources []string) (map[string]int, error) {
	logger.Debug("Getting tracks by time signature for user",
		zap.String("userId", userId),
		zap.Int("timeSignature", timeSignature),
		zap.Strings("sources", sources))

	tracks := make(map[string]int)
	sqlFileMap := map[string]string{
		"top_tracks":                  "topTracksByTimeSignature",
		"saved_tracks":                "savedTracksByTimeSignature",
		"playlists":                   "playlistsTracksByTimeSignature",
		"top_artists_top_tracks":      "topArtistsTopTracksByTimeSignature",
		"top_artists_albums":          "topArtistsAlbumsByTimeSignature",
		"top_artists_singles":         "topArtistsSinglesByTimeSignature",
		"followed_artists_top_tracks": "followedArtistsTopTracksByTimeSignature",
		"followed_artists_albums":     "followedArtistsAlbumsByTimeSignature",
		"followed_artists_singles":    "followedArtistsSinglesByTimeSignature",
		"saved_albums":                "savedAlbumsByTimeSignature",
	}

	for _, source := range sources {
		queryName, ok := sqlFileMap[source]
		if !ok {
			logger.Warn("GetTracksByTimeSignature: Unknown source provided", zap.String("userId", userId), zap.String("source", source))
			continue
		}
		logger.Debug("GetTracksByTimeSignature: Executing select for source",
			zap.String("userId", userId),
			zap.String("source", source),
			zap.String("queryName", queryName))

		rows, err := executeSelect(queryName, userId, timeSignature)
		if err != nil {
			return nil, fmt.Errorf("error executing select for source %s: %v", source, err)
		}

		processedRows := 0
		for rows.Next() {
			var track string
			var ts int
			err := rows.Scan(&track, &ts)
			if err != nil {
				rows.Close()
				return nil, fmt.Errorf("error scanning track for source %s: %v", source, err)
			}
			tracks[track] = ts
			processedRows++
		}
		rows.Close()
		logger.Debug("GetTracksByTimeSignature: Finished processing source",
			zap.String("userId", userId),
			zap.String("source", source),
			zap.Int("processedRows", processedRows))
	}

	logger.Debug("GetTracksByTimeSignature: Successfully retrieved tracks",
		zap.String("userId", userId),
		zap.Int("trackCount", len(tracks)))
	return tracks, nil
}

func SaveFeedback(userId string, trackId string, feedback int) error {
	logger.Debug("Attempting to save feedback",
		zap.String("userId", userId),
		zap.String("trackId", trackId),
		zap.Int("feedback", feedback))

	sqlQuery, err := getQueryString("update", "feedback")
	if err != nil {
		return fmt.Errorf("error getting query string: %v", err)
	}

	db, err := getDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	_, err = db.Exec(context.Background(), sqlQuery,
		userId,
		trackId,
		feedback,
	)
	if err != nil {
		return fmt.Errorf("error creating feedback record: %v", err)
	}

	logger.Debug("Successfully saved feedback",
		zap.String("userId", userId),
		zap.String("trackId", trackId),
		zap.Int("feedback", feedback))
	return nil
}
