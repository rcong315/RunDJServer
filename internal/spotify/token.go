package spotify

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

type SecretTokenResponse struct {
	Token     string `json:"access_token"`
	ExpiresMs int64  `json:"expiration"`
}

// Cache structure holding token, expiration, and mutex
var tokenCache struct {
	sync.RWMutex     // Embed RWMutex for read/write locking
	token            string
	expiresAt        time.Time
	lastFetchAttempt time.Time // Optional: Track last attempt for error cooldown
	fetchErr         error     // Store the last fetch error
}

// expirationBuffer defines how close to expiration we trigger a refresh.
const expirationBuffer = 60 * time.Second // Refresh if expires within 60 seconds

// retryCooldown defines minimum time before retrying after a failed fetch (optional)
const retryCooldown = 15 * time.Second

func fetchNewToken() (string, time.Time, error) {
	fmt.Println("Attempting to fetch a new secret token...")

	apiURL := os.Getenv("TOKEN_URL")
	if apiURL == "" {
		return "", time.Time{}, errors.New("TOKEN_URL environment variable not set")
	}
	url := apiURL + "/token"

	client := http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Add("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to execute request to %s: %w", url, err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to read response body (status %d) from %s: %w", resp.StatusCode, url, err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", time.Time{}, fmt.Errorf("received non-OK status code %d (%s) from %s: %s",
			resp.StatusCode, http.StatusText(resp.StatusCode), url, string(bodyBytes))
	}

	var result SecretTokenResponse
	err = json.Unmarshal(bodyBytes, &result)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to unmarshal token JSON response from %s: %w. Body: %s", url, err, string(bodyBytes))
	}
	if result.Token == "" {
		return "", time.Time{}, fmt.Errorf("parsed access token is empty in response from %s. Body: %s", url, string(bodyBytes))
	}
	if result.ExpiresMs <= 0 {
		return "", time.Time{}, fmt.Errorf("invalid expiration timestamp %d received from %s", result.ExpiresMs, url)
	}

	expirationTime := time.UnixMilli(result.ExpiresMs)

	fmt.Printf("Successfully fetched new token. Expires: %s\n", expirationTime.Format(time.RFC3339))
	return result.Token, expirationTime, nil
}

func getSecretToken() (string, error) {
	now := time.Now()

	// --- Fast path: Check cache with Read Lock ---
	tokenCache.RLock()
	// Check if token exists and is valid (not expired or expiring soon)
	if tokenCache.token != "" && now.Before(tokenCache.expiresAt.Add(-expirationBuffer)) {
		fmt.Println("Returning valid token from cache.")
		token := tokenCache.token // Copy value while holding lock
		tokenCache.RUnlock()
		return token, nil
	}
	// Token is invalid or doesn't exist, need to potentially fetch.
	// Release read lock before attempting write lock.
	tokenCache.RUnlock()
	fmt.Println("Cached token invalid, missing, or expiring soon.")

	// --- Slow path: Acquire Write Lock to Update ---
	tokenCache.Lock()
	defer tokenCache.Unlock() // Ensure unlock happens even on errors during fetch

	// !!! Double-check validity after acquiring write lock !!!
	// Another goroutine might have refreshed the token while we waited for the lock.
	now = time.Now() // Re-check current time
	if tokenCache.token != "" && now.Before(tokenCache.expiresAt.Add(-expirationBuffer)) {
		fmt.Println("Token refreshed by another goroutine while waiting for lock; returning cached token.")
		return tokenCache.token, nil // Return the newly cached token
	}

	// Optional: Prevent rapid-fire retries if the last fetch failed recently
	if tokenCache.fetchErr != nil && now.Before(tokenCache.lastFetchAttempt.Add(retryCooldown)) {
		fmt.Printf("Returning previous fetch error due to retry cooldown (last attempt: %s)\n", tokenCache.lastFetchAttempt.Format(time.RFC3339))
		// Return the specific error from the last failed attempt
		return "", fmt.Errorf("token refresh failed recently, try again after %v: %w", retryCooldown, tokenCache.fetchErr)
	}

	// --- Perform the fetch ---
	newToken, newExpiresAt, err := fetchNewToken()
	tokenCache.lastFetchAttempt = time.Now() // Record attempt time regardless of outcome

	// If fetch failed, store the error and return it. Don't update token/expiry.
	if err != nil {
		fmt.Printf("Failed to fetch new token: %v\n", err)
		tokenCache.fetchErr = err // Store the fetch error
		// Keep the potentially expired token/expiry info as is? Or clear them?
		// Let's keep them but return the error.
		return "", err
	}

	// --- Success: Update cache ---
	fmt.Println("Updating token cache with newly fetched token.")
	tokenCache.token = newToken
	tokenCache.expiresAt = newExpiresAt
	tokenCache.fetchErr = nil // Clear any previous error on success

	return tokenCache.token, nil
}

// func getSecretToken() (string, error) {
// 	// TODO: Retr if failed
// 	if apiURL == "" {
// 		return "", errors.New("TOKEN_URL environment variable not set")
// 	}
// 	fmt.Printf("Attempting to fetch token from %s\n", apiURL)

// 	resp, err := http.Get(apiURL)
// 	if err != nil {
// 		return "", fmt.Errorf("http.Get failed: %w", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		errorBodyBytes, readErr := io.ReadAll(resp.Body)
// 		errorMsg := fmt.Sprintf("API request failed with status code: %d", resp.StatusCode)
// 		if readErr == nil && len(errorBodyBytes) > 0 {
// 			errorMsg += fmt.Sprintf(". Response: %s", string(errorBodyBytes))
// 		} else if readErr != nil {
// 			errorMsg += fmt.Sprintf(". Additionally, failed to read error body: %v", readErr)
// 		}
// 		return "", errors.New(errorMsg)
// 	}

// 	bodyBytes, err := io.ReadAll(resp.Body)
// 	if err != nil {
// 		return "", fmt.Errorf("error reading response body: %w", err)
// 	}

// 	var tokenResp SecretTokenResponse
// 	err = json.Unmarshal(bodyBytes, &tokenResp)
// 	if err != nil {
// 		return "", fmt.Errorf("error unmarshalling JSON response: %w. Raw response: %s", err, string(bodyBytes))
// 	}

// 	if tokenResp.token == "" {
// 		return "", errors.New("received successful response, but access token field was empty")
// 	}

// 	fmt.Println("Successfully retrieved token.")

// 	return tokenResp.token, nil
// }
