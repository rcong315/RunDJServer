package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type SavePlaylistTracksJob struct {
	Token      string
	PlaylistID string
}

func createPlaylistBatcher(userId string, tracker *ProcessedTracker) *spotify.BatchProcessor[*spotify.Playlist] {
	return spotify.NewBatchProcessor[*spotify.Playlist](100, func(playlists []*spotify.Playlist) error {
		dbPlaylists := convertSpotifyPlaylistsToDBPlaylists(playlists)
		var playlistsToSave []*db.Playlist
		for _, playlist := range dbPlaylists {
			if !tracker.CheckAndMark("playlist", playlist.PlaylistId) {
				playlistsToSave = append(playlistsToSave, playlist)
			}
		}

		if len(playlistsToSave) > 0 {
			if err := db.SavePlaylists(playlistsToSave); err != nil {
				return fmt.Errorf("saving playlists batch: %w", err)
			}
			logger.Debug("Saved batch of playlists to DB")
		}

		if err := db.SaveUserPlaylists(userId, dbPlaylists); err != nil {
			return fmt.Errorf("saving user-playlist relations: %w", err)
		}
		logger.Debug("Saved user-playlist relations to DB", zap.String("userId", userId))

		return nil
	})

}

func (j *SavePlaylistTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	token := j.Token
	playlistId := j.PlaylistID

	logger.Debug("Executing SavePlaylistTracksJob", zap.String("playlistId", playlistId))

	trackBatcher := createTrackBatcher("playlist", playlistId, tracker, db.SavePlaylistTracks)

	err := spotify.GetPlaylistsTracksStreaming(token, playlistId, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting tracks for playlist %s: %w", playlistId, err)
	}

	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks for playlist %s: %w", playlistId, err)
	}

	logger.Debug("Successfully executed SavePlaylistTracksJob", zap.String("playlistId", playlistId))
	return nil
}

func processPlaylists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user playlists",
		zap.String("userId", userId))

	playlistBatcher := createPlaylistBatcher(userId, tracker)

	err := spotify.GetUsersPlaylistsStreaming(token, func(playlists []*spotify.Playlist) error {
		for _, playlist := range playlists {
			playlistBatcher.Add(playlist)
			pool.SubmitWithStage(&SavePlaylistTracksJob{
				Token:      token,
				PlaylistID: playlist.Id,
			}, jobWg, stage)
		}

		logger.Debug("Processed batch of playlists",
			zap.String("userId", userId))
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting playlists: %w", err)
	}

	if err := playlistBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining playlists: %w", err)
	}

	logger.Debug("Finished processing user playlists", zap.String("userId", userId))
	return nil
}
