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

func createAlbumBatcher(parentType string, parentId string, tracker *ProcessedTracker, saveRelation func(string, []*db.Album) error) *spotify.BatchProcessor[*spotify.Album] {
	return spotify.NewBatchProcessor(100, func(albums []*spotify.Album) error {
		dbAlbums := convertSpotifyAlbumsToDBAlbums(albums)
		var albumsToSave []*db.Album
		for _, album := range dbAlbums {
			if !tracker.CheckAndMark("album", album.AlbumId) {
				albumsToSave = append(albumsToSave, album)
			}
		}

		if len(albumsToSave) > 0 {
			if err := db.SaveAlbums(albumsToSave); err != nil {
				return fmt.Errorf("saving albums batch: %w", err)
			}
			logger.Debug("Saved batch of albums to DB",
				zap.String(parentType, parentId))
		}

		if err := saveRelation(parentId, dbAlbums); err != nil {
			return fmt.Errorf("saving album relation: %w", err)
		}

		logger.Debug("Saved album relations to DB",
			zap.String(parentType, parentId))
		return nil
	})
}

func (j *SaveAlbumTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	albumId := j.AlbumId

	logger.Debug("Executing SaveAlbumTracksJob",
		zap.String("albumId", albumId))

	trackBatcher := createTrackBatcher("album", albumId, tracker, db.SaveAlbumTracks)

	err := spotify.GetAlbumsTracks(albumId, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting tracks for album %s: %w", albumId, err)
	}

	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks for album %s: %w", albumId, err)
	}

	logger.Debug("Executed SaveAlbumTracksJob",
		zap.String("albumId", albumId))
	return nil
}

func processSavedAlbums(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user saved albums",
		zap.String("userId", userId))

	albumBatcher := createAlbumBatcher("user", userId, tracker, db.SaveUserSavedAlbums)

	err := spotify.GetUsersSavedAlbums(token, func(albums []*spotify.Album) error {
		for _, album := range albums {
			if err := albumBatcher.Add(album); err != nil {
				return fmt.Errorf("adding album to batch: %w", err)
			}
			pool.SubmitWithStage(&SaveAlbumTracksJob{
				AlbumId: album.Id,
			}, jobWg, stage)
		}

		logger.Debug("Processed batch of saved albums",
			zap.String("userId", userId))
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting saved albums: %w", err)
	}

	if err := albumBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining albums: %w", err)
	}

	logger.Debug("Processed user saved albums",
		zap.String("userId", userId))
	return nil
}
