package spotify

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetSecretToken(t *testing.T) {
	// Save original environment variable and restore it after the test
	originalTokenURL := os.Getenv("TOKEN_URL")
	defer os.Setenv("TOKEN_URL", originalTokenURL)

	// Create a mock HTTP server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check request
		assert.Equal(t, "GET", r.Method)

		// Return mock response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		response := `{"access_token": "test-access-token"}`
		w.Write([]byte(response))
	}))
	defer mockServer.Close()

	// Set the TOKEN_URL environment variable to the mock server URL
	os.Setenv("TOKEN_URL", mockServer.URL)

	// Save original apiURL and restore it after the test
	originalAPIURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalAPIURL }()

	// Call the function under test
	token, err := getSecretToken()

	// Assert the result
	assert.NoError(t, err)
	assert.Equal(t, "test-access-token", token)
}

func TestGetSecretToken_MissingTokenURL(t *testing.T) {
	// Save original environment variable and restore it after the test
	originalTokenURL := os.Getenv("TOKEN_URL")
	defer os.Setenv("TOKEN_URL", originalTokenURL)

	// Set the TOKEN_URL environment variable to empty
	os.Setenv("TOKEN_URL", "")

	// Save original apiURL and restore it after the test
	originalAPIURL := apiURL
	apiURL = ""
	defer func() { apiURL = originalAPIURL }()

	// Call the function under test
	token, err := getSecretToken()

	// Assert the result
	assert.Error(t, err)
	assert.Equal(t, "", token)
	assert.Contains(t, err.Error(), "TOKEN_URL environment variable not set")
}

func TestGetSecretToken_RequestError(t *testing.T) {
	// Save original environment variable and restore it after the test
	originalTokenURL := os.Getenv("TOKEN_URL")
	defer os.Setenv("TOKEN_URL", originalTokenURL)

	// Set the TOKEN_URL environment variable to an invalid URL
	os.Setenv("TOKEN_URL", "http://invalid-url")

	// Save original apiURL and restore it after the test
	originalAPIURL := apiURL
	apiURL = "http://invalid-url"
	defer func() { apiURL = originalAPIURL }()

	// Call the function under test
	token, err := getSecretToken()

	// Assert the result
	assert.Error(t, err)
	assert.Equal(t, "", token)
	assert.Contains(t, err.Error(), "http.Get failed")
}

func TestGetSecretToken_ErrorResponse(t *testing.T) {
	// Save original environment variable and restore it after the test
	originalTokenURL := os.Getenv("TOKEN_URL")
	defer os.Setenv("TOKEN_URL", originalTokenURL)

	// Create a mock HTTP server that returns an error
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "invalid_request"}`))
	}))
	defer mockServer.Close()

	// Set the TOKEN_URL environment variable to the mock server URL
	os.Setenv("TOKEN_URL", mockServer.URL)

	// Save original apiURL and restore it after the test
	originalAPIURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalAPIURL }()

	// Call the function under test
	token, err := getSecretToken()

	// Assert the result
	assert.Error(t, err)
	assert.Equal(t, "", token)
	assert.Contains(t, err.Error(), "API request failed with status code: 400")
}

func TestGetSecretToken_InvalidJSON(t *testing.T) {
	// Save original environment variable and restore it after the test
	originalTokenURL := os.Getenv("TOKEN_URL")
	defer os.Setenv("TOKEN_URL", originalTokenURL)

	// Create a mock HTTP server that returns invalid JSON
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json`))
	}))
	defer mockServer.Close()

	// Set the TOKEN_URL environment variable to the mock server URL
	os.Setenv("TOKEN_URL", mockServer.URL)

	// Save original apiURL and restore it after the test
	originalAPIURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalAPIURL }()

	// Call the function under test
	token, err := getSecretToken()

	// Assert the result
	assert.Error(t, err)
	assert.Equal(t, "", token)
	assert.Contains(t, err.Error(), "error unmarshalling JSON response")
}

func TestGetSecretToken_EmptyToken(t *testing.T) {
	// Save original environment variable and restore it after the test
	originalTokenURL := os.Getenv("TOKEN_URL")
	defer os.Setenv("TOKEN_URL", originalTokenURL)

	// Create a mock HTTP server that returns an empty token
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"access_token": ""}`))
	}))
	defer mockServer.Close()

	// Set the TOKEN_URL environment variable to the mock server URL
	os.Setenv("TOKEN_URL", mockServer.URL)

	// Save original apiURL and restore it after the test
	originalAPIURL := apiURL
	apiURL = mockServer.URL
	defer func() { apiURL = originalAPIURL }()

	// Call the function under test
	token, err := getSecretToken()

	// Assert the result
	assert.Error(t, err)
	assert.Equal(t, "", token)
	assert.Contains(t, err.Error(), "received successful response, but access token field was empty")
}
