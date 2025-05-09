package db

import (
	"fmt"
	"log"
)

func SavePlaylists(userId string, playlists []*Playlist, source string) error {
	if len(playlists) == 0 {
		return nil
	}

	err := batchAndSave(playlists, "insertPlaylist", func(item any) []any {
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
	log.Printf("Saved %d playlists for user %s", len(playlists), userId)

	return nil
}

func SaveUserPlaylists(userId string, playlists []*Playlist) error {
	if len(playlists) == 0 {
		return nil
	}

	err := batchAndSave(playlists, "insertUserPlaylist", func(item any) []any {
		playlist := item.(*Playlist)
		return []any{
			userId,
			playlist.PlaylistId,
			0,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user playlists: %v", err)
	}

	return nil
}

func SavePlaylistTracks(playlistId string, tracks []*Track) error {
	if len(tracks) == 0 {
		return nil
	}

	err := batchAndSave(tracks, "insertPlaylistTrack", func(item any) []any {
		track := item.(*Track)
		return []any{
			playlistId,
			track.TrackId,
			0,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving playlist tracks: %v", err)
	}

	return nil
}
