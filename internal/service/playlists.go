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

	var allDbTracks []*db.Track
	var mu sync.Mutex
	
	err := spotify.GetPlaylistsTracksStreaming(token, playlistId, func(tracks []*spotify.Track) error {
		dbTracks := convertSpotifyTracksToDBTracks(tracks)
		
		var tracksToSave []*db.Track
		for _, track := range dbTracks {
			if !tracker.CheckAndMark("track", track.TrackId) {
				tracksToSave = append(tracksToSave, track)
			}
		}
		
		if len(tracksToSave) > 0 {
			if err := db.SaveTracks(tracksToSave); err != nil {
				logger.Error("Error saving tracks batch in SavePlaylistTracksJob",
					zap.String("playlistId", playlistId),
					zap.Int("tracksToSaveCount", len(tracksToSave)),
					zap.Error(err))
				return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
			}
		}
		
		mu.Lock()
		allDbTracks = append(allDbTracks, dbTracks...)
		mu.Unlock()
		
		logger.Debug("Processed batch of playlist tracks", 
			zap.String("playlistId", playlistId), 
			zap.Int("batchSize", len(dbTracks)),
			zap.Int("savedCount", len(tracksToSave)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting tracks for playlist in job", zap.String("playlistId", playlistId), zap.Error(err))
		return fmt.Errorf("getting tracks for playlist %s: %w", playlistId, err)
	}
	
	if len(allDbTracks) == 0 {
		logger.Debug("No tracks found for playlist in job", zap.String("playlistId", playlistId))
		return nil
	}

	err = db.SavePlaylistTracks(playlistId, allDbTracks)
	if err != nil {
		logger.Error("Error saving playlist-track associations in job",
			zap.String("playlistId", playlistId),
			zap.Int("dbTrackCount", len(allDbTracks)),
			zap.Error(err))
		return fmt.Errorf("saving playlist tracks: %w", err)
	}

	logger.Debug("Successfully executed SavePlaylistTracksJob", zap.String("playlistId", playlistId))
	return nil
}

func processPlaylists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Processing user playlists", zap.String("userId", userId))
	
	var allDbPlaylists []*db.Playlist
	var allSpotifyPlaylists []*spotify.Playlist
	var mu sync.Mutex
	submittedJobs := 0
	
	err := spotify.GetUsersPlaylistsStreaming(token, func(playlists []*spotify.Playlist) error {
		dbPlaylists := convertSpotifyPlaylistsToDBPlaylists(playlists)
		
		var playlistsToSave []*db.Playlist
		for _, playlist := range dbPlaylists {
			if !tracker.CheckAndMark("playlist", playlist.PlaylistId) {
				playlistsToSave = append(playlistsToSave, playlist)
			}
		}
		
		if len(playlistsToSave) > 0 {
			if err := db.SavePlaylists(playlistsToSave); err != nil {
				logger.Error("Error saving playlists batch to DB",
					zap.String("userId", userId),
					zap.Int("playlistsToSaveCount", len(playlistsToSave)),
					zap.Error(err))
				return fmt.Errorf("saving playlists: %w", err)
			}
		}
		
		// Submit jobs immediately
		for _, playlist := range playlists {
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
		
		mu.Lock()
		allDbPlaylists = append(allDbPlaylists, dbPlaylists...)
		allSpotifyPlaylists = append(allSpotifyPlaylists, playlists...)
		mu.Unlock()
		
		logger.Debug("Processed batch of playlists", 
			zap.String("userId", userId), 
			zap.Int("batchSize", len(playlists)),
			zap.Int("savedCount", len(playlistsToSave)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting user playlists", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting playlists: %w", err)
	}
	
	if len(allDbPlaylists) == 0 {
		logger.Debug("No playlists found for user", zap.String("userId", userId))
		return nil
	}

	err = db.SaveUserPlaylists(userId, allDbPlaylists)
	if err != nil {
		logger.Error("Error saving user-playlist relations during user processing",
			zap.String("userId", userId),
			zap.Int("dbPlaylistCount", len(allDbPlaylists)),
			zap.Error(err))
		return fmt.Errorf("saving user-playlist relations: %w", err)
	}
	
	logger.Debug("Submitted jobs to save playlist tracks", zap.String("userId", userId), zap.Int("submittedJobCount", submittedJobs))
	logger.Debug("Finished processing user playlists", zap.String("userId", userId))
	return nil
}
