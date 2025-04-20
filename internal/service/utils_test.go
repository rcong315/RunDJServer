package service

import (
	"testing"

	"github.com/rcong315/RunDJServer/internal/spotify"
	"github.com/stretchr/testify/assert"
)

func TestConvertSpotifyUserToDBUser(t *testing.T) {
	// Create a test Spotify user
	spotifyUser := &spotify.User{
		Id:          "test-user-id",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Country:     "US",
		Product:     "premium",
	}
	spotifyUser.Followers.Total = 100
	spotifyUser.ImageURLs = []spotify.Image{
		{URL: "http://example.com/image1.jpg"},
		{URL: "http://example.com/image2.jpg"},
	}

	// Call the function under test
	dbUser := convertSpotifyUserToDBUser(spotifyUser)

	// Assert the result
	assert.NotNil(t, dbUser)
	assert.Equal(t, "test-user-id", dbUser.UserId)
	assert.Equal(t, "test@example.com", dbUser.Email)
	assert.Equal(t, "Test User", dbUser.DisplayName)
	assert.Equal(t, "US", dbUser.Country)
	assert.Equal(t, 100, dbUser.Followers)
	assert.Equal(t, "premium", dbUser.Product)
	assert.Equal(t, 2, len(dbUser.ImageURLs))
	assert.Equal(t, "http://example.com/image1.jpg", dbUser.ImageURLs[0])
	assert.Equal(t, "http://example.com/image2.jpg", dbUser.ImageURLs[1])
}

func TestConvertSpotifyTrackToDBTrack(t *testing.T) {
	// Create a test Spotify track
	spotifyTrack := &spotify.Track{
		Id:               "test-track-id",
		Name:             "Test Track",
		Popularity:       80,
		DurationMS:       180000,
		AvailableMarkets: []string{"US", "CA"},
	}
	spotifyTrack.Album = &spotify.Album{
		Id: "test-album-id",
	}
	spotifyTrack.Artists = []*spotify.Artist{
		{Id: "artist1", Name: "Artist 1"},
		{Id: "artist2", Name: "Artist 2"},
	}
	spotifyTrack.AudioFeatures = &spotify.AudioFeatures{
		Danceability:      0.8,
		Energy:            0.9,
		Key:               5,
		Loudness:          -5.0,
		Mode:              1,
		Speechiness:       0.1,
		Acousticness:      0.2,
		Instrumentallness: 0.3,
		Liveness:          0.4,
		Valence:           0.5,
		Tempo:             120.0,
		Duration:          180000,
		TimeSignature:     4,
	}

	// Call the function under test
	userId := "test-user-id"
	dbTrack := convertSpotifyTrackToDBTrack(userId, spotifyTrack)

	// Assert the result
	assert.NotNil(t, dbTrack)
	assert.Equal(t, "test-track-id", dbTrack.TrackId)
	assert.Equal(t, "Test Track", dbTrack.Name)
	assert.Equal(t, "test-album-id", dbTrack.AlbumId)
	assert.Equal(t, 80, dbTrack.Popularity)
	assert.Equal(t, 180000, dbTrack.DurationMS)
	assert.Equal(t, []string{"US", "CA"}, dbTrack.AvailableMarkets)
	assert.Equal(t, 2, len(dbTrack.ArtistIds))
	assert.Equal(t, "artist1", dbTrack.ArtistIds[0])
	assert.Equal(t, "artist2", dbTrack.ArtistIds[1])

	// Assert audio features
	assert.NotNil(t, dbTrack.AudioFeatures)
	assert.Equal(t, 0.8, dbTrack.AudioFeatures.Danceability)
	assert.Equal(t, 0.9, dbTrack.AudioFeatures.Energy)
	assert.Equal(t, 5, dbTrack.AudioFeatures.Key)
	assert.Equal(t, -5.0, dbTrack.AudioFeatures.Loudness)
	assert.Equal(t, 1, dbTrack.AudioFeatures.Mode)
	assert.Equal(t, 0.1, dbTrack.AudioFeatures.Speechiness)
	assert.Equal(t, 0.2, dbTrack.AudioFeatures.Acousticness)
	assert.Equal(t, 0.3, dbTrack.AudioFeatures.Instrumentallness)
	assert.Equal(t, 0.4, dbTrack.AudioFeatures.Liveness)
	assert.Equal(t, 0.5, dbTrack.AudioFeatures.Valence)
	assert.Equal(t, 120.0, dbTrack.AudioFeatures.Tempo)
	assert.Equal(t, 180000, dbTrack.AudioFeatures.Duration)
	assert.Equal(t, 4, dbTrack.AudioFeatures.TimeSignature)
}

func TestPresetPlaylists(t *testing.T) {
	// Test that the preset playlists map is not empty
	assert.NotEmpty(t, presetPlaylists)

	// Test that specific BPMs have playlist IDs
	assert.Contains(t, presetPlaylists, 120)
	assert.Contains(t, presetPlaylists, 140)
	assert.Contains(t, presetPlaylists, 160)

	// Test that the playlist IDs are not empty
	assert.NotEmpty(t, presetPlaylists[120])
	assert.NotEmpty(t, presetPlaylists[140])
	assert.NotEmpty(t, presetPlaylists[160])
}
