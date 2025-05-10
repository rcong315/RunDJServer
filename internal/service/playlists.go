package service

import (
	"fmt"
	"sync"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type SavePlaylistTracksJob struct {
	Token      string
	PlaylistID string
}

func (j *SavePlaylistTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker) error {
	token := j.Token
	playlistId := j.PlaylistID

	playlistTracks, err := spotify.GetPlaylistsTracks(token, playlistId)
	if err != nil {
		return fmt.Errorf("getting tracks for playlist %s: %w", playlistId, err)
	}
	if len(playlistTracks) == 0 {
		return nil
	}

	dbTracks := convertSpotifyTracksToDBTracks(playlistTracks)
	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}

	err = db.SaveTracks(tracksToSave)
	if err != nil {
		return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
	}

	err = db.SavePlaylistTracks(playlistId, dbTracks)
	if err != nil {
		return fmt.Errorf("saving playlist tracks: %w", err)
	}

	return nil
}

func processPlaylists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	usersPlaylists, err := spotify.GetUsersPlaylists(token)
	if err != nil {
		return fmt.Errorf("getting playlists: %w", err)
	}
	if len(usersPlaylists) == 0 {
		return nil
	}

	dbPlaylists := convertSpotifyPlaylistsToDBPlaylists(usersPlaylists)
	var playlistsToSave []*db.Playlist
	for _, playlist := range dbPlaylists {
		if !tracker.CheckAndMark("playlist", playlist.PlaylistId) {
			playlistsToSave = append(playlistsToSave, playlist)
		}
	}

	err = db.SavePlaylists(playlistsToSave)
	if err != nil {
		return fmt.Errorf("saving playlists: %w", err)
	}

	err = db.SaveUserPlaylists(userId, dbPlaylists)
	if err != nil {
		return fmt.Errorf("saving user-playlist relations: %w", err)
	}

	for _, playlist := range usersPlaylists {
		if playlist != nil && playlist.Id != "" {
			pool.Submit(&SavePlaylistTracksJob{
				Token:      token,
				PlaylistID: playlist.Id,
			}, jobWg)
		}
	}

	return nil
}
