package db

import (
	"fmt"
)

type Artist struct {
	ArtistId   string   `json:"artist_id"`
	Name       string   `json:"name"`
	Genres     []string `json:"genres"`
	Popularity int      `json:"popularity"`
	Followers  int      `json:"followers"`
	ImageURLs  []string `json:"image_urls"`
}

func SaveArtists(artists []*Artist) error {
	err := batchAndSave(artists, "insertArtist", func(item any, _ int) []any {
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

	return nil
}

func SaveUserTopArtists(userId string, artists []*Artist) error {
	err := batchAndSave(artists, "insertUserTopArtist", func(item any, rank int) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
			rank,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user top artists: %v", err)
	}

	return nil
}

func SaveUserFollowedArtists(userId string, artists []*Artist) error {
	err := batchAndSave(artists, "insertUserFollowedArtist", func(item any, _ int) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user followed artists: %v", err)
	}

	return nil
}

func SaveArtistTopTracks(artistId string, tracks []*Track) error {
	err := batchAndSave(tracks, "insertArtistTopTrack", func(item any, rank int) []any {
		track := item.(*Track)
		return []any{
			artistId,
			track.TrackId,
			rank,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving artist top tracks: %v", err)
	}

	return nil
}

func SaveArtistAlbums(artistId string, albums []*Album) error {
	err := batchAndSave(albums, "insertArtistAlbum", func(item any, _ int) []any {
		album := item.(*Album)
		return []any{
			artistId,
			album.AlbumId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving artist albums: %v", err)
	}

	return nil
}
