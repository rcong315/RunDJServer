package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func createTrackBatcher(userId string, tracker *ProcessedTracker) *spotify.BatchProcessor[*spotify.Track] {
	return spotify.NewBatchProcessor[*spotify.Track](500, func(tracks []*spotify.Track) error {
		dbTracks := convertSpotifyTracksToDBTracks(tracks)
		var tracksToSave []*db.Track
		for _, track := range dbTracks {
			if !tracker.CheckAndMark("track", track.TrackId) {
				tracksToSave = append(tracksToSave, track)
			}
		}

		if err := db.SaveTracks(tracksToSave); err != nil {
			logger.Error("Error saving top tracks batch to DB",
				zap.String("userId", userId),
				zap.Error(err))
			return fmt.Errorf("saving tracks batch: %w", err)
		}
		logger.Debug("Saved batch of tracks to DB",
			zap.String("userId", userId))

		if err := db.SaveUserTopTracks(userId, dbTracks); err != nil {
			logger.Error("Error saving user-top track relations to DB",
				zap.String("userId", userId),
				zap.Error(err))
			return fmt.Errorf("saving user-track relations: %w", err)
		}
		logger.Debug("Saved user-top track relations to DB",
			zap.String("userId", userId))

		return nil
	})
}

func processTopTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user top tracks",
		zap.String("userId", userId))

	dbBatcher := createTrackBatcher(userId, tracker)

	err := spotify.GetUsersTopTracks(token, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			dbBatcher.Add(track)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting top tracks: %w", err)
	}

	if err := dbBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks: %w", err)
	}

	logger.Debug("Finished processing user top tracks",
		zap.String("userId", userId))
	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user saved tracks",
		zap.String("userId", userId))

	dbBatcher := createTrackBatcher(userId, tracker)

	err := spotify.GetUsersSavedTracks(token, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			dbBatcher.Add(track)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting saved tracks: %w", err)
	}

	if err := dbBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks: %w", err)
	}

	logger.Debug("Finished processing user saved tracks",
		zap.String("userId", userId))
	return nil
}
