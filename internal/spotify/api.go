package spotify

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

const spotifyAPIURL = "https://api.spotify.com/v1/"
const limitMax = 50

func fetchPaginatedItems(token string, apiURL string, limit int, offset int, responseType interface{}) []string {
	var ids []string

	url := fmt.Sprintf("%s%s?limit=%d&offset=%d", spotifyAPIURL, apiURL, limit, offset)

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			log.Printf("Error creating request: %v", err)
			return ids
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			log.Printf("Error making GET request: %v", err)
			return ids
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			log.Printf("Error from Spotify server: %d for URL: %s", resp.StatusCode, url)
			if resp.StatusCode == http.StatusTooManyRequests {
				log.Printf("Rate limited, waiting 1 minute")
				time.Sleep(60 * time.Second)
				continue
			}
			return ids
		}

		if err := json.NewDecoder(resp.Body).Decode(responseType); err != nil {
			log.Printf("Error decoding response: %v", err)
			return ids
		}

		var nextURL string
		var itemCount int

		switch response := responseType.(type) {
		case *UsersPlaylistsResponse:
			for _, item := range response.Items {
				ids = append(ids, item.Id)
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *UsersTopArtistsResponse:
			for _, item := range response.Items {
				ids = append(ids, item.Id)
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *ArtistsAlbumsResponse:
			for _, item := range response.Items {
				ids = append(ids, item.Id)
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *PlaylistsTracksResponse:
			for _, item := range response.Items {
				if item.Track.Id != "" {
					ids = append(ids, item.Track.Id)
				}
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *UsersTopTracksResponse:
			for _, item := range response.Items {
				ids = append(ids, item.Id)
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *UsersSavedTracksResponse:
			for _, item := range response.Items {
				if item.Track.Id != "" {
					ids = append(ids, item.Track.Id)
				}
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *AlbumsTracksResponse:
			for _, item := range response.Items {
				ids = append(ids, item.Id)
			}
			itemCount = len(response.Items)
			nextURL = response.Next
		case *UsersFollowedArtists:
			for _, item := range response.Artists.Items {
				ids = append(ids, item.Id)
			}
			itemCount = len(response.Artists.Items)
			nextURL = response.Artists.Next
		case *ArtistsTopTrackResponse:
			for _, track := range response.Tracks {
				ids = append(ids, track.Id)
			}
			itemCount = len(response.Tracks)
		default:
			log.Printf("Unknown response type: %T", responseType)
			return ids
		}

		if itemCount == 0 || nextURL == "" {
			return ids
		}

		url = nextURL
	}
}

func getUsersTopTracks(token string) []string {
	apiURL := "me/top/tracks"
	var response UsersTopTracksResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getUsersSavedTracks(token string) []string {
	apiURL := "me/tracks"
	var response UsersSavedTracksResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getUsersPlaylists(token string) []string {
	apiURL := "me/playlists"
	var response UsersPlaylistsResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getPlaylistsTracks(token, id string) []string {
	apiURL := fmt.Sprintf("playlists/%s/tracks", id)
	var response PlaylistsTracksResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getUsersTopArtists(token string) []string {
	apiURL := "me/top/artists?"
	var response UsersTopArtistsResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getUsersFollowedArtists(token string) []string {
	apiURL := "me/following?type=artist"
	var response UsersFollowedArtists
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getArtistsTopTracks(token, id string) []string {
	apiURL := fmt.Sprintf("artists/%s/top-tracks", id)
	var response ArtistsTopTrackResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getArtistsAlbums(token, id string) []string {
	apiURL := fmt.Sprintf("artists/%s/albums", id)
	var response ArtistsAlbumsResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func getAlbumsTracks(token, id string) []string {
	apiURL := fmt.Sprintf("albums/%s/tracks", id)
	var response AlbumsTracksResponse
	return fetchPaginatedItems(token, apiURL, limitMax, 0, &response)
}

func GetAudioFeatures(token string, ids []string) []AudioFeaturesResponse {
	var audioFeatures []AudioFeaturesResponse
	var url = fmt.Sprintf("%s/audio-features?ids=%s", spotifyAPIURL, ids[0])
	for i := 1; i < len(ids); i++ {
		url = fmt.Sprintf("%s,%s", url, ids[i])
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return audioFeatures
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return audioFeatures
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d for URL: %s", resp.StatusCode, url)
		return audioFeatures
	}

	var response struct {
		AudioFeatures []AudioFeaturesResponse `json:"audio_features"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		log.Printf("Error decoding response: %v", err)
		return audioFeatures
	}

	return response.AudioFeatures
}
