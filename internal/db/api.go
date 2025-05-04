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

	return nil
}

func SaveUserTrackRelations(userId string, tracks []*Track, source string) error {
	if len(tracks) == 0 {
		return nil
	}

	err := batchAndSave(tracks, "insertUserTrackRelation", func(item any) []any {
		track := item.(*Track)
		return []any{
			userId,
			track.TrackId,
			[]string{source},
		}
	})
	if err != nil {
		return err
	}

	return nil
}

func SaveUserPlaylistRelations(userId string, playlists []*Playlist, source string) error {
	if len(playlists) == 0 {
		return nil
	}

	err := batchAndSave(playlists, "insertUserPlaylistRelation", func(item any) []any {
		playlist := item.(*Playlist)
		return []any{
			userId,
			playlist.PlaylistId,
			[]string{source},
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user playlist relations: %v", err)
	}

	return nil
}

func SaveUserArtistRelations(userId string, artists []*Artist, source string) error {
	if len(artists) == 0 {
		return nil
	}

	err := batchAndSave(artists, "insertUserArtistRelation", func(item any) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
			[]string{source},
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user artist relations: %v", err)
	}

	return nil
}

func SaveUserAlbumRelations(userId string, albums []*Album, source string) error {
	if len(albums) == 0 {
		return nil
	}

	err := batchAndSave(albums, "insertUserAlbumRelation", func(item any) []any {
		album := item.(*Album)
		return []any{
			userId,
			album.AlbumId,
			[]string{source},
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user album relations: %v", err)
	}

	return nil
}

func SaveTrackPlaylistRelations(playlistId string, tracks []*Track, source string) error {
	if len(tracks) == 0 {
		return nil
	}

	err := batchAndSave(tracks, "insertTrackPlaylistRelation", func(item any) []any {
		track := item.(*Track)
		return []any{
			playlistId,
			track.TrackId,
			[]string{source},
		}
	})
	if err != nil {
		return fmt.Errorf("error saving track-playlist relations: %v", err)
	}

	return nil
}

func GetTracksByBPM(userId string, min float64, max float64) ([]string, error) {
	rows, err := executeSelect("selectTracksByBPM", userId, min, max)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %v", err)
	}
	defer rows.Close()

	var tracks []string
	for rows.Next() {
		var track string
		err := rows.Scan(&track)
		if err != nil {
			return nil, fmt.Errorf("error scanning track: %v", err)
		}
		tracks = append(tracks, track)
	}

	// TODO: Shuffle tracks

	log.Printf("Found %d tracks for user %s with BPM between %f and %f", len(tracks), userId, min, max)
	return tracks, nil
}
