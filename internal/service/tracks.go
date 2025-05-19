package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func processTopTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	logger.Info("Processing user top tracks", zap.String("userId", userId))
	usersTopTracks, err := spotify.GetUsersTopTracks(token)
	if err != nil {
		logger.Error("Error getting user top tracks from Spotify", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting top tracks: %w", err)
	}
	if len(usersTopTracks) == 0 {
		logger.Info("No top tracks found for user from Spotify", zap.String("userId", userId))
		return nil
	}
	logger.Info("Retrieved user top tracks from Spotify", zap.String("userId", userId), zap.Int("count", len(usersTopTracks)))

	dbTracks := convertSpotifyTracksToDBTracks(usersTopTracks)
	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}
	logger.Debug("Top tracks to save after deduplication", zap.String("userId", userId), zap.Int("tracksToSaveCount", len(tracksToSave)), zap.Int("originalDbTrackCount", len(dbTracks)))

	if len(tracksToSave) > 0 {
		err = db.SaveTracks(tracksToSave) // db.SaveTracks should have its own logging
		if err != nil {
			logger.Error("Error saving top tracks to DB",
				zap.String("userId", userId),
				zap.Int("tracksToSaveCount", len(tracksToSave)),
				zap.Error(err))
			return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
		}
	}

	// db.SaveUserTopTracks should have its own logging
	err = db.SaveUserTopTracks(userId, dbTracks)
	if err != nil {
		logger.Error("Error saving user-top track relations to DB",
			zap.String("userId", userId),
			zap.Int("dbTrackCount", len(dbTracks)),
			zap.Error(err))
		return fmt.Errorf("saving user-track relations: %w, tracks: %d", err, len(dbTracks))
	}

	logger.Info("Finished processing user top tracks", zap.String("userId", userId))
	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	logger.Info("Processing user saved tracks", zap.String("userId", userId))
	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
	if err != nil {
		logger.Error("Error getting user saved tracks from Spotify", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting saved tracks: %w", err)
	}
	if len(usersSavedTracks) == 0 {
		logger.Info("No saved tracks found for user from Spotify", zap.String("userId", userId))
		return nil
	}
	logger.Info("Retrieved user saved tracks from Spotify", zap.String("userId", userId), zap.Int("count", len(usersSavedTracks)))

	dbTracks := convertSpotifyTracksToDBTracks(usersSavedTracks)

	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}
	logger.Debug("Saved tracks to save after deduplication", zap.String("userId", userId), zap.Int("tracksToSaveCount", len(tracksToSave)), zap.Int("originalDbTrackCount", len(dbTracks)))

	if len(tracksToSave) > 0 {
		err = db.SaveTracks(tracksToSave) // db.SaveTracks should have its own logging
		if err != nil {
			logger.Error("Error saving saved tracks to DB",
				zap.String("userId", userId),
				zap.Int("tracksToSaveCount", len(tracksToSave)),
				zap.Error(err))
			return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
		}
	}

	// db.SaveUserSavedTracks should have its own logging
	err = db.SaveUserSavedTracks(userId, dbTracks)
	if err != nil {
		logger.Error("Error saving user-saved track relations to DB",
			zap.String("userId", userId),
			zap.Int("dbTrackCount", len(dbTracks)),
			zap.Error(err))
		return fmt.Errorf("saving user-track relations: %w, tracks: %d", err, len(dbTracks))
	}

	logger.Info("Finished processing user saved tracks", zap.String("userId", userId))
	return nil
}
