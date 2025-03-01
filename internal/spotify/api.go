package spotify

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
)

type SpotifyItem struct {
	ID string `json:"id"`
}

type SpotifyTokenResponse struct {
	Items []SpotifyItem `json:"items"`
}

const spotifyAPIURL = "https://api.spotify.com/v1"
const limitMax = 50

func getUsersPlaylists(token string) []string {
	apiURL := fmt.Sprintf("%s/me/playlists", spotifyAPIURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func getPlaylistsSongs(token string, id string) []string {
	apiURL := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIURL, id)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func getUsersTopArtists(token string) []string {
	apiURL := fmt.Sprintf("%s/me/top/artists", spotifyAPIURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func getArtistsTopTracks(token string) []string {
	apiURL := fmt.Sprintf("%s/me/top/tracks", spotifyAPIURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func getArtistsAlbums(token string, id string) []string {
	apiURL := fmt.Sprintf("%s/artists/%s/albums", spotifyAPIURL, id)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func getAlbumsTracks(token string, id string) []string {
	apiURL := fmt.Sprintf("%s/albums/%s/tracks", spotifyAPIURL, id)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}

func getUsersTopTracks(token string) []string {
	apiURL := fmt.Sprintf("%s/me/top/tracks", spotifyAPIURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		log.Printf("Error creating request: %v", err)
		return []string{}
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Error making GET request: %v", err)
		return []string{}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Error from Spotify server: %d", resp.StatusCode)
		return []string{}
	}

	var spotifyResponse SpotifyTokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&spotifyResponse); err != nil {
		log.Printf("Error decoding response: %v", err)
		return []string{}
	}

	var ids []string
	for _, item := range spotifyResponse.Items {
		ids = append(ids, item.ID)
	}

	return ids
}
