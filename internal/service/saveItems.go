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

func saveTracks(userId string, items any, source string, tracker *ProcessedTracker) error {
	tracks, ok := items.([]*spotify.Track)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Track")
	}
	if len(tracks) == 0 {
		log.Printf("No tracks to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d tracks for user %s from source %s", len(tracks), userId, source)
	dbTracks := convertSpotifyTracksToDBTracks(tracks)

	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if track != nil && track.TrackId != "" && !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}

	err := db.SaveTracks(userId, tracksToSave, source)
	if err != nil {
		return fmt.Errorf("saving %d tracks from %s: %w", len(tracksToSave), source, err)
	}

	err = db.SaveUserTrackRelations(userId, dbTracks, source)
	if err != nil {
		return fmt.Errorf("saving %d user-track relations from %s: %w", len(dbTracks), source, err)
	}

	log.Printf("Saved %d tracks for user %s from source %s", len(dbTracks), userId, source)
	return nil
}

func savePlaylists(userId string, items any, source string, tracker *ProcessedTracker) error {
	playlists, ok := items.([]*spotify.Playlist)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Playlist")
	}
	if len(playlists) == 0 {
		log.Printf("No playlists to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d playlists for user %s from source %s", len(playlists), userId, source)
	dbPlaylists := convertSpotifyPlaylistsToDBPlaylists(playlists)

	var playlistsToSave []*db.Playlist
	for _, playlist := range dbPlaylists {
		if playlist != nil && playlist.PlaylistId != "" && !tracker.CheckAndMark("playlist", playlist.PlaylistId) {
			playlistsToSave = append(playlistsToSave, playlist)
		}
	}

	err := db.SavePlaylists(userId, playlistsToSave, source)
	if err != nil {
		return fmt.Errorf("saving %d playlists from %s: %w", len(playlistsToSave), source, err)
	}

	err = db.SaveUserPlaylistRelations(userId, dbPlaylists, source)
	if err != nil {
		return fmt.Errorf("saving %d user-playlist relations from %s: %w", len(dbPlaylists), source, err)
	}

	log.Printf("Saved %d playlists for user %s from source %s", len(dbPlaylists), userId, source)
	return nil
}

func saveArtists(userId string, items any, source string, tracker *ProcessedTracker) error {
	artists, ok := items.([]*spotify.Artist)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Artist")
	}
	if len(artists) == 0 {
		log.Printf("No artists to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d artists for user %s from source %s", len(artists), userId, source)
	dbArtists := convertSpotifyArtistsToDBArtists(artists)

	var artistsToSave []*db.Artist
	for _, artist := range dbArtists {
		if artist != nil && artist.ArtistId != "" && !tracker.CheckAndMark("artist", artist.ArtistId) {
			artistsToSave = append(artistsToSave, artist)
		}
	}

	err := db.SaveArtists(userId, artistsToSave, source)
	if err != nil {
		return fmt.Errorf("saving %d artists from %s: %w", len(artistsToSave), source, err)
	}

	err = db.SaveUserArtistRelations(userId, dbArtists, source)
	if err != nil {
		return fmt.Errorf("saving %d user-artist relations from %s: %w", len(dbArtists), source, err)
	}

	log.Printf("Saved %d artists for user %s from source %s", len(dbArtists), userId, source)
	return nil
}

func saveAlbums(userId string, items any, source string, tracker *ProcessedTracker) error {
	albums, ok := items.([]*spotify.Album)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Album")
	}
	if len(albums) == 0 {
		log.Printf("No albums to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d albums for user %s from source %s", len(albums), userId, source)
	dbAlbums := convertSpotifyAlbumsToDBAlbums(albums)

	var albumsToSave []*db.Album
	for _, album := range dbAlbums {
		if album != nil && album.AlbumId != "" && !tracker.CheckAndMark("album", album.AlbumId) {
			albumsToSave = append(albumsToSave, album)
		}
	}

	err := db.SaveAlbums(userId, albumsToSave, source)
	if err != nil {
		return fmt.Errorf("saving %d albums from %s: %w", len(albumsToSave), source, err)
	}

	err = db.SaveUserAlbumRelations(userId, dbAlbums, source)
	if err != nil {
		return fmt.Errorf("saving %d user-album relations from %s: %w", len(dbAlbums), source, err)
	}

	log.Printf("Saved %d albums for user %s from source %s", len(dbAlbums), userId, source)
	return nil
}
