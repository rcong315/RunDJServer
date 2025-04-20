package spotify

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock HTTP client
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

// Original HTTP client to be replaced with mock
var originalHTTPClient *http.Client

// Setup and teardown helpers
func setupSpotifyTest() *MockHTTPClient {
	originalHTTPClient = &http.Client{}
	mockClient := new(MockHTTPClient)
	return mockClient
}

func teardownSpotifyTest() {
	// Restore original HTTP client if needed
}

// Helper function to create a mock HTTP response
func createMockResponse(statusCode int, body interface{}) *http.Response {
	var responseBody io.ReadCloser

	if body != nil {
		bodyBytes, _ := json.Marshal(body)
		responseBody = io.NopCloser(strings.NewReader(string(bodyBytes)))
	} else {
		responseBody = io.NopCloser(strings.NewReader(""))
	}

	return &http.Response{
		StatusCode: statusCode,
		Body:       responseBody,
	}
}

func TestGetUser(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create a mock user response
	mockUser := User{
		Id:          "test-user-id",
		Email:       "test@example.com",
		DisplayName: "Test User",
		Country:     "US",
		Product:     "premium",
	}
	mockUser.Followers.Total = 100

	// Create a mock HTTP response
	mockResponse := createMockResponse(http.StatusOK, mockUser)

	// Store the original function and create a mock
	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*User, error) {
		assert.Equal(t, "test-token", token)
		assert.Equal(t, spotifyAPIURL+"me", url)
		return []*User{&mockUser}, nil
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	user, err := GetUser("test-token")

	// Assert the result
	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test-user-id", user.Id)
	assert.Equal(t, "test@example.com", user.Email)
	assert.Equal(t, "Test User", user.DisplayName)
	assert.Equal(t, "US", user.Country)
	assert.Equal(t, 100, user.Followers.Total)
	assert.Equal(t, "premium", user.Product)
}

func TestGetUser_Error(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Store the original function and create a mock
	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*User, error) {
		return nil, errors.New("API error")
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	user, err := GetUser("test-token")

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "API error")
}

func TestGetRecommendations(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create a mock recommendations response
	mockTracks := []Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}
	mockResponse := RecommendationsResponse{
		Tracks: mockTracks,
	}

	// Store the original functions and create mocks
	originalGetSecretToken := getSecretToken
	getSecretToken = func() (string, error) {
		return "mock-token", nil
	}
	defer func() { getSecretToken = originalGetSecretToken }()

	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*RecommendationsResponse, error) {
		assert.Equal(t, "mock-token", token)
		assert.Contains(t, url, "seed_artists=artist1")
		assert.Contains(t, url, "seed_genres=genre1")
		assert.Contains(t, url, "min_tempo=118.000000")
		assert.Contains(t, url, "max_tempo=122.000000")
		return []*RecommendationsResponse{&mockResponse}, nil
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	seedArtists := []string{"artist1"}
	seedGenres := []string{"genre1"}
	minTempo := 118.0
	maxTempo := 122.0
	tracks, err := GetRecommendations(seedArtists, seedGenres, minTempo, maxTempo)

	// Assert the result
	assert.NoError(t, err)
	assert.NotNil(t, tracks)
	assert.Equal(t, 2, len(tracks))
	assert.Equal(t, "track1", tracks[0].Id)
	assert.Equal(t, "Track 1", tracks[0].Name)
	assert.Equal(t, "track2", tracks[1].Id)
	assert.Equal(t, "Track 2", tracks[1].Name)
}

func TestGetRecommendations_TokenError(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Store the original function and create a mock
	originalGetSecretToken := getSecretToken
	getSecretToken = func() (string, error) {
		return "", errors.New("token error")
	}
	defer func() { getSecretToken = originalGetSecretToken }()

	// Call the function under test
	seedArtists := []string{"artist1"}
	seedGenres := []string{"genre1"}
	minTempo := 118.0
	maxTempo := 122.0
	tracks, err := GetRecommendations(seedArtists, seedGenres, minTempo, maxTempo)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, tracks)
	assert.Contains(t, err.Error(), "token error")
}

func TestGetRecommendations_APIError(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Store the original functions and create mocks
	originalGetSecretToken := getSecretToken
	getSecretToken = func() (string, error) {
		return "mock-token", nil
	}
	defer func() { getSecretToken = originalGetSecretToken }()

	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*RecommendationsResponse, error) {
		return nil, errors.New("API error")
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	seedArtists := []string{"artist1"}
	seedGenres := []string{"genre1"}
	minTempo := 118.0
	maxTempo := 122.0
	tracks, err := GetRecommendations(seedArtists, seedGenres, minTempo, maxTempo)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, tracks)
	assert.Contains(t, err.Error(), "API error")
}

