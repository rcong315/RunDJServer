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

func SaveTracks(userId string, tracks *[]Track, source string) error {
	log.Printf("Saving %d tracks", len(*tracks))
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
			*track.AudioFeatures,
		}
	})

	if err != nil {
		return fmt.Errorf("error saving tracks: %v", err)
	}

	var userTrackRelations []UserTrackRelation
	for _, track := range *tracks {
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

func GetTracksByBPM(userId string, min int, max int) (*[]Track, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	result, err := db.Exec(context.Background(), "SELECT * FROM track")
	if err != nil {
		return nil, fmt.Errorf("database execution error: %v", err)
	}

	return result, nil
}

// func SaveAlbums(userId string, albums []Album) error {
// 	return batchAndSave(userId, albums, InsertAlbumQuery, func(userId string, item any) []any {
// 		album := item.(Album)
// 		return []any{
// 			album.AlbumId,
// 			userId,
// 			album.Name,
// 			album.ArtistIds,
// 			album.Genres,
// 			album.Popularity,
// 			album.AlbumType,
// 			album.TotalTracks,
// 			album.ReleaseDate,
// 			album.AvailableMarkets,
// 			album.ImageURLs,
// 		}
// 	})
// }

// func SaveArtists(userId string, artists []Artist) error {
// 	return batchAndSave(userId, artists, InsertArtistQuery, func(userId string, item any) []any {
// 		artist := item.(Artist)
// 		return []any{
// 			artist.ArtistId,
// 			userId,
// 			artist.Name,
// 			artist.Genres,
// 			artist.Popularity,
// 			artist.Followers,
// 			artist.ImageURLs,
// 		}
// 	})
// }

// func SavePlaylists(userId string, playlists []Playlist) error {
// 	return batchAndSave(userId, playlists, InsertPlaylistQuery, func(userId string, item any) []any {
// 		playlist := item.(Playlist)
// 		return []any{
// 			playlist.PlaylistId,
// 			userId,
// 			playlist.OwnerId,
// 			playlist.Name,
// 			playlist.Description,
// 			playlist.Public,
// 			playlist.ImageURLs,
// 		}
// 	})
// }
