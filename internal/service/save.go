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
	var token = "BQBHUGv9VxO6joxO7_5iEmur3iSZZ0BQpvP4yGe9FmGSmzHvbKgI3QbP_llkaibB7kheWrwgsqmL3T3no3yhTJAZQQFCflaZSQDyl5doNl4XL8cc6Q8vSoIoTnlG8Jo6-cYJbflKgyc"
	tracks, err := spotify.GetAudioFeatures(token, tracks)
	if err != nil {
		return err
	}
	var trackData []db.Track
	for _, track := range tracks {
		dbTrack := convertSpotifyTrackToDBTrack(userId, track)
		trackData = append(trackData, dbTrack)
	}
	return db.SaveTracks(userId, trackData, source)
}
