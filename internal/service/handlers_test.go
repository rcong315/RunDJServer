package service

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/rcong315/RunDJServer/internal/spotify"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock for spotify package
type MockSpotify struct {
	mock.Mock
}

func (m *MockSpotify) GetUser(token string) (*spotify.User, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*spotify.User), args.Error(1)
}

func (m *MockSpotify) GetRecommendations(seedArtists, seedGenres []string, minTempo, maxTempo float64) ([]*spotify.Track, error) {
	args := m.Called(seedArtists, seedGenres, minTempo, maxTempo)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*spotify.Track), args.Error(1)
}

func (m *MockSpotify) GetUsersTopTracks(token string) ([]*spotify.Track, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*spotify.Track), args.Error(1)
}

// Original functions to be replaced with mocks
var (
	originalGetUser            = spotify.GetUser
	originalGetRecommendations = spotify.GetRecommendations
	originalGetUsersTopTracks  = spotify.GetUsersTopTracks
	originalSaveUser           func(user *spotify.User)
	originalSaveTracks         func(userId string, tracks []*spotify.Track, source string)
)

// Setup and teardown helpers
func setupTest() (*gin.Engine, *MockSpotify) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mockSpotify := new(MockSpotify)

	// Save original functions
	originalSaveUser = saveUser
	originalSaveTracks = saveTracks

	return r, mockSpotify
}

func teardownTest() {
	// Restore original functions
	spotify.GetUser = originalGetUser
	spotify.GetRecommendations = originalGetRecommendations
	spotify.GetUsersTopTracks = originalGetUsersTopTracks
	saveUser = originalSaveUser
	saveTracks = originalSaveTracks
}

func TestHomeHandler(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/", HomeHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "RunDJ Backend", w.Body.String())
}

func TestThanksHandler(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/thanks", ThanksHandler)

	// Create a test request
	req := httptest.NewRequest("GET", "/thanks", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "getsongbpm.com")
	assert.Equal(t, "text/html", w.Header().Get("Content-Type"))
}

func TestRegisterHandler_MissingAccessToken(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/register", RegisterHandler)

	// Create a test request without access_token
	req := httptest.NewRequest("GET", "/register", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing access_token", response["error"])
}

func TestRegisterHandler_Success(t *testing.T) {
	r, mockSpotify := setupTest()
	defer teardownTest()

	// Mock the GetUser function
	mockUser := &spotify.User{
		Id:          "test-user-id",
		Email:       "test@example.com",
		DisplayName: "Test User",
	}
	mockSpotify.On("GetUser", "test-token").Return(mockUser, nil)
	spotify.GetUser = mockSpotify.GetUser

	// Mock the GetUsersTopTracks function
	mockTracks := []*spotify.Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}
	mockSpotify.On("GetUsersTopTracks", "test-token").Return(mockTracks, nil)
	spotify.GetUsersTopTracks = mockSpotify.GetUsersTopTracks

	// Mock the saveUser and saveTracks functions
	saveUserCalled := false
	saveUser = func(user *spotify.User) {
		saveUserCalled = true
		assert.Equal(t, mockUser, user)
	}

	saveTracksCalled := false
	saveTracks = func(userId string, tracks []*spotify.Track, source string) {
		saveTracksCalled = true
		assert.Equal(t, "test-user-id", userId)
		assert.Equal(t, mockTracks, tracks)
		assert.Equal(t, "top tracks", source)
	}

	r.GET("/register", RegisterHandler)

	// Create a test request with access_token
	req := httptest.NewRequest("GET", "/register?access_token=test-token", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response Message
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "success", response.Status)
	assert.Equal(t, "User registered successfully, processing tracks", response.Message)

	// Verify that the mock functions were called
	mockSpotify.AssertExpectations(t)
	assert.True(t, saveUserCalled)

	// Note: saveTracks is called in a goroutine, so we might not be able to verify it immediately
	// In a real test, you might need to use a wait group or channel to synchronize
}

