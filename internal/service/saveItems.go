package service

import (
	"errors"
	"fmt"
	"log"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func saveUser(user *spotify.User) {
	dbUser := convertSpotifyUserToDBUser(user)
	err := db.SaveUser(dbUser)
	if err != nil {
		log.Print(err)
	} else {
		log.Printf("User saved: %s", user.Id)
	}
}

// TODO: Use generics

func saveTracks(userId string, items any, source string) error {
	tracks, ok := items.([]*spotify.Track)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Track")
	}
	if len(tracks) == 0 {
		log.Printf("No tracks to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d tracks for user %s from source %s", len(tracks), userId, source)
	var trackData []*db.Track
	for _, track := range tracks {
		if track != nil && track.Id != "" {
			dbTrack := convertSpotifyTrackToDBTrack(track)
			trackData = append(trackData, dbTrack)
		}
	}

	if len(trackData) == 0 {
		log.Printf("No valid tracks found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SaveTracks(userId, trackData, source)
	if err != nil {
		log.Printf("Error saving %d tracks for user %s from source %s: %v", len(trackData), userId, source, err)
		return fmt.Errorf("saving %d tracks from %s: %w", len(trackData), source, err)
	}
	log.Printf("Saved %d tracks for user %s from source %s", len(trackData), userId, source)
	return nil
}

func savePlaylists(userId string, items any, source string) error {
	playlists, ok := items.([]*spotify.Playlist)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Playlist")
	}
	if len(playlists) == 0 {
		log.Printf("No playlists to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d playlists for user %s from source %s", len(playlists), userId, source)
	var playlistData []*db.Playlist
	for _, playlist := range playlists {
		if playlist != nil && playlist.Id != "" {
			dbPlaylist := convertSpotifyPlaylistToDBPlaylist(playlist)
			playlistData = append(playlistData, dbPlaylist)
		}
	}

	if len(playlistData) == 0 {
		log.Printf("No valid playlists found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SavePlaylists(userId, playlistData, source)
	if err != nil {
		log.Printf("Error saving %d playlists for user %s from source %s: %v", len(playlistData), userId, source, err)
		return fmt.Errorf("saving %d playlists from %s: %w", len(playlistData), source, err)
	}
	log.Printf("Saved %d playlists for user %s from source %s", len(playlistData), userId, source)
	return nil
}

func saveArtists(userId string, items any, source string) error {
	artists, ok := items.([]*spotify.Artist)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Artist")
	}
	if len(artists) == 0 {
		log.Printf("No artists to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d artists for user %s from source %s", len(artists), userId, source)
	var artistData []*db.Artist
	for _, artist := range artists {
		if artist != nil && artist.Id != "" {
			dbPlaylist := convertSpotifyArtistToDBArtist(artist)
			artistData = append(artistData, dbPlaylist)
		}
	}

	if len(artistData) == 0 {
		log.Printf("No valid artists found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SaveArtists(userId, artistData, source)
	if err != nil {
		log.Printf("Error saving %d artists for user %s from source %s: %v", len(artistData), userId, source, err)
		return fmt.Errorf("saving %d artists from %s: %w", len(artistData), source, err)
	}
	log.Printf("Saved %d artists for user %s from source %s", len(artistData), userId, source)
	return nil
}

func saveAlbums(userId string, items any, source string) error {
	albums, ok := items.([]*spotify.Album)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Album")
	}
	if len(albums) == 0 {
		log.Printf("No albums to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d albums for user %s from source %s", len(albums), userId, source)
	var albumData []*db.Album
	for _, album := range albums {
		if album != nil && album.Id != "" {
			dbAlbum := convertSpotifyAlbumToDBAlbum(album)
			albumData = append(albumData, dbAlbum)
		}
	}

	if len(albumData) == 0 {
		log.Printf("No valid albums found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SaveAlbums(userId, albumData, source)
	if err != nil {
		log.Printf("Error saving %d albums for user %s from source %s: %v", len(albumData), userId, source, err)
		return fmt.Errorf("saving %d albums from %s: %w", len(albumData), source, err)
	}
	log.Printf("Saved %d albums for user %s from source %s", len(albumData), userId, source)
	return nil
}
