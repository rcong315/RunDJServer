package service

import (
	"fmt"
	"sync"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func processTopTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	usersTopTracks, err := spotify.GetUsersTopTracks(token)
	if err != nil {
		return fmt.Errorf("getting top tracks: %w", err)
	}
	if len(usersTopTracks) == 0 {
		return nil
	}

	dbTracks := convertSpotifyTracksToDBTracks(usersTopTracks)
	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}
	err = db.SaveTracks(tracksToSave)
	if err != nil {
		return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
	}

	err = db.SaveUserTopTracks(userId, dbTracks)
	if err != nil {
		return fmt.Errorf("saving user-track relations: %w, tracks: %d", err, len(dbTracks))
	}

	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
	if err != nil {
		return fmt.Errorf("getting saved tracks: %w", err)
	}
	if len(usersSavedTracks) == 0 {
		return nil
	}

	dbTracks := convertSpotifyTracksToDBTracks(usersSavedTracks)

	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}

	err = db.SaveTracks(tracksToSave)
	if err != nil {
		return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
	}

	err = db.SaveUserSavedTracks(userId, dbTracks)
	if err != nil {
		return fmt.Errorf("saving user-track relations: %w, tracks: %d", err, len(dbTracks))
	}

	return nil
}
