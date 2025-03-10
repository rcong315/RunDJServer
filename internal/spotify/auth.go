package spotify

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
)

const (
	spotifyAuthURL  = "https://accounts.spotify.com/authorize"
	spotifyTokenURL = "https://accounts.spotify.com/api/token"
)

func AuthHandler(c *gin.Context) {
	config, err := GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Configuration error: " + err.Error()})
		return
	}

	state := generateRandomString(16)
	scope := "user-read-private user-read-email user-read-playback-state user-modify-playback-state user-read-currently-playing"

	params := url.Values{}
	params.Add("response_type", "code")
	params.Add("client_id", config.ClientID)
	params.Add("scope", scope)
	params.Add("redirect_uri", config.RedirectURI)
	params.Add("state", state)
	params.Add("show_dialog", "true")

	authURL := spotifyAuthURL + "?" + params.Encode()
	c.Redirect(http.StatusFound, authURL)
}

func CallbackHandler(c *gin.Context) {
	config, err := GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Configuration error: " + err.Error()})
		return
	}

	code := c.Query("code")
	state := c.Query("state")

	if state == "" {
		c.Redirect(http.StatusFound, config.FrontendURI+"/#error=state_mismatch")
		return
	}

	// Prepare token request data
	data := url.Values{}
	data.Set("code", code)
	data.Set("redirect_uri", config.RedirectURI)
	data.Set("grant_type", "authorization_code")

	// Make request to Spotify token API
	tokenResponse, err := makeTokenRequest(config, data)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		c.Redirect(http.StatusFound, config.FrontendURI+"/#error=invalid_token")
		return
	}

	// Redirect to frontend with tokens
	redirectURL := fmt.Sprintf("%s/#access_token=%s&refresh_token=%s&expires_in=%d",
		config.FrontendURI,
		tokenResponse.AccessToken,
		tokenResponse.RefreshToken,
		tokenResponse.ExpiresIn)

	c.Redirect(http.StatusFound, redirectURL)
}

func TokenHandler(c *gin.Context) {
	config, err := GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Configuration error: " + err.Error()})
		return
	}

	var tokenRequest TokenRequest
	if err := c.ShouldBindJSON(&tokenRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if tokenRequest.Code == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Authorization code is required"})
		return
	}

	// Prepare token request data
	data := url.Values{}
	data.Set("code", tokenRequest.Code)
	data.Set("redirect_uri", config.RedirectURI)
	data.Set("grant_type", "authorization_code")

	// Make request to Spotify token API
	tokenResponse, err := makeTokenRequest(config, data)
	if err != nil {
		log.Printf("Token exchange error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to get token"})
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

func RefreshHandler(c *gin.Context) {
	config, err := GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Configuration error: " + err.Error()})
		return
	}

	var refreshRequest RefreshRequest
	if err := c.ShouldBindJSON(&refreshRequest); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Invalid request body"})
		return
	}

	if refreshRequest.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Refresh token is required"})
		return
	}

	// Prepare refresh token request data
	data := url.Values{}
	data.Set("refresh_token", refreshRequest.RefreshToken)
	data.Set("grant_type", "refresh_token")

	// Make request to Spotify token API
	tokenResponse, err := makeTokenRequest(config, data)
	if err != nil {
		log.Printf("Token refresh error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Failed to refresh token"})
		return
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// makeTokenRequest sends a request to the Spotify token API
func makeTokenRequest(config *Config, data url.Values) (*TokenResponse, error) {
	// Create authorization header
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(config.ClientID+":"+config.ClientSecret))

	// Create request
	req, err := http.NewRequest("POST", spotifyTokenURL, bytes.NewBufferString(data.Encode()))
	if err != nil {
		return nil, err
	}

	// Set headers
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Content-Length", fmt.Sprintf("%d", len(data.Encode())))
	req.Header.Add("Authorization", authHeader)

	// Send request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Check for error status
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("spotify API returned %d: %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResponse TokenResponse
	err = json.Unmarshal(body, &tokenResponse)
	if err != nil {
		return nil, err
	}

	return &tokenResponse, nil
}
