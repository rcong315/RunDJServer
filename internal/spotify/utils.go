package spotify

import (
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"sync"
	"time"
)

var (
	config     *Config
	configOnce sync.Once
	configErr  error
)

func GetConfig() (*Config, error) {
	configOnce.Do(func() {
		config, configErr = loadConfig()
	})
	return config, configErr
}

func loadConfig() (*Config, error) {
	// err := godotenv.Load()
	// if err != nil {
	// 	log.Println("Warning: .env file not found. Using system environment variables.")
	// }

	config := &Config{
		ClientID:     os.Getenv("SPOTIFY_CLIENT_ID"),
		ClientSecret: os.Getenv("SPOTIFY_CLIENT_SECRET"),
		RedirectURI:  os.Getenv("REDIRECT_URI"),
		FrontendURI:  os.Getenv("FRONTEND_URI"),
		Port:         os.Getenv("PORT"),
	}

	if config.ClientID == "" || config.ClientSecret == "" {
		return nil, fmt.Errorf("SPOTIFY_CLIENT_ID and SPOTIFY_CLIENT_SECRET must be set")
	}

	return config, nil
}

func MapTracksToArray(m map[string]Track) []Track {
	arr := make([]Track, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func MapAlbumsToArray(m map[string]Album) []Album {
	arr := make([]Album, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func MapArtistsToArray(m map[string]Artist) []Artist {
	arr := make([]Artist, 0, len(m))
	for _, v := range m {
		arr = append(arr, v)
	}
	return arr
}

func MapPlaylistsToArray(m map[string]Playlist) []Playlist {
	arr := make([]Playlist, 0, len(m))
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
	case *UsersFollowedArtists:
		return r.Artists.Next
	case *ArtistsTopTrackResponse:
		return "" // No pagination for top tracks
	case *ArtistsAlbumsResponse:
		return r.Next
	case *AlbumsTracksResponse:
		return r.Next
	default:
		return ""
	}
}

func extractOffsetAndLimit(apiURL string) (int, int) {
	parsedURL, err := url.Parse(apiURL)
	if err != nil {
		return 0, limitMax
	}

	query := parsedURL.Query()

	offset := 0
	if offsetStr := query.Get("offset"); offsetStr != "" {
		if parsedOffset, err := strconv.Atoi(offsetStr); err == nil {
			offset = parsedOffset
		}
	}

	limit := limitMax
	if limitStr := query.Get("limit"); limitStr != "" {
		if parsedLimit, err := strconv.Atoi(limitStr); err == nil {
			limit = parsedLimit
		}
	}

	return offset, limit
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

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	seededRand := rand.New(rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}
