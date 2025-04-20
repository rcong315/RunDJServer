package spotify

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// Mock for Config
type MockConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	FrontendURI  string
	Port         string
}

// Original functions to be replaced with mocks
var (
	originalGetConfig        func() (*Config, error)
	originalMakeTokenRequest func(config *Config, data url.Values) (*TokenResponse, error)
)

// Setup and teardown helpers
func setupAuthTest() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	// Save original functions
	originalGetConfig = GetConfig
	originalMakeTokenRequest = makeTokenRequest

	return r
}

func teardownAuthTest() {
	// Restore original functions
	GetConfig = originalGetConfig
	makeTokenRequest = originalMakeTokenRequest
}

func TestTokenHandler_ConfigError(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function to return an error
	GetConfig = func() (*Config, error) {
		return nil, errors.New("config error")
	}

	r.POST("/token", TokenHandler)

	// Create a test request
	reqBody := `{"code": "test-code"}`
	req := httptest.NewRequest("POST", "/token", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "Configuration error")
}

func TestTokenHandler_MissingCode(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function
	GetConfig = func() (*Config, error) {
		return &Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURI:  "http://localhost:3000/callback",
		}, nil
	}

	r.POST("/token", TokenHandler)

	// Create a test request without code
	reqBody := `{}`
	req := httptest.NewRequest("POST", "/token", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Authorization code is required", response.Error)
}

