package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// TODO: Update updated_at

// TODO: Save backups to filesystem

func SaveUser(user *User) error {
	sqlQuery, err := getQueryString("insertUser")
	if err != nil {
		return fmt.Errorf("error getting query string: %v", err)
	}

	db, err := getDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	_, err = db.Exec(context.Background(), sqlQuery,
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

	err := batchAndSave(tracks, "insertTrack", func(item any) []any {
		track := item.(*Track)

		// Marshal AudioFeatures to JSON
		var audioFeaturesJSON string
		if track.AudioFeatures != nil {
			audioFeaturesBytes, err := json.Marshal(track.AudioFeatures)
			if err != nil {
				log.Printf("error marshaling audio features for track %s: %v", track.TrackId, err)
			} else {
				audioFeaturesJSON = string(audioFeaturesBytes)
			}
		}

		return []any{
			track.TrackId,
			track.Name,
			track.ArtistIds,
			track.AlbumId,
			track.Popularity,
			track.DurationMS,
			track.AvailableMarkets,
			audioFeaturesJSON,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving tracks: %v", err)
	}
	log.Printf("Saved %d tracks for user %s", len(tracks), userId)

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
	err = batchAndSave(userTrackRelations, "insertUserTrackRelation", func(item any) []any {
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
	err = batchAndSave(userPlaylistRelations, "insertUserPlaylistRelation", func(item any) []any {
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
	err = batchAndSave(userArtistRelations, "insertUserArtistRelation", func(item any) []any {
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
	err = batchAndSave(userAlbumRelations, "insertUserAlbumRelation", func(item any) []any {
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

func GetTracksByBPM(userId string, min float64, max float64) ([]*Track, error) {
	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %v", err)
	}

	rows, err := db.Query(context.Background(), "selectTracksByBPM", userId, min, max)
	if err != nil {
		return nil, fmt.Errorf("error getting tracks by BPM: %v", err)
	}
	defer rows.Close()

	var tracks []*Track
	for rows.Next() {
		var track Track
		err := rows.Scan(
			&track.TrackId,
		)
		if err != nil {
			return nil, fmt.Errorf("error scanning track: %v", err)
		}
		tracks = append(tracks, &track)
	}

	return tracks, nil
}
