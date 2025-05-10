package db

import (
	"fmt"
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
	err := batchAndSave(playlists, "insertPlaylist", func(item any, _ int) []any {
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

	return nil
}

func SaveUserPlaylists(userId string, playlists []*Playlist) error {
	err := batchAndSave(playlists, "insertUserPlaylist", func(item any, _ int) []any {
		playlist := item.(*Playlist)
		return []any{
			userId,
			playlist.PlaylistId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user playlists: %v", err)
	}

	return nil
}

func SavePlaylistTracks(playlistId string, tracks []*Track) error {
	err := batchAndSave(tracks, "insertPlaylistTrack", func(item any, _ int) []any {
		track := item.(*Track)
		return []any{
			playlistId,
			track.TrackId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving playlist tracks: %v", err)
	}

	return nil
}
