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

func GetUsersPlaylists(token string) ([]*Playlist, error) {
	logger.Debug("Attempting to get user's playlists")
	url := fmt.Sprintf("%s/me/playlists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersPlaylistsResponse](token, url)
	if err != nil {
		logger.Error("Error fetching user's playlists", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allPlaylists []*Playlist
	for _, response := range responses {
		for i := range response.Items {
			allPlaylists = append(allPlaylists, &response.Items[i])
		}
	}

	logger.Debug("Successfully retrieved user's playlists", zap.Int("count", len(allPlaylists)))
	return allPlaylists, nil
}

func GetPlaylistsTracks(token string, playlistId string) ([]*Track, error) {
	logger.Debug("Attempting to get tracks for playlist", zap.String("playlistId", playlistId))
	url := fmt.Sprintf("%s/playlists/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, playlistId, limitMax, 0)
	logger.Debug("Fetching playlist tracks from URL", zap.String("playlistId", playlistId), zap.String("url", url))

	responses, err := fetchAllResults[PlaylistsTracksResponse](token, url)
	if err != nil {
		logger.Error("Error fetching playlist tracks", zap.String("playlistId", playlistId), zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Items {
			allTracks = append(allTracks, &response.Items[i].Track)
		}
	}
	logger.Debug("Successfully retrieved initial playlist tracks list", zap.String("playlistId", playlistId), zap.Int("count", len(allTracks)))

	allTracks, err = getAudioFeatures(allTracks)
	if err != nil {
		logger.Error("Error getting audio features for playlist tracks", zap.String("playlistId", playlistId), zap.Int("trackCount", len(allTracks)), zap.Error(err))
		// Return tracks even if audio features fail for some
	}
	logger.Debug("Successfully retrieved playlist tracks with audio features", zap.String("playlistId", playlistId), zap.Int("count", len(allTracks)))
	return allTracks, err
}

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
		logger.Error("Error marshalling create playlist data", zap.Error(err), zap.Any("postData", postData))
		return nil, err
	}
	logger.Debug("Create playlist request body", zap.ByteString("jsonData", jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		logger.Error("Error creating POST request for create playlist", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		logger.Error("Error sending create playlist request", zap.Error(err), zap.String("url", url))
		return nil, err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		logger.Error("Error reading create playlist response body", zap.Error(err))
		return nil, err
	}
	bodyString := string(bodyBytes)
	logger.Debug("Create playlist response body", zap.String("body", bodyString), zap.Int("statusCode", resp.StatusCode))

	if resp.StatusCode != http.StatusCreated {
		logger.Error("Failed to create playlist, non-201 status",
			zap.Int("statusCode", resp.StatusCode),
			zap.String("responseBody", bodyString),
			zap.String("url", url))
		return nil, fmt.Errorf("failed to create playlist: %s (status %d)", bodyString, resp.StatusCode)
	}

	playlist := &Playlist{}
	err = json.Unmarshal(bodyBytes, playlist)
	if err != nil {
		logger.Error("Error unmarshalling create playlist response", zap.Error(err), zap.String("responseBody", bodyString))
		return nil, err
	}
	playlistId := playlist.Id
	if playlistId == "" {
		logger.Error("Failed to parse playlist ID from create playlist response", zap.String("responseBody", bodyString))
		return nil, fmt.Errorf("failed to parse playlist ID from response: %s", bodyString)
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
				logger.Error("Error marshalling add tracks data", zap.Error(err), zap.String("playlistId", playlistId))
				return playlist, err // Return created playlist even if adding tracks fails partially
			}
			logger.Debug("Add tracks request body", zap.ByteString("jsonData", addTracksJsonData))

			addTracksReq, err := http.NewRequest("POST", addTracksURL, bytes.NewBuffer(addTracksJsonData))
			if err != nil {
				logger.Error("Error creating POST request for add tracks", zap.Error(err), zap.String("url", addTracksURL))
				return playlist, err
			}

			addTracksReq.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
			addTracksReq.Header.Set("Content-Type", "application/json")
			// client is already defined

			addTracksResp, err := client.Do(addTracksReq)
			if err != nil {
				logger.Error("Error sending add tracks request", zap.Error(err), zap.String("url", addTracksURL))
				return playlist, err
			}
			defer addTracksResp.Body.Close() // Defer inside loop is okay for non-critical resources

			if addTracksResp.StatusCode != http.StatusCreated && addTracksResp.StatusCode != http.StatusOK { // Some APIs use 200 for adding items
				addTracksBodyBytes, _ := io.ReadAll(addTracksResp.Body) // Read body for error logging
				logger.Error("Failed to add tracks to playlist, non-20x status",
					zap.Int("statusCode", addTracksResp.StatusCode),
					zap.ByteString("responseBody", addTracksBodyBytes),
					zap.String("url", addTracksURL))
				return playlist, fmt.Errorf("failed to add tracks to playlist: status %d, body: %s", addTracksResp.StatusCode, string(addTracksBodyBytes))
			}
			logger.Debug("Successfully added batch of tracks to playlist", zap.String("playlistId", playlistId), zap.Int("batchSize", len(ids)))
		}
		logger.Debug("Finished adding all tracks to playlist", zap.String("playlistId", playlistId), zap.Int("totalTracksAdded", len(tracks)))
	}

	return playlist, nil
}

// GetUsersPlaylistsStreaming fetches user's playlists and processes each page immediately
func GetUsersPlaylistsStreaming(token string, processor func([]*Playlist) error) error {
	logger.Debug("Attempting to get user's playlists (streaming)")
	url := fmt.Sprintf("%s/me/playlists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	return fetchAllResultsStreaming[UsersPlaylistsResponse](token, url, func(response *UsersPlaylistsResponse) error {
		playlists := make([]*Playlist, len(response.Items))
		for i := range response.Items {
			playlists[i] = &response.Items[i]
		}
		logger.Debug("Processing batch of playlists", zap.Int("count", len(playlists)))
		return processor(playlists)
	})
}

// GetPlaylistsTracksStreaming fetches playlist tracks and processes each batch with audio features
func GetPlaylistsTracksStreaming(token string, playlistId string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get tracks for playlist (streaming)", zap.String("playlistId", playlistId))
	url := fmt.Sprintf("%s/playlists/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, playlistId, limitMax, 0)

	batcher := NewBatchProcessor[*Track](100, func(tracks []*Track) error {
		enrichedTracks, err := getAudioFeatures(tracks)
		if err != nil {
			logger.Error("Error getting audio features for playlist tracks batch", 
				zap.String("playlistId", playlistId), 
				zap.Int("trackCount", len(tracks)), 
				zap.Error(err))
		}
		return processor(enrichedTracks)
	})

	err := fetchAllResultsStreaming[PlaylistsTracksResponse](token, url, func(response *PlaylistsTracksResponse) error {
		for i := range response.Items {
			if err := batcher.Add(&response.Items[i].Track); err != nil {
				return err
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return batcher.Flush()
}
