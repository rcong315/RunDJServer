package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type SaveAlbumTracksJob struct {
	AlbumId string
}

func (j *SaveAlbumTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	albumId := j.AlbumId

	var allDbTracks []*db.Track
	var mu sync.Mutex
	
	err := spotify.GetAlbumsTracksStreaming(albumId, func(tracks []*spotify.Track) error {
		dbTracks := convertSpotifyTracksToDBTracks(tracks)
		
		var tracksToSave []*db.Track
		for _, track := range dbTracks {
			if !tracker.CheckAndMark("track", track.TrackId) {
				tracksToSave = append(tracksToSave, track)
			}
		}
		
		if len(tracksToSave) > 0 {
			if err := db.SaveTracks(tracksToSave); err != nil {
				return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
			}
		}
		
		mu.Lock()
		allDbTracks = append(allDbTracks, dbTracks...)
		mu.Unlock()
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("getting tracks for album %s: %w", albumId, err)
	}
	
	if len(allDbTracks) == 0 {
		return nil
	}

	err = db.SaveAlbumTracks(albumId, allDbTracks)
	if err != nil {
		return fmt.Errorf("saving album tracks: %w", err)
	}

	return nil
}

func processSavedAlbums(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Getting user's saved albums", zap.String("userId", userId))
	
	var allDbAlbums []*db.Album
	var allSpotifyAlbums []*spotify.Album
	var mu sync.Mutex
	
	err := spotify.GetUsersSavedAlbumsStreaming(token, func(albums []*spotify.Album) error {
		dbAlbums := convertSpotifyAlbumsToDBAlbums(albums)
		
		var albumsToSave []*db.Album
		for _, album := range dbAlbums {
			if !tracker.CheckAndMark("album", album.AlbumId) {
				albumsToSave = append(albumsToSave, album)
			}
		}
		
		if len(albumsToSave) > 0 {
			if err := db.SaveAlbums(albumsToSave); err != nil {
				logger.Error("Error saving albums batch to DB",
					zap.String("userId", userId),
					zap.Int("albumsToSaveCount", len(albumsToSave)),
					zap.Error(err))
				return fmt.Errorf("saving albums: %w", err)
			}
		}
		
		// Submit jobs for album tracks immediately
		for _, album := range albums {
			pool.SubmitWithStage(&SaveAlbumTracksJob{
				AlbumId: album.Id,
			}, jobWg, stage)
		}
		
		mu.Lock()
		allDbAlbums = append(allDbAlbums, dbAlbums...)
		allSpotifyAlbums = append(allSpotifyAlbums, albums...)
		mu.Unlock()
		
		logger.Debug("Processed batch of saved albums", 
			zap.String("userId", userId), 
			zap.Int("batchSize", len(albums)),
			zap.Int("savedCount", len(albumsToSave)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting user's saved albums", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting saved albums: %w", err)
	}
	
	if len(allDbAlbums) == 0 {
		return nil
	}

	err = db.SaveUserSavedAlbums(userId, allDbAlbums)
	if err != nil {
		return fmt.Errorf("saving user-album relation: %w", err)
	}

	return nil
}
