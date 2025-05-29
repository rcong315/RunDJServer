package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func createTrackBatcher(parentType string, parentId string, tracker *ProcessedTracker, saveRelation func(string, []*db.Track) error) *spotify.BatchProcessor[*spotify.Track] {
	return spotify.NewBatchProcessor[*spotify.Track](100, func(tracks []*spotify.Track) error {
		dbTracks := convertSpotifyTracksToDBTracks(tracks)
		var tracksToSave []*db.Track
		for _, track := range dbTracks {
			if !tracker.CheckAndMark("track", track.TrackId) {
				tracksToSave = append(tracksToSave, track)
			}
		}

		if len(tracksToSave) > 0 {
			if err := db.SaveTracks(tracksToSave); err != nil {
				return fmt.Errorf("saving tracks batch: %w", err)
			}
			logger.Debug("Saved batch of tracks to DB",
				zap.String(parentType, parentId))
		}

		if err := saveRelation(parentId, dbTracks); err != nil {
			return fmt.Errorf("saving track relations: %w", err)
		}
		logger.Debug("Saved track relations to DB",
			zap.String(parentType, parentId))

		return nil
	})
}

func processTopTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user top tracks",
		zap.String("userId", userId))

	trackBatcher := createTrackBatcher("user", userId, tracker, db.SaveUserTopTracks)

	// TODO: fix ranking
	err := spotify.GetUsersTopTracks(token, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting top tracks: %w", err)
	}

	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks: %w", err)
	}

	logger.Debug("Finished processing user top tracks",
		zap.String("userId", userId))
	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user saved tracks",
		zap.String("userId", userId))

	trackBatcher := createTrackBatcher("user", userId, tracker, db.SaveUserSavedTracks)

	err := spotify.GetUsersSavedTracks(token, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting saved tracks: %w", err)
	}

	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks: %w", err)
	}

	logger.Debug("Finished processing user saved tracks",
		zap.String("userId", userId))
	return nil
}
