package spotify

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const spotifyAPIURL = "https://api.spotify.com/v1/"
const limitMax = 50
const maxRetries = 3

var retryLimits = []int{20, 10, 5}

func fetchPaginatedItems[T any](token string, url string) (*T, error) {
	var lastErr error
	currentLimit := limitMax

	for attempt := range maxRetries {
		if attempt > 0 {
			// Exponential backoff
			waitTime := time.Duration(attempt*3) * time.Second
			log.Printf("Retrying request (attempt %d) after %v", attempt+1, waitTime)
			time.Sleep(waitTime)
		}

		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			lastErr = err
			continue
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error making GET request: %v", err)
			lastErr = err
			continue
		}

		if resp.StatusCode != http.StatusOK {
			resp.Body.Close() // Close the body before processing errors

			log.Printf("Error from Spotify server: %d for URL: %s", resp.StatusCode, url)

			// Handle rate limiting
			if resp.StatusCode == http.StatusTooManyRequests {
				retryAfter := 60
				if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
					if seconds, err := strconv.Atoi(retryHeader); err == nil {
						retryAfter = seconds
					}
				}
				log.Printf("Rate limited, waiting %d seconds", retryAfter)
				time.Sleep(time.Duration(retryAfter) * time.Second)
				lastErr = fmt.Errorf("rate limited, retrying after %d seconds", retryAfter)
				continue
			} else if resp.StatusCode == http.StatusBadGateway {
				// For 502 errors, try reducing the limit
				if strings.Contains(url, "limit=") && currentLimit > 10 {
					currentLimit = retryLimits[attempt]
					log.Printf("Got a 502 error, reducing limit to %d", currentLimit)
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
				log.Printf("Error decoding response: %v", err)
				lastErr = err
				continue
			}

			return &result, nil
		}
	}

	// If we get here, all retries failed
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

func GetAudioFeatures(token string, tracks []Track) ([]Track, error) {
	trackMap := make(map[string]Track)
	for _, track := range tracks {
		trackMap[track.Id] = track
	}

	for i := 0; i < len(tracks); i += 100 { // Iterate in batches of 100
		var ids []string
		for j := i; j < i+100 && j < len(tracks); j++ {
			ids = append(ids, tracks[j].Id)
		}

		url := fmt.Sprintf("%saudio-features?ids=", spotifyAPIURL) + strings.Join(ids, ",")

		responses, err := fetchAllResults[AudioFeaturesResponse](token, url)
		if err != nil {
			return nil, err
		}

		for _, audioFeatures := range responses[0].AudioFeatures {
			id := audioFeatures.Id
			track := trackMap[id]
			track.AudioFeatures = audioFeatures
			trackMap[id] = track
		}
	}

	result := make([]Track, 0, len(trackMap))
	for _, track := range trackMap {
		result = append(result, track)
	}

	return result, nil
}

func GetUsersTopTracks(token string) ([]Track, error) {
	url := fmt.Sprintf("%sme/top/tracks/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersTopTracksResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allTracks []Track
	for _, response := range responses {
		allTracks = append(allTracks, response.Items...)
	}

	return allTracks, nil
}

// func getUsersSavedTracks(token string) ([]UsersSavedTracksResponse, error) {
// 	var response []UsersSavedTracksResponse
// 	if err := fetchPaginatedItems(token, "me/tracks", limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getUsersPlaylists(token string) ([]UsersPlaylistsResponse, error) {
// 	var response []UsersPlaylistsResponse
// 	if err := fetchPaginatedItems(token, "me/playlists", limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getPlaylistsTracks(token, id string) ([]PlaylistsTracksResponse, error) {
// 	var response []PlaylistsTracksResponse
// 	apiURL := fmt.Sprintf("playlists/%s/tracks", id)
// 	if err := fetchPaginatedItems(token, apiURL, limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getUsersTopArtists(token string) ([]UsersTopArtistsResponse, error) {
// 	var response []UsersTopArtistsResponse
// 	if err := fetchPaginatedItems(token, "me/top/artists", limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getUsersFollowedArtists(token string) ([]UsersFollowedArtists, error) {
// 	var response []UsersFollowedArtists
// 	if err := fetchPaginatedItems(token, "me/following?type=artist", limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getArtistsTopTracks(token, id string) ([]ArtistsTopTrackResponse, error) {
// 	var response []ArtistsTopTrackResponse
// 	apiURL := fmt.Sprintf("artists/%s/top-tracks", id)
// 	if err := fetchPaginatedItems(token, apiURL, limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getArtistsAlbums(token, id string) ([]ArtistsAlbumsResponse, error) {
// 	var response []ArtistsAlbumsResponse
// 	apiURL := fmt.Sprintf("artists/%s/albums", id)
// 	if err := fetchPaginatedItems(token, apiURL, limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func getAlbumsTracks(token, id string) ([]AlbumsTracksResponse, error) {
// 	var response []AlbumsTracksResponse
// 	apiURL := fmt.Sprintf("albums/%s/tracks", id)
// 	if err := fetchPaginatedItems(token, apiURL, limitMax, 0, &response); err != nil {
// 		return nil, err
// 	}
// 	return response, nil
// }

// func GetAudioFeatures(token string, ids []string) []AudioFeaturesResponse {
// 	var audioFeatures []AudioFeaturesResponse
// 	var url = fmt.Sprintf("%s/audio-features?ids=%s", spotifyAPIURL, ids[0])
// 	for i := 1; i < len(ids); i++ {
// 		url = fmt.Sprintf("%s,%s", url, ids[i])
// 	}

// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		log.Printf("Error creating request: %v", err)
// 		return audioFeatures
// 	}
// 	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		log.Printf("Error making GET request: %v", err)
// 		return audioFeatures
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		log.Printf("Error from Spotify server: %d for URL: %s", resp.StatusCode, url)
// 		return audioFeatures
// 	}

// 	var response struct {
// 		AudioFeatures []AudioFeaturesResponse `json:"audio_features"`
// 	}
// 	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
// 		log.Printf("Error decoding response: %v", err)
// 		return audioFeatures
// 	}

// 	return response.AudioFeatures
// }
