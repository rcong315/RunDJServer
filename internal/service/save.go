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
		log.Printf("Error saving user: %v", err)
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
		log.Printf("Error saving %d tracks for user %s from source %s", len(tracks), userId, source)
	} else {
		log.Printf("Saved %d tracks for user %s from source %s", len(tracks), userId, source)
	}
}

// func savePlaylists(userId string, playlists []*spotify.Playlist) {

// }

// func saveArtists(userId string, artists []*spotify.Artist, source string) {

// }

// func saveAlbums(userId string, albums []*spotify.Album, source string) {

// }

// func saveAllTracks(token string) {
// 	user, err := spotify.GetUser(token)
// 	if err != nil {
// 		log.Printf("Error getting user: %v", err)
// 		return
// 	}
// 	userId := user.Id

// 	// User's top tracks
// 	log.Print("Getting user's top tracks")
// 	usersTopTracks, err := spotify.GetUsersTopTracks(token)
// 	if err != nil {
// 		log.Printf("Error getting user's top tracks: %v", err)
// 	}
// 	go saveTracks(userId, usersTopTracks, "top tracks")

// 	// User's saved tracks
// 	log.Printf("Getting user's saved tracks")
// 	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
// 	if err != nil {
// 		log.Printf("Error getting user's saved tracks: %v", err)
// 	}
// 	go saveTracks(userId, usersSavedTracks, "saved tracks")

// 	// User's playlists
// 	log.Printf("Getting tracks from user's playlists")
// 	usersPlaylists, err := spotify.GetUsersPlaylists(token)
// 	if err != nil {
// 		log.Printf("Error getting user's playlists: %v", err)
// 	}
// 	go savePlaylists(userId, usersPlaylists)
// 	for _, playlist := range usersPlaylists {
// 		playlistId := playlist.Id
// 		log.Printf("Getting tracks from playlist: %s", playlistId)
// 		playlistTracks := spotify.GetPlaylistsTracks(token, playlistId)
// 		go saveTracks(userId, playlistTracks, "playlist tracks")
// 	}

// 	// User's top artists
// 	log.Printf("Getting tracks from user's top artists")
// 	usersTopArtists, err := spotify.GetUsersTopArtists(token)
// 	if err != nil {
// 		log.Printf("Error getting user's top artists: %v", err)
// 	}
// 	go saveArtists(userId, usersTopArtists, "top artists")
// 	for _, artist := range usersTopArtists {
// 		artistId := artist.Id
// 		log.Printf("Getting tracks from top artist's top tracks: %s", artistId)
// 		artistTopTracks := spotify.GetArtistsTopTracks(token, artistId)
// 		go saveTracks(userId, artistTopTracks, "top artists top tracks")

// 		// Top artist's albums
// 		log.Printf("Getting tracks from top artist's albums: %s", artistId)
// 		artistAlbums := spotify.GetArtistsAlbums(token, artistId)
// 		go saveAlbums(userId, artistAlbums, "top artists")
// 		for _, album := range artistAlbums {
// 			albumId := album.Id
// 			log.Printf("Getting tracks from album: %s", albumId)
// 			albumTracks := spotify.GetAlbumsTracks(token, albumId)
// 			go saveTracks(userId, albumTracks, "top artist's albums")
// 		}
// 	}

// 	// User's followed artists
// 	log.Printf("Getting tracks from user's followed artists")
// 	usersFollowedArtists, err := spotify.GetUsersFollowedArtists(token)
// 	if err != nil {
// 		log.Printf("Error getting user's followed artists: %v", err)
// 	}
// 	go saveArtists(userId, usersFollowedArtists)
// 	for _, artist := range usersFollowedArtists {
// 		artistId := artist.Id
// 		log.Printf("Getting tracks from followed artist's top tracks: %s", artistId)
// 		artistTopTracks := spotify.GetArtistsTopTracks(token, artistId)
// 		go saveTracks(userId, artistTopTracks, "followed artists top tracks")

// 		// Followed artist's albums
// 		log.Printf("Getting tracks from followed artist's albums: %s", artistId)
// 		artistAlbums := spotify.GetArtistsAlbums(token, artistId)
// 		go saveAlbums(userId, artistAlbums, "followed artists")
// 		for _, album := range artistAlbums {
// 			albumId := album.Id
// 			log.Printf("Getting tracks from album: %s", albumId)
// 			albumTracks := spotify.GetAlbumsTracks(token, albumId)
// 			go saveTracks(userId, albumTracks, "followed artist's albums")
// 		}
// 	}

// 	// User's saved albums
// 	log.Printf("Getting tracks from user's saved albums")
// 	usersSavedAlbums, err := spotify.GetUsersSavedAlbums(token)
// 	if err != nil {
// 		log.Printf("Error getting user's saved albums: %v", err)
// 	}
// 	go saveAlbums(userId, usersSavedAlbums)
// 	for _, album := range usersSavedAlbums {
// 		albumId := album.Id
// 		log.Printf("Getting tracks from album: %s", albumId)
// 		albumTracks := spotify.GetAlbumsTracks(token, albumId)
// 		go saveTracks(userId, albumTracks, "saved albums")
// 	}
// }
