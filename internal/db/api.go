package db

import (
	"context"
	"fmt"
)

func SaveUser(user User) error {
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	_, err = db.Exec(context.Background(), InsertUserQuery,
		user.UserId,
		user.Email,
		user.DisplayName,
		user.Country,
		user.Followers,
		user.Product,
		user.ExplicitFilterEnabled,
		user.ImageURLs,
	)
	if err != nil {
		return fmt.Errorf("error creating user record: %v", err)
	}

	return nil
}

func SaveTracks(userId string, tracks []Track) error {
	return batchAndSave(userId, tracks, InsertTrackQuery, func(userId string, item any) []any {
		track := item.(Track)
		return []any{
			track.TrackId,
			userId,
			track.Name,
			track.ArtistIds,
			track.AlbumId,
			track.Popularity,
			track.DurationMS,
			track.AvailableMarkets,
			track.AudioFeatures,
		}
	})
}

func SaveAlbums(userId string, albums []Album) error {
	return batchAndSave(userId, albums, InsertAlbumQuery, func(userId string, item any) []any {
		album := item.(Album)
		return []any{
			album.AlbumId,
			userId,
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
}

func SaveArtists(userId string, artists []Artist) error {
	return batchAndSave(userId, artists, InsertArtistQuery, func(userId string, item any) []any {
		artist := item.(Artist)
		return []any{
			artist.ArtistId,
			userId,
			artist.Name,
			artist.Genres,
			artist.Popularity,
			artist.Followers,
			artist.ImageURLs,
		}
	})
}

func SavePlaylists(userId string, playlists []Playlist) error {
	return batchAndSave(userId, playlists, InsertPlaylistQuery, func(userId string, item any) []any {
		playlist := item.(Playlist)
		return []any{
			playlist.PlaylistId,
			userId,
			playlist.OwnerId,
			playlist.Name,
			playlist.Description,
			playlist.Public,
			playlist.ImageURLs,
		}
	})
}
