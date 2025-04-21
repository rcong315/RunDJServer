package service

import (
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

func saveTracks(userId string, tracks []*spotify.Track, source string) {
	log.Printf("Saving %d tracks for user %s from source %s", len(tracks), userId, source)
	var trackData []*db.Track
	for _, track := range tracks {
		dbTrack := convertSpotifyTrackToDBTrack(userId, track)
		trackData = append(trackData, dbTrack)
	}

	err := db.SaveTracks(userId, trackData, source)
	if err != nil {
		log.Print(err)
	} else {
		log.Printf("Saved %d tracks", len(trackData))
	}
}

func savePlaylists(userId string, playlists []*spotify.Playlist, source string) {
	log.Printf("Saving %d playlists for user %s from source %s", len(playlists), userId, source)
	var playlistData []*db.Playlist
	for _, playlist := range playlists {
		dbPlaylist := convertSpotifyPlaylistToDBPlaylist(userId, playlist)
		playlistData = append(playlistData, dbPlaylist)
	}

	err := db.SavePlaylists(userId, playlistData, source)
	if err != nil {
		log.Printf("Error saving playlists: %v", err)
	} else {
		log.Printf("Saved %d playlists", len(playlistData))
	}
}

func saveArtists(userId string, artists []*spotify.Artist, source string) {
	log.Printf("Saving %d artists for user %s from source %s", len(artists), userId, source)
	var artistData []*db.Artist
	for _, artist := range artists {
		dbArtist := convertSpotifyArtistToDBArtist(userId, artist)
		artistData = append(artistData, dbArtist)
	}

	err := db.SaveArtists(userId, artistData, source)
	if err != nil {
		log.Printf("Error saving artists: %v", err)
	} else {
		log.Printf("Saved %d artists", len(artistData))
	}
}

func saveAlbums(userId string, albums []*spotify.Album, source string) {
	log.Printf("Saving %d albums for user %s from source %s", len(albums), userId, source)
	var albumData []*db.Album
	for _, album := range albums {
		dbAlbum := convertSpotifyAlbumToDBAlbum(userId, album)
		albumData = append(albumData, dbAlbum)
	}

	err := db.SaveAlbums(userId, albumData, source)
	if err != nil {
		log.Printf("Error saving albums: %v", err)
	} else {
		log.Printf("Saved %d albums", len(albumData))
	}
}

// TODO: When to run this function? On register and when else? Cron?
func saveAllTracks(token string, userId string) {
	log.Printf("Getting all tracks for user %s", userId)

	// User's top tracks
	log.Print("Getting user's top tracks")
	usersTopTracks, err := spotify.GetUsersTopTracks(token)
	if err != nil {
		log.Printf("Error getting user's top tracks: %v", err)
	}
	go saveTracks(userId, usersTopTracks, "top tracks")

	// User's saved tracks
	log.Printf("Getting user's saved tracks")
	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
	if err != nil {
		log.Printf("Error getting user's saved tracks: %v", err)
	}
	go saveTracks(userId, usersSavedTracks, "saved tracks")

	// User's playlists
	log.Printf("Getting tracks from user's playlists")
	usersPlaylists, err := spotify.GetUsersPlaylists(token)
	if err != nil {
		log.Printf("Error getting user's playlists: %v", err)
	}
	go savePlaylists(userId, usersPlaylists, "playlists")
	for _, playlist := range usersPlaylists {
		playlistId := playlist.Id
		log.Printf("Getting tracks from playlist: %s", playlistId)
		playlistTracks, err := spotify.GetPlaylistsTracks(token, playlistId)
		if err != nil {
			log.Printf("Error getting tracks from playlist %s: %v", playlistId, err)
		}
		go saveTracks(userId, playlistTracks, "playlist tracks")
	}

	// User's top artists
	log.Printf("Getting tracks from user's top artists")
	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		log.Printf("Error getting user's top artists: %v", err)
	}
	go saveArtists(userId, usersTopArtists, "top artists")
	for _, artist := range usersTopArtists {
		artistId := artist.Id
		log.Printf("Getting tracks from top artist's top tracks: %s", artistId)
		artistTopTracks, err := spotify.GetArtistsTopTracks(token, artistId)
		if err != nil {
			log.Printf("Error getting top tracks for artist %s: %v", artistId, err)
		}
		go saveTracks(userId, artistTopTracks, "top artists top tracks")

		// Top artist's albums
		log.Printf("Getting tracks from top artist's albums: %s", artistId)
		artistAlbums, err := spotify.GetArtistsAlbums(token, artistId)
		if err != nil {
			log.Printf("Error getting albums for artist %s: %v", artistId, err)
		}
		go saveAlbums(userId, artistAlbums, "top artists")
		for _, album := range artistAlbums {
			albumId := album.Id
			log.Printf("Getting tracks from album: %s", albumId)
			albumTracks, err := spotify.GetAlbumsTracks(token, albumId)
			if err != nil {
				log.Printf("Error getting tracks from album %s: %v", albumId, err)
			}
			go saveTracks(userId, albumTracks, "top artist's albums")
		}
	}

	// User's followed artists
	log.Printf("Getting tracks from user's followed artists")
	usersFollowedArtists, err := spotify.GetUsersFollowedArtists(token)
	if err != nil {
		log.Printf("Error getting user's followed artists: %v", err)
	}
	go saveArtists(userId, usersFollowedArtists, "followed artists")
	for _, artist := range usersFollowedArtists {
		artistId := artist.Id
		log.Printf("Getting tracks from followed artist's top tracks: %s", artistId)
		artistTopTracks, err := spotify.GetArtistsTopTracks(token, artistId)
		if err != nil {
			log.Printf("Error getting top tracks for artist %s: %v", artistId, err)
		}
		go saveTracks(userId, artistTopTracks, "followed artists top tracks")

		// Followed artist's albums
		log.Printf("Getting tracks from followed artist's albums: %s", artistId)
		artistAlbums, err := spotify.GetArtistsAlbums(token, artistId)
		if err != nil {
			log.Printf("Error getting albums for artist %s: %v", artistId, err)
		}
		go saveAlbums(userId, artistAlbums, "followed artists")
		for _, album := range artistAlbums {
			albumId := album.Id
			log.Printf("Getting tracks from album: %s", albumId)
			albumTracks, err := spotify.GetAlbumsTracks(token, albumId)
			if err != nil {
				log.Printf("Error getting tracks from album %s: %v", albumId, err)
			}
			go saveTracks(userId, albumTracks, "followed artist's albums")
		}
	}

	// User's saved albums
	log.Printf("Getting tracks from user's saved albums")
	usersSavedAlbums, err := spotify.GetUsersSavedAlbums(token)
	if err != nil {
		log.Printf("Error getting user's saved albums: %v", err)
	}
	go saveAlbums(userId, usersSavedAlbums, "saved albums")
	for _, album := range usersSavedAlbums {
		albumId := album.Id
		log.Printf("Getting tracks from album: %s", albumId)
		albumTracks, err := spotify.GetAlbumsTracks(token, albumId)
		if err != nil {
			log.Printf("Error getting tracks from album %s: %v", albumId, err)
		}
		go saveTracks(userId, albumTracks, "saved albums")
	}

}
