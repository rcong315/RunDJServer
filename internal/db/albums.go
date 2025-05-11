package db

import (
	"fmt"
)

type Album struct {
	AlbumId          string   `json:"album_id"`
	Name             string   `json:"name"`
	ArtistIds        []string `json:"artist_ids"`
	Genres           []string `json:"genres"`
	Popularity       int      `json:"popularity"`
	AlbumType        string   `json:"album_type"`
	TotalTracks      int      `json:"total_tracks"`
	ReleaseDate      string   `json:"release_date"`
	AvailableMarkets []string `json:"available_markets"`
	ImageURLs        []string `json:"image_urls"`
}

func SaveAlbums(albums []*Album) error {
	err := batchAndSave(albums, "album", func(item any, _ int) []any {
		album := item.(*Album)
		return []any{
			album.AlbumId,
			album.Name,
			album.ArtistIds,
			album.Genres,
			album.Popularity,
			album.AlbumType,
			album.TotalTracks,
			album.ReleaseDate,
			album.AvailableMarkets,
			album.ImageURLs,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving albums: %v", err)
	}

	return nil
}

func SaveUserSavedAlbums(userId string, albums []*Album) error {
	err := batchAndSave(albums, "userSavedAlbum", func(item any, _ int) []any {
		album := item.(*Album)
		return []any{
			userId,
			album.AlbumId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user saved albums: %v", err)
	}

	return nil
}

func SaveAlbumTracks(albumId string, tracks []*Track) error {
	err := batchAndSave(tracks, "albumTrack", func(item any, _ int) []any {
		track := item.(*Track)
		return []any{
			albumId,
			track.TrackId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving album tracks: %v", err)
	}

	return nil
}
