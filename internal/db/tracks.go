package db

import (
	"context"
	"encoding/json"
	"fmt"
)

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
	err := batchAndSave(tracks, "track", func(item any, _ int) []any {
		track := item.(*Track)

		var audioFeaturesJSON string
		bpm := 0.0
		if track.AudioFeatures != nil {
			bpm = track.AudioFeatures.Tempo
			audioFeaturesBytes, err := json.Marshal(track.AudioFeatures)
			if err != nil {
			} else {
				audioFeaturesJSON = string(audioFeaturesBytes)
			}
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
		}
	})
	if err != nil {
		return fmt.Errorf("error saving tracks: %v", err)
	}

	return nil
}

func SaveUserTopTracks(userId string, tracks []*Track) error {
	err := batchAndSave(tracks, "userTopTrack", func(item any, rank int) []any {
		track := item.(*Track)
		return []any{
			userId,
			track.TrackId,
			rank,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user top tracks: %v", err)
	}

	return nil
}

func SaveUserSavedTracks(userId string, tracks []*Track) error {
	err := batchAndSave(tracks, "userSavedTrack", func(item any, _ int) []any {
		track := item.(*Track)
		return []any{
			userId,
			track.TrackId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user saved tracks: %v", err)
	}

	return nil
}

func GetTracksByBPM(userId string, min float64, max float64, sources []string) (map[string]float64, error) {
	tracks := make(map[string]float64)
	for _, source := range sources {
		var sqlFileName string
		if source == "top_tracks" {
			sqlFileName = "topTracksByBPM"
		} else if source == "saved_tracks" {
			sqlFileName = "savedTracksByBPM"
		} else if source == "playlists" {
			sqlFileName = "playlistsTracksByBPM"
		} else if source == "top_artists_top_tracks" {
			sqlFileName = "topArtistsTopTracksByBPM"
		} else if source == "top_artists_albums" {
			sqlFileName = "topArtistsAlbumsByBPM"
		} else if source == "top_artists_singles" {
			sqlFileName = "topArtistsSinglesByBPM"
		} else if source == "followed_artists_top_tracks" {
			sqlFileName = "followedArtistsTopTracksByBPM"
		} else if source == "followed_artists_albums" {
			sqlFileName = "followedArtistsAlbumsByBPM"
		} else if source == "followed_artists_singles" {
			sqlFileName = "followedArtistsSinglesByBPM"
		} else if source == "saved_albums" {
			sqlFileName = "savedAlbumsByBPM"
		} else {
			continue
		}

		rows, err := executeSelect(sqlFileName, userId, min, max)
		if err != nil {
			return nil, fmt.Errorf("error executing select: %v", err)
		}
		defer rows.Close()

		for rows.Next() {
			var track string
			var bpm float64
			err := rows.Scan(&track, &bpm)
			if err != nil {
				return nil, fmt.Errorf("error scanning track: %v", err)
			}
			tracks[track] = bpm
		}
	}

	return tracks, nil
}

func SaveFeedback(userId string, trackId string, feedback int) error {
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

	return nil
}
