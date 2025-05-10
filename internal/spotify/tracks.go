package spotify

import (
	"fmt"
	"strings"
)

type Track struct {
	Id               string         `json:"id"`
	Name             string         `json:"name"`
	Album            *Album         `json:"album"`
	Artists          []*Artist      `json:"artists"`
	Popularity       int            `json:"popularity"`
	DurationMS       int            `json:"duration_ms"`
	AvailableMarkets []string       `json:"available_markets"`
	AudioFeatures    *AudioFeatures `json:"audio_features"`
}

type AudioFeatures struct {
	Id                string  `json:"id"`
	Danceability      float64 `json:"danceability"`
	Energy            float64 `json:"energy"`
	Key               int     `json:"key"`
	Loudness          float64 `json:"loudness"`
	Mode              int     `json:"mode"`
	Speechiness       float64 `json:"speechiness"`
	Acousticness      float64 `json:"acousticness"`
	Instrumentallness float64 `json:"instrumentallness"`
	Liveness          float64 `json:"liveness"`
	Valence           float64 `json:"valence"`
	Tempo             float64 `json:"tempo"`
	Duration          int     `json:"duration_ms"`
	TimeSignature     int     `json:"time_signature"`
}

type RecommendationsResponse struct {
	Seeds []struct {
		InitialPoolSize    int    `json:"initialPoolSize"`
		AfterFilteringSize int    `json:"afterFilteringSize"`
		AfterRelinkingSize int    `json:"afterRelinkingSize"`
		Id                 string `json:"id"`
		Type               string `json:"type"`
		Href               string `json:"href"`
	} `json:"seeds"`
	Tracks []Track `json:"tracks"`
}

type UsersSavedTrackItem struct {
	Track Track `json:"track"`
}

type UsersTopTracksResponse struct {
	Items []Track `json:"items"`
	Next  string  `json:"next"`
}

type UsersSavedTracksResponse struct {
	Items []UsersSavedTrackItem `json:"items"`
	Next  string                `json:"next"`
}

type AudioFeaturesResponse struct {
	AudioFeatures []AudioFeatures `json:"audio_features"`
}

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

	return allTracks, err
}
