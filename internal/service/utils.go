package service

import (
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func convertSpotifyTrackToDBTrack(userId string, track spotify.Track) db.Track {
	artistIds := make([]string, len(track.Artists))
	for i, artist := range track.Artists {
		artistIds[i] = artist.Id
	}

	// Create db.Track object
	dbTrack := db.Track{
		TrackId:          track.Id,
		UserIds:          []string{userId},
		Name:             track.Name,
		ArtistIds:        artistIds,
		AlbumId:          track.Album.Id,
		Popularity:       track.Popularity,
		DurationMS:       track.DurationMS,
		AvailableMarkets: track.AvailableMarkets,
		// AudioFeatures: AudioFeatures{
		// 	Danceability:      audioFeatures.Danceability,
		// 	Energy:            audioFeatures.Energy,
		// 	Key:               audioFeatures.Key,
		// 	Loudness:          audioFeatures.Loudness,
		// 	Mode:              audioFeatures.Mode,
		// 	Speechiness:       audioFeatures.Speechiness,
		// 	Acousticness:      audioFeatures.Acousticness,
		// 	Instrumentallness: audioFeatures.Instrumentallness,
		// 	Liveness:          audioFeatures.Liveness,
		// 	Valence:           audioFeatures.Valence,
		// 	Tempo:             audioFeatures.Tempo,
		// 	Duration:          audioFeatures.Duration,
		// 	TimeSignature:     audioFeatures.TimeSignature,
		// },
	}

	return dbTrack
}
