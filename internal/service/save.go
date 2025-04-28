package service

import (
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func saveUser(user spotify.User) error {
	dbUser := convertSpotifyUserToDBUser(user)
	return db.SaveUser(dbUser)
}

func saveTracks(userId string, tracks []spotify.Track, source string) error {
	var trackData []db.Track
	for _, track := range tracks {
		dbTrack := convertSpotifyTrackToDBTrack(userId, track)
		trackData = append(trackData, dbTrack)
	}
	return db.SaveTracks(userId, trackData, source)
}
