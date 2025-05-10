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
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	spotifyAuthURL  = "https://accounts.spotify.com/authorize"
	spotifyTokenURL = "https://accounts.spotify.com/api/token"
)

type TokenRequest struct {
	Code string `json:"code" form:"code"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type TokenResponse struct {
	Token        string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

func TokenHandler(c *gin.Context) {
	log.Printf("TokenHandler: Processing request from %s", c.ClientIP())

	config, err := GetConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: "Configuration error: " + err.Error()})
		return
	}

	var tokenRequest TokenRequest
	if err := c.ShouldBind(&tokenRequest); err != nil {
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
	tokenResponse, err := makeTokenRequest(config.ClientId, config.ClientSecret, data)
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

	contentType := c.GetHeader("Content-Type")
	log.Printf("Content-Type: %s", contentType)

	// Get refresh token directly from form data first
	refreshToken := c.PostForm("refresh_token")

	// If not found in form, read and log the raw body
	if refreshToken == "" {
		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			log.Printf("Error reading request body: %v", err)
			c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Could not read request body"})
			return
		}

		// Log the raw body for debugging
		bodyStr := string(bodyBytes)
		log.Printf("Raw request body: %s", bodyStr)

		// Restore the body for later use
		c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

		// Try to parse the body as form data
		formValues, err := url.ParseQuery(bodyStr)
		if err != nil {
			log.Printf("Error parsing form data: %v", err)
		} else {
			refreshToken = formValues.Get("refresh_token")
			log.Printf("Found refresh_token in parsed form: %s", refreshToken)
		}

		// As a fallback, try to search for the refresh token in the body string
		if refreshToken == "" && strings.Contains(bodyStr, "refresh_token=") {
			parts := strings.Split(bodyStr, "refresh_token=")
			if len(parts) > 1 {
				tokenPart := parts[1]
				// If there are other parameters, cut at the &
				if ampIndex := strings.Index(tokenPart, "&"); ampIndex != -1 {
					refreshToken = tokenPart[:ampIndex]
				} else {
					refreshToken = tokenPart
				}
				log.Printf("Extracted refresh_token from body: %s", refreshToken)
			}
		}
	}

	// Check if we have a refresh token
	if refreshToken == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{Error: "Refresh token is required"})
		return
	}

	// Prepare refresh token request data
	data := url.Values{}
	data.Set("refresh_token", refreshToken)
	data.Set("grant_type", "refresh_token")

	// Make request to Spotify token API
	tokenResponse, err := makeTokenRequest(config.ClientId, config.ClientSecret, data)
	if err != nil {
		log.Printf("Token refresh error: %v", err)
		c.JSON(http.StatusInternalServerError, ErrorResponse{Error: fmt.Sprintf("Failed to refresh token: %v", err)})
		return
	}

	// If the response doesn't include a refresh token, add the one we used
	if tokenResponse.RefreshToken == "" {
		tokenResponse.RefreshToken = refreshToken
	}

	c.JSON(http.StatusOK, tokenResponse)
}

// makeTokenRequest sends a request to the Spotify token API
func makeTokenRequest(clientId string, clientSecret string, data url.Values) (*TokenResponse, error) {
	// Create authorization header
	authHeader := "Basic " + base64.StdEncoding.EncodeToString([]byte(clientId+":"+clientSecret))

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
