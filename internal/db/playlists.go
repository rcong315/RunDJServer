package db

import (
	"fmt"

	"go.uber.org/zap"
)

type Playlist struct {
	PlaylistId  string   `json:"playlist_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	OwnerId     string   `json:"owner_id"`
	Public      bool     `json:"public"`
	Followers   int      `json:"followers"`
	ImageURLs   []string `json:"image_urls"`
}

// TODO: Delete deleted playlists
func SavePlaylists(playlists []*Playlist) error {
	if len(playlists) == 0 {
		logger.Debug("SavePlaylists: No playlists to save.")
		return nil
	}
	logger.Debug("Attempting to save playlists", zap.Int("count", len(playlists)))

	err := batchAndSave(playlists, "playlist", func(item any, _ int) []any {
		playlist := item.(*Playlist)
		return []any{
			playlist.PlaylistId,
			playlist.Name,
			playlist.Description,
			playlist.OwnerId,
			playlist.Public,
			playlist.Followers,
			playlist.ImageURLs,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving playlists: %v", err)
	}

	logger.Debug("Successfully saved playlists batch", zap.Int("count", len(playlists)))
	return nil
}

func SaveUserPlaylists(userId string, playlists []*Playlist) error {
	if len(playlists) == 0 {
		logger.Debug("SaveUserPlaylists: No playlists to associate for user.", zap.String("userId", userId))
		return nil
	}
	logger.Debug("Attempting to save user-playlist associations", zap.String("userId", userId), zap.Int("count", len(playlists)))

	err := batchAndSave(playlists, "userPlaylist", func(item any, _ int) []any {
		playlist := item.(*Playlist)
		return []any{
			userId,
			playlist.PlaylistId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user playlists: %v", err)
	}

	logger.Debug("Successfully saved user-playlist associations batch", zap.String("userId", userId), zap.Int("count", len(playlists)))
	return nil
}

func SavePlaylistTracks(playlistId string, tracks []*Track) error {
	if len(tracks) == 0 {
		logger.Debug("SavePlaylistTracks: No tracks to associate with playlist.", zap.String("playlistId", playlistId))
		return nil
	}
	logger.Debug("Attempting to save playlist-track associations", zap.String("playlistId", playlistId), zap.Int("trackCount", len(tracks)))

	err := batchAndSave(tracks, "playlistTrack", func(item any, _ int) []any {
		track := item.(*Track)
		return []any{
			playlistId,
			track.TrackId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving playlist tracks: %v", err)
	}

	logger.Debug("Successfully saved playlist-track associations batch", zap.String("playlistId", playlistId), zap.Int("trackCount", len(tracks)))
	return nil
}
