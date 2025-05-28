package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func processTopTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user top tracks", zap.String("userId", userId))
	
	var allDbTracks []*db.Track
	var mu sync.Mutex
	
	// Batch processor for efficient DB saves
	dbBatcher := spotify.NewBatchProcessor[*db.Track](500, func(tracks []*db.Track) error {
		if len(tracks) > 0 {
			if err := db.SaveTracks(tracks); err != nil {
				logger.Error("Error saving top tracks batch to DB",
					zap.String("userId", userId),
					zap.Int("batchSize", len(tracks)),
					zap.Error(err))
				return fmt.Errorf("saving tracks batch: %w", err)
			}
			logger.Debug("Saved batch of tracks to DB", 
				zap.String("userId", userId), 
				zap.Int("batchSize", len(tracks)))
		}
		return nil
	})
	
	err := spotify.GetUsersTopTracksStreaming(token, func(tracks []*spotify.Track) error {
		dbTracks := convertSpotifyTracksToDBTracks(tracks)
		
		for _, track := range dbTracks {
			if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
				if err := dbBatcher.Add(track); err != nil {
					return err
				}
			}
		}
		
		mu.Lock()
		allDbTracks = append(allDbTracks, dbTracks...)
		mu.Unlock()
		
		logger.Debug("Processed page of top tracks", 
			zap.String("userId", userId), 
			zap.Int("pageSize", len(dbTracks)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting user top tracks from Spotify", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting top tracks: %w", err)
	}
	
	// Flush any remaining tracks in the batch
	if err := dbBatcher.Flush(); err != nil {
		logger.Error("Error flushing final tracks batch", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("flushing tracks batch: %w", err)
	}
	
	if len(allDbTracks) == 0 {
		logger.Debug("No top tracks found for user from Spotify", zap.String("userId", userId))
		return nil
	}

	err = db.SaveUserTopTracks(userId, allDbTracks)
	if err != nil {
		logger.Error("Error saving user-top track relations to DB",
			zap.String("userId", userId),
			zap.Int("dbTrackCount", len(allDbTracks)),
			zap.Error(err))
		return fmt.Errorf("saving user-track relations: %w, tracks: %d", err, len(allDbTracks))
	}

	logger.Debug("Finished processing user top tracks", zap.String("userId", userId), zap.Int("totalCount", len(allDbTracks)))
	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user saved tracks", zap.String("userId", userId))
	
	var allDbTracks []*db.Track
	var mu sync.Mutex
	
	err := spotify.GetUsersSavedTracksStreaming(token, func(tracks []*spotify.Track) error {
		dbTracks := convertSpotifyTracksToDBTracks(tracks)
		
		var tracksToSave []*db.Track
		for _, track := range dbTracks {
			if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
				tracksToSave = append(tracksToSave, track)
			}
		}
		
		if len(tracksToSave) > 0 {
			if err := db.SaveTracks(tracksToSave); err != nil {
				logger.Error("Error saving saved tracks batch to DB",
					zap.String("userId", userId),
					zap.Int("tracksToSaveCount", len(tracksToSave)),
					zap.Error(err))
				return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
			}
		}
		
		mu.Lock()
		allDbTracks = append(allDbTracks, dbTracks...)
		mu.Unlock()
		
		logger.Debug("Processed batch of saved tracks", 
			zap.String("userId", userId), 
			zap.Int("batchSize", len(dbTracks)),
			zap.Int("savedCount", len(tracksToSave)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting user saved tracks from Spotify", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting saved tracks: %w", err)
	}
	
	if len(allDbTracks) == 0 {
		logger.Debug("No saved tracks found for user from Spotify", zap.String("userId", userId))
		return nil
	}

	err = db.SaveUserSavedTracks(userId, allDbTracks)
	if err != nil {
		logger.Error("Error saving user-saved track relations to DB",
			zap.String("userId", userId),
			zap.Int("dbTrackCount", len(allDbTracks)),
			zap.Error(err))
		return fmt.Errorf("saving user-track relations: %w, tracks: %d", err, len(allDbTracks))
	}

	logger.Debug("Finished processing user saved tracks", zap.String("userId", userId), zap.Int("totalCount", len(allDbTracks)))
	return nil
}
