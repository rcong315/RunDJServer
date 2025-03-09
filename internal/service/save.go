package service

import (
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func save(userId string, tracks []spotify.Track) {
	var trackData []db.Track
	for _, track := range tracks {
		dbTrack := convertSpotifyTrackToDBTrack(userId, track)
		trackData = append(trackData, dbTrack)
	}
	db.SaveTracks(userId, trackData)
}
