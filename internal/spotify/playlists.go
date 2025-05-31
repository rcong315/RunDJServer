package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"time"

	"go.uber.org/zap"
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

func GetUsersPlaylists(token string, processor func([]*Playlist) error) error {
	logger.Debug("Attempting to get user's playlists")
	url := fmt.Sprintf("%s/me/playlists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	err := fetchAllResultsStreaming(token, url, func(response *UsersPlaylistsResponse) error {
		playlists := make([]*Playlist, len(response.Items))
		for i := range response.Items {
			playlists[i] = &response.Items[i]
		}

		if err := processor(playlists); err != nil {
			return fmt.Errorf("processing playlists batch: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching playlists: %w", err)
	}

	logger.Debug("Retrieved user's playlists")
	return nil
}

func GetPlaylistsTracks(token string, playlistId string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get tracks for playlist", zap.String("playlistId", playlistId))
	url := fmt.Sprintf("%s/playlists/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, playlistId, limitMax, 0)

	audioFeaturesBatcher := createAudioFeaturesBatcher(processor)

	err := fetchAllResultsStreaming(token, url, func(response *PlaylistsTracksResponse) error {
		for i := range response.Items {
			if err := audioFeaturesBatcher.Add(&response.Items[i].Track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching tracks for playlist %s: %w", playlistId, err)
	}

	if err := audioFeaturesBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks for playlist %s: %w", playlistId, err)
	}

	return nil
}

// TODO: Review
func CreatePlaylist(token string, userId string, bpm float64, min float64, max float64, tracks []string) (*Playlist, error) {
	logger.Debug("Attempting to create playlist for user",
		zap.String("userId", userId),
		zap.Float64("bpm", bpm),
		zap.Int("trackCount", len(tracks)))

	name := fmt.Sprintf("RunDJ %d BPM", int(math.Round(bpm)))
	description := fmt.Sprintf("This playlist was created by RunDJ. All songs in this playlist have a BPM range of %f-%f", min, max)
	public := false

	url := fmt.Sprintf("%s/users/%s/playlists", spotifyAPIURL, userId)
	logger.Debug("Create playlist request URL", zap.String("url", url))

	postData := map[string]any{
		"name":        name,
		"description": description,
		"public":      public,
	}
	jsonData, err := json.Marshal(postData)
	if err != nil {
		return nil, fmt.Errorf("marshalling create playlist data: %w", err)
	}
	logger.Debug("Create playlist request body", zap.ByteString("jsonData", jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("creating POST request for create playlist: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("sending create playlist request: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading create playlist response body: %w", err)
	}
	bodyString := string(bodyBytes)
	logger.Debug("Create playlist response body", zap.String("body", bodyString), zap.Int("statusCode", resp.StatusCode))

	if resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("creating playlist: status %d, body: %s", resp.StatusCode, bodyString)
	}

	playlist := &Playlist{}
	err = json.Unmarshal(bodyBytes, playlist)
	if err != nil {
		return nil, fmt.Errorf("unmarshalling create playlist response: %w", err)
	}
	playlistId := playlist.Id
	if playlistId == "" {
		return nil, fmt.Errorf("parsing playlist ID from response: %s", bodyString)
	}
	logger.Debug("Successfully created playlist", zap.String("playlistId", playlistId), zap.String("userId", userId), zap.String("name", name))

	// Add tracks to the created playlist
	if len(tracks) > 0 {
		logger.Debug("Adding tracks to created playlist", zap.String("playlistId", playlistId), zap.Int("trackCount", len(tracks)))
		for i := 0; i < len(tracks); i += 100 {
			var ids []string
			for j := i; j < i+100 && j < len(tracks); j++ {
				ids = append(ids, "spotify:track:"+tracks[j])
			}

			addTracksURL := fmt.Sprintf("%s/playlists/%s/tracks", spotifyAPIURL, playlistId)
			logger.Debug("Add tracks to playlist URL", zap.String("url", addTracksURL), zap.Int("batchSize", len(ids)))

			addTracksJsonData, err := json.Marshal(map[string]any{
				"uris": ids,
			})
			if err != nil {
				return playlist, fmt.Errorf("marshalling add tracks data: %w", err)
			}
			logger.Debug("Add tracks request body", zap.ByteString("jsonData", addTracksJsonData))

			addTracksReq, err := http.NewRequest("POST", addTracksURL, bytes.NewBuffer(addTracksJsonData))
			if err != nil {
				return playlist, fmt.Errorf("creating POST request for add tracks: %w", err)
			}

			addTracksReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			addTracksReq.Header.Set("Content-Type", "application/json")
			// client is already defined

			addTracksResp, err := client.Do(addTracksReq)
			if err != nil {
				return playlist, fmt.Errorf("sending add tracks request: %w", err)
			}
			defer addTracksResp.Body.Close() // Defer inside loop is okay for non-critical resources

			if addTracksResp.StatusCode != http.StatusCreated && addTracksResp.StatusCode != http.StatusOK { // Some APIs use 200 for adding items
				addTracksBodyBytes, _ := io.ReadAll(addTracksResp.Body) // Read body for error logging
				return playlist, fmt.Errorf("adding tracks to playlist %s: status %d, body: %s", playlistId, addTracksResp.StatusCode, string(addTracksBodyBytes))
			}
			logger.Debug("Successfully added batch of tracks to playlist", zap.String("playlistId", playlistId), zap.Int("batchSize", len(ids)))
		}
		logger.Debug("Finished adding all tracks to playlist", zap.String("playlistId", playlistId), zap.Int("totalTracksAdded", len(tracks)))
	}

	return playlist, nil
}
