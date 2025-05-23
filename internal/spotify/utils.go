package spotify

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

var (
	config         *Config
	configOnce     sync.Once
	configErr      error
	httpClient     *http.Client
	httpClientOnce sync.Once
)

const (
	spotifyAPIURL = "https://api.spotify.com/v1"
	limitMax      = 50
	maxRetries    = 3
)

var retryLimits = []int{20, 10, 5}

func init() {
	httpClientOnce.Do(func() {
		transport := &http.Transport{
			MaxIdleConns:        100,
			IdleConnTimeout:     90 * time.Second,
			MaxIdleConnsPerHost: 10,
		}
		httpClient = &http.Client{
			Transport: transport,
			Timeout:   30 * time.Second,
		}
	})
}

type Image struct {
	URL string `json:"url"`
}

type Config struct {
	ClientId     string
	ClientSecret string
	RedirectURI  string
	FrontendURI  string
	Port         string
}

func GetConfig() (*Config, error) {
	configOnce.Do(func() {
		config, configErr = loadConfig()
	})
	return config, configErr
}

func loadConfig() (*Config, error) {
	config := &Config{
		ClientId:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("REDIRECT_URI"),
		Port:         os.Getenv("PORT"),
	}

	if config.ClientId == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	return config, nil
}

func MapTracksToArray(m map[string]*Track) []*Track {
	arr := make([]*Track, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func MapAlbumsToArray(m map[string]*Album) []*Album {
	arr := make([]*Album, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func MapArtistsToArray(m map[string]*Artist) []*Artist {
	arr := make([]*Artist, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func MapPlaylistsToArray(m map[string]*Playlist) []*Playlist {
	arr := make([]*Playlist, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func getNextURL(response any) string {
	switch r := response.(type) {
	case *UsersTopTracksResponse:
		return r.Next
	case *UsersSavedTracksResponse:
		return r.Next
	case *UsersPlaylistsResponse:
		return r.Next
	case *PlaylistsTracksResponse:
		return r.Next
	case *UsersTopArtistsResponse:
		return r.Next
	case *UsersFollowedArtistsResponse:
		return r.Artists.Next
	case *ArtistsTopTracksResponse:
		return ""
	case *ArtistsAlbumsResponse:
		return r.Next
	case *AlbumsTracksResponse:
		return r.Next
	default:
		return ""
	}
}

func modifyURLLimit(apiURL string, newLimit int) string {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return apiURL
	}

	query := parsedURL.Query()
	query.Set("limit", strconv.Itoa(newLimit))
	parsedURL.RawQuery = query.Encode()

	return parsedURL.String()
}

func fetchPaginatedItems[T any](token string, url string) (*T, error) {
	// TODO: Manage rate limiting
	var lastErr error
	currentLimit := limitMax

	for attempt := range maxRetries {
		if attempt > 0 {
			// Exponential backoff
			waitTime := time.Duration(attempt*3) * time.Second
			logger.Info("Retrying request",
				zap.Int("attempt", attempt+1),
				zap.Duration("waitTime", waitTime),
				zap.String("url", url))
			time.Sleep(waitTime)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			logger.Error("Error creating HTTP request", zap.Error(err), zap.String("url", url))
			lastErr = err
			continue
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		resp, err := httpClient.Do(req)
		if err != nil {
			logger.Error("Error making GET request", zap.Error(err), zap.String("url", url))
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close() // Close the body before processing errors

			logger.Error("Error response from Spotify server",
				zap.Int("statusCode", resp.StatusCode),
				zap.String("url", url))

			// Handle rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := 60
				// if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				// 	if seconds, err := strconv.Atoi(retryHeader); err == nil {
				// 		retryAfter = seconds
				// 	}
				// }
				logger.Warn("Rate limited by Spotify",
					zap.Int("retryAfterSeconds", retryAfter),
					zap.String("url", url))
				time.Sleep(time.Duration(retryAfter) * time.Second)
				lastErr = fmt.Errorf("rate limited, retrying after %d seconds", retryAfter)
				continue
			} else if resp.StatusCode == http.StatusBadGateway {
				// For 502 errors, try reducing the limit
				if strings.Contains(url, "limit=") && currentLimit > 10 {
					currentLimit = retryLimits[attempt] // Be careful with index out of bounds if maxRetries > len(retryLimits)
					logger.Warn("Got a 502 error, reducing request limit",
						zap.Int("newLimit", currentLimit),
						zap.String("url", url))
					url = modifyURLLimit(url, currentLimit)
					lastErr = fmt.Errorf("reduced limit to %d after 502 error", currentLimit)
					continue
				}
			} else if resp.StatusCode >= 500 {
				// Other server errors might be temporary
				lastErr = fmt.Errorf("server returned status code %d, may retry", resp.StatusCode)
				continue
			} else {
				// Client errors (4xx) except 429 are likely not recoverable with retries
				return nil, fmt.Errorf("server returned status code %d", resp.StatusCode)
			}
		} else {
			// Success path
			defer resp.Body.Close()
			var result T
			if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
				logger.Error("Error decoding Spotify response body", zap.Error(err), zap.String("url", url))
				lastErr = err
				continue
			}

			return &result, nil
		}
	}

	// If we get here, all retries failed
	logger.Error("All Spotify request attempts failed",
		zap.Int("attempts", maxRetries),
		zap.String("url", url),
		zap.Error(lastErr))
	return nil, fmt.Errorf("all %d attempts failed, last error: %v", maxRetries, lastErr)
}

func fetchAllResults[T any](token string, initialURL string) ([]*T, error) {
	var results []*T
	url := initialURL
	for {
		response, err := fetchPaginatedItems[T](token, url)
		if err != nil {
			break
		}
		results = append(results, response)

		url = getNextURL(response)
		if url == "" {
			break
		}
	}
	return results, nil
}
