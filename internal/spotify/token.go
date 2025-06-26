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

	"go.uber.org/zap"
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
	logger.Debug("Attempting to fetch a new secret token")

	apiURL := os.Getenv("TOKEN_URL")
	if apiURL == "" {
		return "", time.Time{}, errors.New("TOKEN_URL environment variable not set")
	}
	url := apiURL + "/token"

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to create request for %s: %w", url, err)
	}
	req.Header.Add("Accept", "application/json")
	req.Header.Add("X-API-Key", os.Getenv("TOKEN_API_KEY"))

	resp, err := httpClient.Do(req)
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

	logger.Debug("Successfully fetched new secret token", zap.Time("expiresAt", expirationTime))
	return result.Token, expirationTime, nil
}

// GetSecretToken retrieves the cached Spotify API token or fetches a new one if needed
func GetSecretToken() (string, error) {
	return getSecretToken()
}

func getSecretToken() (string, error) {
	now := time.Now()

	// --- Fast path: Check cache with Read Lock ---
	tokenCache.RLock()
	// Check if token exists and is valid (not expired or expiring soon)
	if tokenCache.token != "" && now.Before(tokenCache.expiresAt.Add(-expirationBuffer)) {
		logger.Debug("Returning valid token from cache", zap.Time("cachedExpiresAt", tokenCache.expiresAt))
		token := tokenCache.token // Copy value while holding lock
		tokenCache.RUnlock()
		return token, nil
	}
	// Token is invalid or doesn't exist, need to potentially fetch.
	// Release read lock before attempting write lock.
	tokenCache.RUnlock()

	// Safely log token info, handling case where token might be empty or too short
	tokenInfo := "empty"
	if len(tokenCache.token) > 0 {
		if len(tokenCache.token) >= 4 {
			tokenInfo = "****" + tokenCache.token[len(tokenCache.token)-4:]
		} else {
			tokenInfo = "****" + tokenCache.token // Log full token if less than 4 chars
		}
	}

	logger.Debug("Cached token invalid, missing, or expiring soon. Attempting refresh.",
		zap.String("currentToken", tokenInfo), // Safely log token identification
		zap.Time("currentExpiresAt", tokenCache.expiresAt))

	// --- Slow path: Acquire Write Lock to Update ---
	tokenCache.Lock()
	defer tokenCache.Unlock() // Ensure unlock happens even on errors during fetch

	// !!! Double-check validity after acquiring write lock !!!
	// Another goroutine might have refreshed the token while we waited for the lock.
	now = time.Now() // Re-check current time
	if tokenCache.token != "" && now.Before(tokenCache.expiresAt.Add(-expirationBuffer)) {
		logger.Debug("Token refreshed by another goroutine while waiting for lock; returning cached token.",
			zap.Time("newCachedExpiresAt", tokenCache.expiresAt))
		return tokenCache.token, nil // Return the newly cached token
	}

	// Optional: Prevent rapid-fire retries if the last fetch failed recently
	if tokenCache.fetchErr != nil && now.Before(tokenCache.lastFetchAttempt.Add(retryCooldown)) {
		logger.Warn("Returning previous fetch error due to retry cooldown",
			zap.Time("lastAttempt", tokenCache.lastFetchAttempt),
			zap.Error(tokenCache.fetchErr))
		// Return the specific error from the last failed attempt
		return "", fmt.Errorf("token refresh failed recently, try again after %v: %w", retryCooldown, tokenCache.fetchErr)
	}

	// --- Perform the fetch ---
	newToken, newExpiresAt, err := fetchNewToken()
	tokenCache.lastFetchAttempt = time.Now() // Record attempt time regardless of outcome

	// If fetch failed, store the error and return it. Don't update token/expiry.
	if err != nil {
		tokenCache.fetchErr = err // Store the fetch error
		return "", err
	}

	// --- Success: Update cache ---
	logger.Debug("Updating token cache with newly fetched token", zap.Time("newExpiresAt", newExpiresAt))
	tokenCache.token = newToken
	tokenCache.expiresAt = newExpiresAt
	tokenCache.fetchErr = nil // Clear any previous error on success

	return tokenCache.token, nil
}
