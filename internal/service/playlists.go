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

func (j *SavePlaylistTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	token := j.Token // Token should not be logged directly for security
	playlistId := j.PlaylistID
	logger.Debug("Executing SavePlaylistTracksJob", zap.String("playlistId", playlistId))

	playlistTracks, err := spotify.GetPlaylistsTracks(token, playlistId)
	if err != nil {
		logger.Error("Error getting tracks for playlist in job", zap.String("playlistId", playlistId), zap.Error(err))
		return fmt.Errorf("getting tracks for playlist %s: %w", playlistId, err)
	}
	if len(playlistTracks) == 0 {
		logger.Debug("No tracks found for playlist in job", zap.String("playlistId", playlistId))
		return nil
	}
	logger.Debug("Retrieved tracks for playlist", zap.String("playlistId", playlistId), zap.Int("trackCount", len(playlistTracks)))

	dbTracks := convertSpotifyTracksToDBTracks(playlistTracks)
	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}
	logger.Debug("Tracks to save after deduplication", zap.String("playlistId", playlistId), zap.Int("tracksToSaveCount", len(tracksToSave)))

	if len(tracksToSave) > 0 {
		err = db.SaveTracks(tracksToSave) // db.SaveTracks should have its own logging
		if err != nil {
			logger.Error("Error saving tracks in SavePlaylistTracksJob",
				zap.String("playlistId", playlistId),
				zap.Int("tracksToSaveCount", len(tracksToSave)),
				zap.Error(err))
			return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
		}
	}

	// SavePlaylistTracks might attempt to save all original dbTracks, not just tracksToSave.
	// Assuming db.SavePlaylistTracks handles its own logging for success/failure.
	err = db.SavePlaylistTracks(playlistId, dbTracks)
	if err != nil {
		logger.Error("Error saving playlist-track associations in job",
			zap.String("playlistId", playlistId),
			zap.Int("dbTrackCount", len(dbTracks)),
			zap.Error(err))
		return fmt.Errorf("saving playlist tracks: %w", err)
	}

	logger.Debug("Successfully executed SavePlaylistTracksJob", zap.String("playlistId", playlistId))
	return nil
}

func processPlaylists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user playlists", zap.String("userId", userId))
	usersPlaylists, err := spotify.GetUsersPlaylists(token)
	if err != nil {
		logger.Error("Error getting user playlists", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting playlists: %w", err)
	}
	if len(usersPlaylists) == 0 {
		logger.Debug("No playlists found for user", zap.String("userId", userId))
		return nil
	}
	logger.Debug("Retrieved user playlists", zap.String("userId", userId), zap.Int("playlistCount", len(usersPlaylists)))

	dbPlaylists := convertSpotifyPlaylistsToDBPlaylists(usersPlaylists)
	var playlistsToSave []*db.Playlist
	for _, playlist := range dbPlaylists {
		if !tracker.CheckAndMark("playlist", playlist.PlaylistId) {
			playlistsToSave = append(playlistsToSave, playlist)
		}
	}
	logger.Debug("Playlists to save after deduplication", zap.String("userId", userId), zap.Int("playlistsToSaveCount", len(playlistsToSave)))

	if len(playlistsToSave) > 0 {
		err = db.SavePlaylists(playlistsToSave) // db.SavePlaylists should have its own logging
		if err != nil {
			logger.Error("Error saving playlists during user processing",
				zap.String("userId", userId),
				zap.Int("playlistsToSaveCount", len(playlistsToSave)),
				zap.Error(err))
			return fmt.Errorf("saving playlists: %w", err)
		}
	}

	// db.SaveUserPlaylists should have its own logging
	err = db.SaveUserPlaylists(userId, dbPlaylists)
	if err != nil {
		logger.Error("Error saving user-playlist relations during user processing",
			zap.String("userId", userId),
			zap.Int("dbPlaylistCount", len(dbPlaylists)),
			zap.Error(err))
		return fmt.Errorf("saving user-playlist relations: %w", err)
	}

	submittedJobs := 0
	for _, playlist := range usersPlaylists {
		if playlist != nil && playlist.Id != "" {
			pool.SubmitWithStage(&SavePlaylistTracksJob{
				Token:      token,
				PlaylistID: playlist.Id,
			}, jobWg, stage)
			submittedJobs++
		} else {
			logger.Warn("Encountered nil or empty ID playlist during job submission", zap.String("userId", userId))
		}
	}
	logger.Debug("Submitted jobs to save playlist tracks", zap.String("userId", userId), zap.Int("submittedJobCount", submittedJobs))

	logger.Debug("Finished processing user playlists", zap.String("userId", userId))
	return nil
}
