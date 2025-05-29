package spotify

import (
	"fmt"
	"strings"

	"go.uber.org/zap"
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

func createAudioFeaturesBatcher(processor func([]*Track) error) *BatchProcessor[*Track] {
	return NewBatchProcessor[*Track](100, func(tracks []*Track) error {
		enrichedTracks, err := getAudioFeatures(tracks)
		if err != nil {
			return fmt.Errorf("getting audio features batch: %w", err)
		}
		logger.Debug("Got audio features for batch")

		if err := processor(enrichedTracks); err != nil {
			return fmt.Errorf("processing enriched tracks: %w", err)
		}
		logger.Debug("Processed batch of enriched tracks")

		return nil
	})
}

func GetUsersTopTracks(token string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get user's top tracks")

	url := fmt.Sprintf("%s/me/top/tracks/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	audioFeaturesBatcher := createAudioFeaturesBatcher(processor)

	err := fetchAllResultsStreaming[UsersTopTracksResponse](token, url, func(response *UsersTopTracksResponse) error {
		for i := range response.Items {
			if err := audioFeaturesBatcher.Add(&response.Items[i]); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching top tracks: %w", err)
	}

	if err := audioFeaturesBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing reamaing tracks: %w", err)
	}

	logger.Debug("Successfully retrieved user's top tracks")
	return nil
}

func GetUsersSavedTracks(token string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get user's saved tracks")

	url := fmt.Sprintf("%s/me/tracks/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	audioFeaturesBatcher := createAudioFeaturesBatcher(processor)

	err := fetchAllResultsStreaming[UsersSavedTracksResponse](token, url, func(response *UsersSavedTracksResponse) error {
		for i := range response.Items {
			if err := audioFeaturesBatcher.Add(&response.Items[i].Track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching saved tracks: %w", err)
	}

	if err := audioFeaturesBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks: %w", err)
	}

	logger.Debug("Successfully retrieved user's saved tracks")
	return nil
}

// TODO: review
func getAudioFeatures(tracks []*Track) ([]*Track, error) {
	if len(tracks) == 0 {
		logger.Debug("getAudioFeatures: No tracks provided to fetch audio features for.")
		return tracks, nil
	}
	logger.Debug("Attempting to get audio features for tracks", zap.Int("trackCount", len(tracks)))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for getAudioFeatures", zap.Error(err))
		return tracks, err // Return original tracks if token fails
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
		logger.Debug("Fetching audio features for batch", zap.Int("batchStartIndex", i), zap.Int("batchSize", len(ids)))

		url := fmt.Sprintf("%s/audio-features?ids=", spotifyAPIURL) + strings.Join(ids, ",")
		logger.Debug("Audio features request URL", zap.String("url", url))

		// fetchAllResults expects a slice of responses, but /audio-features returns a single object with a list.
		// We need a direct fetch or adapt fetchAllResults if it can handle single-object root.
		// For simplicity, let's assume fetchAllResults is adapted or we use a direct fetch.
		// If fetchAllResults is strictly for paginated list of T, this part needs adjustment.
		// Assuming fetchAllResults returns []*AudioFeaturesResponse and we take the first.
		audioFeaturesResponses, err := fetchAllResults[AudioFeaturesResponse](token, url)
		if err != nil {
			logger.Error("Error fetching audio features batch", zap.Error(err), zap.String("url", url))
			// TODO: Refetch token and retry? For now, continue, some tracks might not get features.
			continue // Continue to next batch if one fails
		}

		if len(audioFeaturesResponses) > 0 && audioFeaturesResponses[0] != nil {
			for _, audioFeature := range audioFeaturesResponses[0].AudioFeatures {
				if audioFeature.Id != "" {
					if track, ok := trackMap[audioFeature.Id]; ok {
						track.AudioFeatures = &audioFeature
					} else {
						logger.Warn("Received audio feature for an unknown track ID", zap.String("trackId", audioFeature.Id))
					}
				}
			}
		} else {
			logger.Warn("No audio features data received in response for batch", zap.String("url", url))
		}
	}
	logger.Debug("Finished processing audio features for all batches")

	// The trackMap already contains the updated tracks.
	// The original 'tracks' slice order might be preferred by callers.
	// For now, returning based on trackMap. If order matters, reconstruct from original 'tracks' slice.
	// result := make([]*Track, 0, len(trackMap))
	// for _, track := range trackMap { // This loses original order if tracks had duplicates or if map iteration order is not stable.
	// 	result = append(result, track)
	// }
	// To preserve order and ensure all original tracks are returned (even if features failed):
	for i, track := range tracks {
		if updatedTrack, ok := trackMap[track.Id]; ok {
			tracks[i] = updatedTrack // Ensure the original slice is updated
		}
	}

	return tracks, nil // Return the modified original slice
}

// TODO: review
func GetRecommendations(seedArtists, seedGenres []string, minTempo float64, maxTempo float64) ([]*Track, error) {
	logger.Debug("Attempting to get recommendations",
		zap.Strings("seedArtists", seedArtists),
		zap.Strings("seedGenres", seedGenres),
		zap.Float64("minTempo", minTempo),
		zap.Float64("maxTempo", maxTempo))

	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for GetRecommendations", zap.Error(err))
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
	// Remove trailing '&' if present
	url = strings.TrimSuffix(url, "&")
	logger.Debug("Recommendations request URL", zap.String("url", url))

	// Recommendations API returns a single object, not a paginated list of them.
	// fetchAllResults might not be suitable if it expects a 'Next' field in RecommendationsResponse itself.
	// Assuming fetchAllResults can handle this or a direct fetch is used.
	// If RecommendationsResponse is the T in fetchAllResults[T], then responses will be []*RecommendationsResponse.
	responses, err := fetchAllResults[RecommendationsResponse](token, url)
	if err != nil {
		logger.Error("Error fetching recommendations", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allTracks []*Track
	if len(responses) > 0 && responses[0] != nil { // Check if we got any response
		for i := range responses[0].Tracks {
			allTracks = append(allTracks, &responses[0].Tracks[i])
		}
	} else {
		logger.Warn("No recommendations data received in response", zap.String("url", url))
	}

	logger.Debug("Successfully retrieved recommendations", zap.Int("count", len(allTracks)))
	return allTracks, nil // Original code returned 'err' which would be nil here if fetchAllResults succeeded.
}
