package db

import (
	"fmt"
	"log"
)

func SaveArtists(userId string, artists []*Artist, source string) error {
	if len(artists) == 0 {
		return nil
	}

	err := batchAndSave(artists, "insertArtist", func(item any) []any {
		artist := item.(*Artist)
		return []any{
			artist.ArtistId,
			artist.Name,
			artist.Genres,
			artist.Popularity,
			artist.Followers,
			artist.ImageURLs,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving artists: %v", err)
	}
	log.Printf("Saved %d artists for user %s", len(artists), userId)

	return nil
}

func SaveUserTopArtists(userId string, artists []*Artist) error {
	if len(artists) == 0 {
		return nil
	}

	err := batchAndSave(artists, "insertUserTopArtist", func(item any) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
			0,
			//TODO: rank
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user top artists: %v", err)
	}

	return nil
}

func SaveUserFollowedArtists(userId string, artists []*Artist) error {
	if len(artists) == 0 {
		return nil
	}

	err := batchAndSave(artists, "insertUserFollowedArtist", func(item any) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
			0,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user followed artists: %v", err)
	}

	return nil
}