func TestRegisterHandler_GetUserError(t *testing.T) {
	r, mockSpotify := setupTest()
	defer teardownTest()

	// Mock the GetUser function to return an error
	mockSpotify.On("GetUser", "test-token").Return(nil, errors.New("user fetch error"))
	spotify.GetUser = mockSpotify.GetUser

	r.GET("/register", RegisterHandler)

	// Create a test request with access_token
	req := httptest.NewRequest("GET", "/register?access_token=test-token", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Error getting user")

	// Verify that the mock function was called
	mockSpotify.AssertExpectations(t)
}

func TestPresetPlaylistHandler_MissingBPM(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/preset", PresetPlaylistHandler)

	// Create a test request without bpm
	req := httptest.NewRequest("GET", "/preset", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing bpm", response["error"])
}

func TestPresetPlaylistHandler_InvalidBPM(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/preset", PresetPlaylistHandler)

	// Create a test request with invalid bpm
	req := httptest.NewRequest("GET", "/preset?bpm=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid bpm")
}

func TestPresetPlaylistHandler_PlaylistNotFound(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/preset", PresetPlaylistHandler)

	// Create a test request with bpm that doesn't have a preset playlist
	req := httptest.NewRequest("GET", "/preset?bpm=999", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Playlist not found for the given BPM", response["error"])
}

func TestPresetPlaylistHandler_Success(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/preset", PresetPlaylistHandler)

	// Create a test request with valid bpm
	req := httptest.NewRequest("GET", "/preset?bpm=120", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, presetPlaylists[120], w.Body.String())
}

func TestRecommendationsHandler_MissingAccessToken(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/recommendations", RecommendationsHandler)

	// Create a test request without access_token
	req := httptest.NewRequest("GET", "/recommendations", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing access_token", response["error"])
}

func TestRecommendationsHandler_MissingSeedArtistsAndGenres(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/recommendations", RecommendationsHandler)

	// Create a test request with access_token but no seed_artists or seed_genres
	req := httptest.NewRequest("GET", "/recommendations?access_token=test-token", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing seed_artists or seed_genres", response["error"])
}

func TestRecommendationsHandler_MissingBPM(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/recommendations", RecommendationsHandler)

	// Create a test request with access_token and seed_artists but no bpm
	req := httptest.NewRequest("GET", "/recommendations?access_token=test-token&seed_artists=artist1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Missing bpm", response["error"])
}

func TestRecommendationsHandler_InvalidBPM(t *testing.T) {
	r, _ := setupTest()
	defer teardownTest()

	r.GET("/recommendations", RecommendationsHandler)

	// Create a test request with access_token, seed_artists, but invalid bpm
	req := httptest.NewRequest("GET", "/recommendations?access_token=test-token&seed_artists=artist1&bpm=invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Invalid bpm")
}

func TestRecommendationsHandler_Success(t *testing.T) {
	r, mockSpotify := setupTest()
	defer teardownTest()

	// Mock the GetRecommendations function
	seedArtists := []string{"artist1"}
	seedGenres := []string{"genre1"}
	minBPM := 118.0
	maxBPM := 122.0
	mockTracks := []*spotify.Track{
		{Id: "track1", Name: "Track 1"},
		{Id: "track2", Name: "Track 2"},
	}
	mockSpotify.On("GetRecommendations", seedArtists, seedGenres, minBPM, maxBPM).Return(mockTracks, nil)
	spotify.GetRecommendations = mockSpotify.GetRecommendations

	r.GET("/recommendations", RecommendationsHandler)

	// Create a test request with all required parameters
	req := httptest.NewRequest("GET", "/recommendations?access_token=test-token&seed_artists=artist1&seed_genres=genre1&bpm=120", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var trackIds []string
	err := json.Unmarshal(w.Body.Bytes(), &trackIds)
	assert.NoError(t, err)
	assert.Equal(t, []string{"track1", "track2"}, trackIds)

	// Verify that the mock function was called
	mockSpotify.AssertExpectations(t)
}

func TestRecommendationsHandler_GetRecommendationsError(t *testing.T) {
	r, mockSpotify := setupTest()
	defer teardownTest()

	// Mock the GetRecommendations function to return an error
	seedArtists := []string{"artist1"}
	seedGenres := []string{"genre1"}
	minBPM := 118.0
	maxBPM := 122.0
	mockSpotify.On("GetRecommendations", seedArtists, seedGenres, minBPM, maxBPM).Return(nil, errors.New("recommendations error"))
	spotify.GetRecommendations = mockSpotify.GetRecommendations

	r.GET("/recommendations", RecommendationsHandler)

	// Create a test request with all required parameters
	req := httptest.NewRequest("GET", "/recommendations?access_token=test-token&seed_artists=artist1&seed_genres=genre1&bpm=120", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response["error"], "Error getting recommendations")

	// Verify that the mock function was called
	mockSpotify.AssertExpectations(t)
}
