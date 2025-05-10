package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"
)

type Playlist struct {
	Id    string `json:"id"`
	Owner struct {
		Id          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Followers   struct {
		Total int `json:"total"`
	} `json:"followers"`
	Images []Image `json:"images"`
}

type UsersPlaylistsResponse struct {
	Items []Playlist `json:"items"`
	Next  string     `json:"next"`
}

type PlaylistsTracksResponse struct {
	Items []UsersSavedTrackItem `json:"items"`
	Next  string                `json:"next"`
}

func GetUsersPlaylists(token string) ([]*Playlist, error) {
	url := fmt.Sprintf("%s/me/playlists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersPlaylistsResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allPlaylists []*Playlist
	for _, response := range responses {
		for i := range response.Items {
			allPlaylists = append(allPlaylists, &response.Items[i])
		}
	}

	return allPlaylists, nil
}

func GetPlaylistsTracks(token string, id string) ([]*Track, error) {
	url := fmt.Sprintf("%s/playlists/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, id, limitMax, 0)

	responses, err := fetchAllResults[PlaylistsTracksResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Items {
			allTracks = append(allTracks, &response.Items[i].Track)
		}
	}

	allTracks, err = getAudioFeatures(allTracks)
	return allTracks, err
}

func CreatePlaylist(token string, userId string, bpm float64, min float64, max float64, tracks []string) (*Playlist, error) {
	// TODO: Check if playlist already exists

	name := fmt.Sprintf("RunDJ %d BPM", int(math.Round(bpm)))
	description := fmt.Sprintf("This playlist was created by RunDJ. All songs in this playlist have a BPM range of %f-%f", min, max)
	public := false

	url := fmt.Sprintf("%s/users/%s/playlists", spotifyAPIURL, userId)

	postData := map[string]any{
		"name":        name,
		"description": description,
		"public":      public,
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create playlist: %s", resp.Status)
	}

	playlist := &Playlist{}
	err = json.Unmarshal(bodyBytes, playlist)
	if err != nil {
		return nil, err
	}
	playlistId := playlist.Id
	if playlistId == "" {
		return nil, fmt.Errorf("failed to parse playlist ID from response: %s", bodyString)
	}

	for i := 0; i < len(tracks); i += 100 {
		var ids []string
		for j := i; j < i+100 && j < len(tracks); j++ {
			ids = append(ids, "spotify:track:"+tracks[j])
		}

		url := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIURL, playlistId)

		jsonData, err := json.Marshal(map[string]any{
			"uris": ids,
		})
		if err != nil {
			return playlist, err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return playlist, err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return playlist, err
		}
		if resp.StatusCode != http.StatusCreated {
			return playlist, fmt.Errorf("failed to add tracks to playlist: %s", resp.Status)
		}
	}

	return playlist, nil
}
