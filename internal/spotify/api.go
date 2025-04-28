package spotify

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"
	"time"
)

var retryLimits = []int{20, 10, 5}

// TODO: decorator, check access token before every handler

func GetUser(token string) (*User, error) {
	url := fmt.Sprintf("%s/me", spotifyAPIURL)
	response, err := fetchAllResults[User](token, url)
	if err != nil {
		return nil, err
	}
	if len(response) == 0 {
		return nil, fmt.Errorf("no user found")
	}
	return response[0], nil
}

func GetRecommendations(seedArtists, seedGenres []string, minTempo float64, maxTempo float64) ([]*Track, error) {
	token, err := getSecretToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/recommendations?limit=%d&", spotifyAPIURL, 100)
	if len(seedArtists) > 0 {
		url += "seed_artists=" + strings.Join(seedArtists, ",") + "&"
	}
	if len(seedGenres) > 0 {
		url += "seed_genres=" + strings.Join(seedGenres, ",") + "&"
	}
	if minTempo > 0 {
		url += fmt.Sprintf("min_tempo=%f&", minTempo)
	}
	if maxTempo > 0 {
		url += fmt.Sprintf("max_tempo=%f", maxTempo)
	}

	response, err := fetchAllResults[RecommendationsResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allTracks []*Track
	for _, response := range response {
		for i := range response.Tracks {
			allTracks = append(allTracks, &response.Tracks[i])
		}
	}

	// allTracks, err = getAudioFeatures(allTracks)
	return allTracks, err
}

// TODO: All get track functions do the same code with different url, create a helper function

func GetUsersTopTracks(token string) ([]*Track, error) {
	url := fmt.Sprintf("%s/me/top/tracks/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersTopTracksResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Items {
			allTracks = append(allTracks, &response.Items[i])
		}
	}

	allTracks, err = getAudioFeatures(allTracks)
	return allTracks, err
}

func GetUsersSavedTracks(token string) ([]*Track, error) {
	url := fmt.Sprintf("%s/me/tracks/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersSavedTracksResponse](token, url)
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

func GetUsersTopArtists(token string) ([]*Artist, error) {
	url := fmt.Sprintf("%s/me/top/artists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersTopArtistsResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allArtists []*Artist
	for _, response := range responses {
		for i := range response.Items {
			allArtists = append(allArtists, &response.Items[i])
		}
	}

	return allArtists, nil
}

func GetUsersFollowedArtists(token string) ([]*Artist, error) {
	url := fmt.Sprintf("%s/me/following?type=artist&limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersFollowedArtistsResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allArtists []*Artist
	for _, response := range responses {
		for i := range response.Artists.Items {
			allArtists = append(allArtists, &response.Artists.Items[i])
		}
	}

	return allArtists, nil
}

func GetUsersSavedAlbums(token string) ([]*Album, error) {
	url := fmt.Sprintf("%s/me/albums?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersSavedAlbumsResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allAlbums []*Album
	for _, response := range responses {
		for i := range response.Items {
			allAlbums = append(allAlbums, &response.Items[i].Album)
		}
	}

	return allAlbums, nil
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

func GetArtistsTopTracks(id string) ([]*Track, error) {
	token, err := getSecretToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/artists/%s/top-tracks", spotifyAPIURL, id)

	responses, err := fetchAllResults[ArtistsTopTracksResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Tracks {
			allTracks = append(allTracks, &response.Tracks[i])
		}
	}

	allTracks, err = getAudioFeatures(allTracks)
	return allTracks, err
}

func GetArtistsAlbums(id string) ([]*Album, error) {
	token, err := getSecretToken()
	if err != nil {
		return nil, err
	}

	albumTypes := "album,single"
	url := fmt.Sprintf("%s/artists/%s/albums?include_groups=%s&limit=%d&offset=%d", spotifyAPIURL, id, albumTypes, limitMax, 0)

	responses, err := fetchAllResults[ArtistsAlbumsResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allAlbums []*Album
	for _, response := range responses {
		for i := range response.Items {
			allAlbums = append(allAlbums, &response.Items[i])
		}
	}

	return allAlbums, nil
}

func GetAlbumsTracks(id string) ([]*Track, error) {
	token, err := getSecretToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/albums/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, id, limitMax, 0)

	responses, err := fetchAllResults[AlbumsTracksResponse](token, url)
	if err != nil {
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Items {
			allTracks = append(allTracks, &response.Items[i])
		}
	}

	allTracks, err = getAudioFeatures(allTracks)
	return allTracks, err
}

func getAudioFeatures(tracks []*Track) ([]*Track, error) {
	token, err := getSecretToken()
	if err != nil {
		return nil, err
	}

	trackMap := make(map[string]*Track)
	for _, track := range tracks {
		trackMap[track.Id] = track
	}

	for i := 0; i < len(tracks); i += 100 { // Iterate in batches of 100
		var ids []string
		for j := i; j < i+100 && j < len(tracks); j++ {
			ids = append(ids, tracks[j].Id)
		}

		url := fmt.Sprintf("%s/audio-features?ids=", spotifyAPIURL) + strings.Join(ids, ",")

		responses, err := fetchAllResults[AudioFeaturesResponse](token, url)
		if err != nil {
			// TODO: Refetch token and retry
			return nil, err
		}

		for _, audioFeatures := range responses[0].AudioFeatures {
			if audioFeatures.Id != "" {
				id := audioFeatures.Id
				track := trackMap[id]
				track.AudioFeatures = &audioFeatures
				trackMap[id] = track
			}
		}
	}

	result := make([]*Track, 0, len(trackMap))
	for _, track := range trackMap {
		result = append(result, track)
	}

	return result, nil
}

func CreatePlaylist(token string, userId string, bpm float64, min float64, max float64, tracks []string) error {
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
		return err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{
		Timeout: 30 * time.Second,
	}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	bodyString := string(bodyBytes)

	if resp.StatusCode != http.StatusCreated {
		return fmt.Errorf("failed to create playlist: %s", resp.Status)
	}

	var playlistResponse PlaylistResponse
	err = json.Unmarshal(bodyBytes, &playlistResponse)
	if err != nil {
		return err
	}
	playlistId := playlistResponse.Id
	if playlistId == "" {
		return fmt.Errorf("failed to parse playlist ID from response: %s", bodyString)
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
			return err
		}

		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			return err
		}

		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))
		req.Header.Set("Content-Type", "application/json")
		client := &http.Client{
			Timeout: 30 * time.Second,
		}

		resp, err := client.Do(req)
		if err != nil {
			return err
		}
		if resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("failed to add tracks to playlist: %s", resp.Status)
		}
	}

	return nil
}
