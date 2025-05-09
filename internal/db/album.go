package db

import (
	"fmt"
	"log"
)

func SaveAlbums(userId string, albums []*Album, source string) error {
	if len(albums) == 0 {
		return nil
	}

	err := batchAndSave(albums, "insertAlbum", func(item any) []any {
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
	log.Printf("Saved %d albums for user %s", len(albums), userId)

	return nil
}

func SaveUserSavedAlbums(userId string, albums []*Album) error {
	if len(albums) == 0 {
		return nil
	}

	err := batchAndSave(albums, "insertUserSavedAlbum", func(item any) []any {
		album := item.(*Album)
		return []any{
			userId,
			album.AlbumId,
			0,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user saved albums: %v", err)
	}

	return nil
}
