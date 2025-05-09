package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// TODO: Update updated_at

// TODO: Save backups to filesystem

func SaveTracks(userId string, tracks []*Track, source string) error {
	if len(tracks) == 0 {
		return nil
	}

	err := batchAndSave(tracks, "insertTrack", func(item any) []any {
		track := item.(*Track)

		var audioFeaturesJSON string
		if track.AudioFeatures != nil {
			audioFeaturesBytes, err := json.Marshal(track.AudioFeatures)
			if err != nil {
				log.Printf("error marshaling audio features for track %s: %v", track.TrackId, err)
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
		}
	})
	if err != nil {
		return fmt.Errorf("error saving tracks: %v", err)
	}
	log.Printf("Saved %d tracks for user %s", len(tracks), userId)

	return nil
}

func SaveUserTopTracks(userId string, tracks []*Track) error {
	if len(tracks) == 0 {
		return nil
	}

	err := batchAndSave(tracks, "insertUserTopTrack", func(item any) []any {
		track := item.(*Track)
		return []any{
			userId,
			track.TrackId,
			0,
			//TODO: rank
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user top tracks: %v", err)
	}

	return nil
}

func SaveUserSavedTracks(userId string, tracks []*Track) error {
	if len(tracks) == 0 {
		return nil
	}

	err := batchAndSave(tracks, "insertUserSavedTrack", func(item any) []any {
		track := item.(*Track)
		return []any{
			userId,
			track.TrackId,
			0,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user saved tracks: %v", err)
	}

	return nil
}

func GetTracksByBPM(userId string, min float64, max float64, sources []string) (map[string]float64, error) {
	rows, err := executeSelect("selectTracksByBPM", userId, min, max, sources)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	tracks := make(map[string]float64)
	for rows.Next() {
		var track string
		var bpm float64
		err := rows.Scan(&track, &bpm)
		if err != nil {
			return nil, fmt.Errorf("error scanning track: %v", err)
		}
		tracks[track] = bpm
	}

	// TODO: Shuffle tracks

	log.Printf("Found %d tracks for user %s with BPM between %f and %f", len(tracks), userId, min, max)
	return tracks, nil
}

func SaveFeedback(userId string, trackId string, feedback int) error {
	sqlQuery, err := getQueryString("updateFeedback")
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