func TestGetUsersTopTracks(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create a mock top tracks response
	mockTracks := []Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}
	mockResponse := UsersTopTracksResponse{
		Items: mockTracks,
	}

	// Store the original functions and create mocks
	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*UsersTopTracksResponse, error) {
		assert.Equal(t, "test-token", token)
		assert.Contains(t, url, "me/top/tracks")
		return []*UsersTopTracksResponse{&mockResponse}, nil
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	originalGetAudioFeatures := getAudioFeatures
	getAudioFeatures = func(tracks []*Track) ([]*Track, error) {
		// Just return the tracks as is for this test
		return tracks, nil
	}
	defer func() { getAudioFeatures = originalGetAudioFeatures }()

	// Call the function under test
	tracks, err := GetUsersTopTracks("test-token")

	// Assert the result
	assert.NoError(t, err)
	assert.NotNil(t, tracks)
	assert.Equal(t, 2, len(tracks))
	assert.Equal(t, "track1", tracks[0].Id)
	assert.Equal(t, "Track 1", tracks[0].Name)
	assert.Equal(t, "track2", tracks[1].Id)
	assert.Equal(t, "Track 2", tracks[1].Name)
}

func TestGetUsersTopTracks_APIError(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Store the original function and create a mock
	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*UsersTopTracksResponse, error) {
		return nil, errors.New("API error")
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	tracks, err := GetUsersTopTracks("test-token")

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, tracks)
	assert.Contains(t, err.Error(), "API error")
}

func TestGetUsersTopTracks_AudioFeaturesError(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create a mock top tracks response
	mockTracks := []Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}
	mockResponse := UsersTopTracksResponse{
		Items: mockTracks,
	}

	// Store the original functions and create mocks
	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*UsersTopTracksResponse, error) {
		return []*UsersTopTracksResponse{&mockResponse}, nil
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	originalGetAudioFeatures := getAudioFeatures
	getAudioFeatures = func(tracks []*Track) ([]*Track, error) {
		return nil, errors.New("audio features error")
	}
	defer func() { getAudioFeatures = originalGetAudioFeatures }()

	// Call the function under test
	tracks, err := GetUsersTopTracks("test-token")

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, tracks)
	assert.Contains(t, err.Error(), "audio features error")
}

func TestGetAudioFeatures(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create mock tracks
	mockTracks := []*Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}

	// Create a mock audio features response
	mockAudioFeatures := []AudioFeatures{
		{Id: "track1", Tempo: 120.0},
		{Id: "track2", Tempo: 130.0},
	}
	mockResponse := AudioFeaturesResponse{
		AudioFeatures: mockAudioFeatures,
	}

	// Store the original functions and create mocks
	originalGetSecretToken := getSecretToken
	getSecretToken = func() (string, error) {
		return "mock-token", nil
	}
	defer func() { getSecretToken = originalGetSecretToken }()

	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*AudioFeaturesResponse, error) {
		assert.Equal(t, "mock-token", token)
		assert.Contains(t, url, "audio-features?ids=track1,track2")
		return []*AudioFeaturesResponse{&mockResponse}, nil
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	tracks, err := getAudioFeatures(mockTracks)

	// Assert the result
	assert.NoError(t, err)
	assert.NotNil(t, tracks)
	assert.Equal(t, 2, len(tracks))
	assert.Equal(t, "track1", tracks[0].Id)
	assert.Equal(t, "Track 1", tracks[0].Name)
	assert.Equal(t, 120.0, tracks[0].AudioFeatures.Tempo)
	assert.Equal(t, "track2", tracks[1].Id)
	assert.Equal(t, "Track 2", tracks[1].Name)
	assert.Equal(t, 130.0, tracks[1].AudioFeatures.Tempo)
}

func TestGetAudioFeatures_TokenError(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create mock tracks
	mockTracks := []*Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}

	// Store the original function and create a mock
	originalGetSecretToken := getSecretToken
	getSecretToken = func() (string, error) {
		return "", errors.New("token error")
	}
	defer func() { getSecretToken = originalGetSecretToken }()

	// Call the function under test
	tracks, err := getAudioFeatures(mockTracks)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, tracks)
	assert.Contains(t, err.Error(), "token error")
}

func TestGetAudioFeatures_APIError(t *testing.T) {
	mockClient := setupSpotifyTest()
	defer teardownSpotifyTest()

	// Create mock tracks
	mockTracks := []*Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}

	// Store the original functions and create mocks
	originalGetSecretToken := getSecretToken
	getSecretToken = func() (string, error) {
		return "mock-token", nil
	}
	defer func() { getSecretToken = originalGetSecretToken }()

	originalFetchAllResults := fetchAllResults
	fetchAllResults = func(token string, url string) ([]*AudioFeaturesResponse, error) {
		return nil, errors.New("API error")
	}
	defer func() { fetchAllResults = originalFetchAllResults }()

	// Call the function under test
	tracks, err := getAudioFeatures(mockTracks)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, tracks)
	assert.Contains(t, err.Error(), "API error")
}
