package db

import (
	"context"
	"fmt"
	"log"
)

func SaveUser(user *User) error {
	db, err := getDB()
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
		user.ImageURLs,
	)
	if err != nil {
		return fmt.Errorf("error creating user record: %v", err)
	}

	return nil
}

// TODO: Common function for saving relations

func SaveTracks(userId string, tracks []*Track, source string) error {
	if len(tracks) == 0 {
		return nil
	}

	log.Printf("Saving %d tracks", len(tracks))
	err := batchAndSave(tracks, InsertTrackQuery, func(item any) []any {
		track := item.(Track)
		return []any{
			track.TrackId,
			track.Name,
			track.ArtistIds,
			track.AlbumId,
			track.Popularity,
			track.DurationMS,
			track.AvailableMarkets,
			track.AudioFeatures,
			source,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving tracks: %v", err)
	}

	var userTrackRelations []UserTrackRelation
	for _, track := range tracks {
		userTrackRelation := UserTrackRelation{
			UserId:  userId,
			TrackId: track.TrackId,
			Sources: []string{source},
		}
		userTrackRelations = append(userTrackRelations, userTrackRelation)
	}
	log.Printf("Saving %d user track relations for user %s", len(userTrackRelations), userId)
	err = batchAndSave(userTrackRelations, InsertUserTrackRelationQuery, func(item any) []any {
		userTrackRelation := item.(UserTrackRelation)
		return []any{
			userTrackRelation.UserId,
			userTrackRelation.TrackId,
			userTrackRelation.Sources,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user track relations: %v", err)
	}

	return nil
}

func SavePlaylists(userId string, playlists []*Playlist, source string) error {
	if len(playlists) == 0 {
		return nil
	}

	err := batchAndSave(playlists, InsertPlaylistQuery, func(item any) []any {
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
	if err != nil {
		return fmt.Errorf("error saving playlists: %v", err)
	}

	var userPlaylistRelations []UserPlaylistRelation
	for _, playlist := range playlists {
		userPlaylistRelation := UserPlaylistRelation{
			UserId:     userId,
			PlaylistId: playlist.PlaylistId,
			Sources:    []string{source},
		}
		userPlaylistRelations = append(userPlaylistRelations, userPlaylistRelation)
	}
	log.Printf("Saving %d user playlist relations for user %s", len(userPlaylistRelations), userId)
	err = batchAndSave(userPlaylistRelations, InsertUserPlaylistRelationQuery, func(item any) []any {
		userPlaylistRelation := item.(UserPlaylistRelation)
		return []any{
			userPlaylistRelation.UserId,
			userPlaylistRelation.PlaylistId,
			userPlaylistRelation.Sources,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user playlist relations: %v", err)
	}

	return nil
}

func SaveArtists(userId string, artists []*Artist, source string) error {
	if len(artists) == 0 {
		return nil
	}

	err := batchAndSave(artists, InsertArtistQuery, func(item any) []any {
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

	if err != nil {
		return fmt.Errorf("error saving artists: %v", err)
	}

	var userArtistRelations []UserArtistRelation
	for _, artist := range artists {
		userArtistRelation := UserArtistRelation{
			UserId:   userId,
			ArtistId: artist.ArtistId,
			Sources:  []string{source},
		}
		userArtistRelations = append(userArtistRelations, userArtistRelation)
	}
	log.Printf("Saving %d user artist relations for user %s", len(userArtistRelations), userId)
	err = batchAndSave(userArtistRelations, InsertUserArtistRelationQuery, func(item any) []any {
		userArtistRelation := item.(UserArtistRelation)
		return []any{
			userArtistRelation.UserId,
			userArtistRelation.ArtistId,
			userArtistRelation.Sources,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user artist relations: %v", err)
	}

	return nil
}

func SaveAlbums(userId string, albums []*Album, source string) error {
	if len(albums) == 0 {
		return nil
	}

	log.Printf("Saving %d albums", len(albums))
	err := batchAndSave(albums, InsertAlbumQuery, func(item any) []any {
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
	if err != nil {
		return fmt.Errorf("error saving albums: %v", err)
	}

	var userAlbumRelations []UserAlbumRelation
	for _, album := range albums {
		userAlbumRelation := UserAlbumRelation{
			UserId:  userId,
			AlbumId: album.AlbumId,
			Sources: []string{source},
		}
		userAlbumRelations = append(userAlbumRelations, userAlbumRelation)
	}
	log.Printf("Saving %d user album relations for user %s", len(userAlbumRelations), userId)
	err = batchAndSave(userAlbumRelations, InsertUserAlbumRelationQuery, func(item any) []any {
		userAlbumRelation := item.(UserAlbumRelation)
		return []any{
			userAlbumRelation.UserId,
			userAlbumRelation.AlbumId,
			userAlbumRelation.Sources,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user album relations: %v", err)
	}

	return nil
}

// TODO: Get tracks by BPM
func GetTracksByBPM(userId string, min int, max int) ([]*Track, error) {
	return nil, nil
}
