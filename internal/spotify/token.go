package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
)

type SecretTokenResponse struct {
	AccessToken string `json:"access_token"`
}

var apiURL = os.Getenv("TOKEN_URL")

func getSecretToken() (string, error) {
	// TODO: Retry
	if apiURL == "" {
		return "", errors.New("TOKEN_URL environment variable not set")
	}
	fmt.Printf("Attempting to fetch token from %s...\n", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("http.Get failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errorBodyBytes, readErr := io.ReadAll(resp.Body)
		errorMsg := fmt.Sprintf("API request failed with status code: %d", resp.StatusCode)
		if readErr == nil && len(errorBodyBytes) > 0 {
			errorMsg += fmt.Sprintf(". Response: %s", string(errorBodyBytes))
		} else if readErr != nil {
			errorMsg += fmt.Sprintf(". Additionally, failed to read error body: %v", readErr)
		}
		return "", errors.New(errorMsg)
	}

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %w", err)
	}

	var tokenResp SecretTokenResponse
	err = json.Unmarshal(bodyBytes, &tokenResp)
	if err != nil {
		return "", fmt.Errorf("error unmarshalling JSON response: %w. Raw response: %s", err, string(bodyBytes))
	}

	if tokenResp.AccessToken == "" {
		return "", errors.New("received successful response, but access token field was empty")
	}

	fmt.Println("Successfully retrieved token.")

	return tokenResp.AccessToken, nil
}
