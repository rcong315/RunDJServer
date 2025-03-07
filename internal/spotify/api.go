package spotify

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"
)

const spotifyAPIURL = "https://api.spotify.com/v1/"
const limitMax = 50

func getNextURL(response interface{}) string {
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

func fetchPaginatedItems(token string, url string, responseType interface{}) (interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return nil, err
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d for URL: %s", resp.StatusCode, url)
		if resp.StatusCode == http.StatusTooManyRequests {
			retryAfter := 60
			if retryHeader := resp.Header.Get("Retry-After"); retryHeader != "" {
				if seconds, err := strconv.Atoi(retryHeader); err == nil {
					retryAfter = seconds
				}
			}
			log.Printf("Rate limited, waiting %d seconds", retryAfter)
			time.Sleep(time.Duration(retryAfter) * time.Second)
		}
		return nil, fmt.Errorf("server returned status code %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(responseType); err != nil {
		log.Printf("Error decoding response: %v", err)
		return nil, err
	}
	return responseType, nil
}

func fetchAllResults(token string, url string, responseType interface{}) ([]interface{}, error) {
	var results []interface{}
	for {
		response, err := fetchPaginatedItems(token, url, responseType)
		if err != nil {
			return results, err
		}
		results = append(results, response)

		nextURL := getNextURL(response)
		if nextURL == "" {
			break
		}
		url = nextURL
	}
	return results, nil
}

func getUsersTopTracks(token string) ([]Track, error) {
	var allTracks []Track
	url := fmt.Sprintf("%sme/top/tracks/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)
	responseType := &UsersTopTracksResponse{}

	responses, err := fetchAllResults(token, url, responseType)
	if err != nil {
		return nil, err
	}
	for _, response := range responses {
		if typedResponse, ok := response.(*UsersTopTracksResponse); ok {
			allTracks = append(allTracks, typedResponse.Items...)
		}
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