func TestTokenHandler_TokenRequestError(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function
	mockConfig := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	GetConfig = func() (*Config, error) {
		return mockConfig, nil
	}

	// Mock the makeTokenRequest function to return an error
	makeTokenRequest = func(config *Config, data url.Values) (*TokenResponse, error) {
		return nil, errors.New("token request error")
	}

	r.POST("/token", TokenHandler)

	// Create a test request with code
	reqBody := `{"code": "test-code"}`
	req := httptest.NewRequest("POST", "/token", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Failed to get token", response.Error)
}

func TestTokenHandler_Success(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function
	mockConfig := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	GetConfig = func() (*Config, error) {
		return mockConfig, nil
	}

	// Mock the makeTokenRequest function
	mockTokenResponse := &TokenResponse{
		AccessToken:  "test-access-token",
		TokenType:    "Bearer",
		ExpiresIn:    3600,
		RefreshToken: "test-refresh-token",
		Scope:        "user-read-private",
	}
	makeTokenRequest = func(config *Config, data url.Values) (*TokenResponse, error) {
		assert.Equal(t, mockConfig, config)
		assert.Equal(t, "test-code", data.Get("code"))
		assert.Equal(t, "http://localhost:3000/callback", data.Get("redirect_uri"))
		assert.Equal(t, "authorization_code", data.Get("grant_type"))
		return mockTokenResponse, nil
	}

	r.POST("/token", TokenHandler)

	// Create a test request with code
	reqBody := `{"code": "test-code"}`
	req := httptest.NewRequest("POST", "/token", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response TokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, mockTokenResponse.AccessToken, response.AccessToken)
	assert.Equal(t, mockTokenResponse.TokenType, response.TokenType)
	assert.Equal(t, mockTokenResponse.ExpiresIn, response.ExpiresIn)
	assert.Equal(t, mockTokenResponse.RefreshToken, response.RefreshToken)
	assert.Equal(t, mockTokenResponse.Scope, response.Scope)
}

func TestRefreshHandler_ConfigError(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function to return an error
	GetConfig = func() (*Config, error) {
		return nil, errors.New("config error")
	}

	r.POST("/refresh", RefreshHandler)

	// Create a test request
	reqBody := `{"refresh_token": "test-refresh-token"}`
	req := httptest.NewRequest("POST", "/refresh", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "Configuration error")
}

func TestRefreshHandler_MissingRefreshToken(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function
	GetConfig = func() (*Config, error) {
		return &Config{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			RedirectURI:  "http://localhost:3000/callback",
		}, nil
	}

	r.POST("/refresh", RefreshHandler)

	// Create a test request without refresh_token
	reqBody := `{}`
	req := httptest.NewRequest("POST", "/refresh", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusBadRequest, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Refresh token is required", response.Error)
}

func TestRefreshHandler_TokenRequestError(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function
	mockConfig := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	GetConfig = func() (*Config, error) {
		return mockConfig, nil
	}

	// Mock the makeTokenRequest function to return an error
	makeTokenRequest = func(config *Config, data url.Values) (*TokenResponse, error) {
		return nil, errors.New("token request error")
	}

	r.POST("/refresh", RefreshHandler)

	// Create a test request with refresh_token
	reqBody := `{"refresh_token": "test-refresh-token"}`
	req := httptest.NewRequest("POST", "/refresh", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusInternalServerError, w.Code)

	var response ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Contains(t, response.Error, "Failed to refresh token")
}

func TestRefreshHandler_Success(t *testing.T) {
	r := setupAuthTest()
	defer teardownAuthTest()

	// Mock the GetConfig function
	mockConfig := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	GetConfig = func() (*Config, error) {
		return mockConfig, nil
	}

	// Mock the makeTokenRequest function
	mockTokenResponse := &TokenResponse{
		AccessToken: "test-access-token",
		TokenType:   "Bearer",
		ExpiresIn:   3600,
		Scope:       "user-read-private",
	}
	makeTokenRequest = func(config *Config, data url.Values) (*TokenResponse, error) {
		assert.Equal(t, mockConfig, config)
		assert.Equal(t, "test-refresh-token", data.Get("refresh_token"))
		assert.Equal(t, "refresh_token", data.Get("grant_type"))
		return mockTokenResponse, nil
	}

	r.POST("/refresh", RefreshHandler)

	// Create a test request with refresh_token
	reqBody := `{"refresh_token": "test-refresh-token"}`
	req := httptest.NewRequest("POST", "/refresh", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Assert response
	assert.Equal(t, http.StatusOK, w.Code)

	var response TokenResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, mockTokenResponse.AccessToken, response.AccessToken)
	assert.Equal(t, mockTokenResponse.TokenType, response.TokenType)
	assert.Equal(t, mockTokenResponse.ExpiresIn, response.ExpiresIn)
	assert.Equal(t, "test-refresh-token", response.RefreshToken) // Should include the original refresh token
	assert.Equal(t, mockTokenResponse.Scope, response.Scope)
}

func TestMakeTokenRequest(t *testing.T) {
	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/token", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		assert.Contains(t, r.Header.Get("Authorization"), "Basic ")

		// Parse form data
		err := r.ParseForm()
		assert.NoError(t, err)
		assert.Equal(t, "test-code", r.Form.Get("code"))
		assert.Equal(t, "authorization_code", r.Form.Get("grant_type"))

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{
			"access_token": "test-access-token",
			"token_type": "Bearer",
			"expires_in": 3600,
			"refresh_token": "test-refresh-token",
			"scope": "user-read-private"
		}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Override the spotifyTokenURL constant for this test
	originalSpotifyTokenURL := spotifyTokenURL
	spotifyTokenURL = mockServer.URL + "/api/token"
	defer func() { spotifyTokenURL = originalSpotifyTokenURL }()

	// Create test config and data
	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	data := url.Values{}
	data.Set("code", "test-code")
	data.Set("grant_type", "authorization_code")

	// Call the function under test
	response, err := makeTokenRequest(config, data)

	// Assert the result
	assert.NoError(t, err)
	assert.NotNil(t, response)
	assert.Equal(t, "test-access-token", response.AccessToken)
	assert.Equal(t, "Bearer", response.TokenType)
	assert.Equal(t, 3600, response.ExpiresIn)
	assert.Equal(t, "test-refresh-token", response.RefreshToken)
	assert.Equal(t, "user-read-private", response.Scope)
}

func TestMakeTokenRequest_RequestError(t *testing.T) {
	// Override the spotifyTokenURL constant for this test to an invalid URL
	originalSpotifyTokenURL := spotifyTokenURL
	spotifyTokenURL = "http://invalid-url"
	defer func() { spotifyTokenURL = originalSpotifyTokenURL }()

	// Create test config and data
	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	data := url.Values{}
	data.Set("code", "test-code")
	data.Set("grant_type", "authorization_code")

	// Call the function under test
	response, err := makeTokenRequest(config, data)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, response)
}

func TestMakeTokenRequest_ErrorResponse(t *testing.T) {
	// Create a mock HTTP server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_grant"}`))
	}))
	defer mockServer.Close()

	// Override the spotifyTokenURL constant for this test
	originalSpotifyTokenURL := spotifyTokenURL
	spotifyTokenURL = mockServer.URL
	defer func() { spotifyTokenURL = originalSpotifyTokenURL }()

	// Create test config and data
	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	data := url.Values{}
	data.Set("code", "test-code")
	data.Set("grant_type", "authorization_code")

	// Call the function under test
	response, err := makeTokenRequest(config, data)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, response)
	assert.Contains(t, err.Error(), "spotify API returned 400")
}

func TestMakeTokenRequest_InvalidJSON(t *testing.T) {
	// Create a mock HTTP server that returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer mockServer.Close()

	// Override the spotifyTokenURL constant for this test
	originalSpotifyTokenURL := spotifyTokenURL
	spotifyTokenURL = mockServer.URL
	defer func() { spotifyTokenURL = originalSpotifyTokenURL }()

	// Create test config and data
	config := &Config{
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		RedirectURI:  "http://localhost:3000/callback",
	}
	data := url.Values{}
	data.Set("code", "test-code")
	data.Set("grant_type", "authorization_code")

	// Call the function under test
	response, err := makeTokenRequest(config, data)

	// Assert the result
	assert.Error(t, err)
	assert.Nil(t, response)
}
