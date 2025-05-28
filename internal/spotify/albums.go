package spotify

import (
	"fmt"

	"go.uber.org/zap"
)

type Album struct {
	Id               string    `json:"id"`
	Name             string    `json:"name"`
	Artists          []*Artist `json:"artists"`
	Genres           []string  `json:"genres"`
	Popularity       int       `json:"popularity"`
	AlbumType        string    `json:"album_type"`
	TotalTracks      int       `json:"total_tracks"`
	ReleaseDate      string    `json:"release_date"`
	AvailableMarkets []string  `json:"available_markets"`
	Images           []Image   `json:"images"`
}

type SavedAlbum struct {
	Album Album `json:"album"`
}

type UsersSavedAlbumsResponse struct {
	Items []SavedAlbum `json:"items"`
	Next  string       `json:"next"`
}

type AlbumsTracksResponse struct {
	Items []Track `json:"items"`
	Next  string  `json:"next"`
}

func GetUsersSavedAlbums(token string) ([]*Album, error) {
	logger.Debug("Attempting to get user's saved albums")
	url := fmt.Sprintf("%s/me/albums?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersSavedAlbumsResponse](token, url)
	if err != nil {
		logger.Error("Error fetching user's saved albums", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allAlbums []*Album
	for _, response := range responses {
		for i := range response.Items {
			allAlbums = append(allAlbums, &response.Items[i].Album)
		}
	}

	logger.Debug("Successfully retrieved user's saved albums", zap.Int("count", len(allAlbums)))
	return allAlbums, nil
}

func GetAlbumsTracks(albumId string) ([]*Track, error) {
	logger.Debug("Attempting to get tracks for album", zap.String("albumId", albumId))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for GetAlbumsTracks", zap.String("albumId", albumId), zap.Error(err))
		return nil, err
	}

	url := fmt.Sprintf("%s/albums/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, albumId, limitMax, 0)
	logger.Debug("Fetching album tracks from URL", zap.String("albumId", albumId), zap.String("url", url))

	responses, err := fetchAllResults[AlbumsTracksResponse](token, url)
	if err != nil {
		logger.Error("Error fetching album tracks", zap.String("albumId", albumId), zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Items {
			allTracks = append(allTracks, &response.Items[i])
		}
	}
	logger.Debug("Successfully retrieved initial album tracks list", zap.String("albumId", albumId), zap.Int("count", len(allTracks)))

	allTracks, err = getAudioFeatures(allTracks)
	if err != nil {
		logger.Error("Error getting audio features for album tracks", zap.String("albumId", albumId), zap.Int("trackCount", len(allTracks)), zap.Error(err))
		return allTracks, err // Return tracks even if audio features fail for some
	}

	logger.Debug("Successfully retrieved album tracks with audio features", zap.String("albumId", albumId), zap.Int("count", len(allTracks)))
	return allTracks, nil
}

// GetUsersSavedAlbumsStreaming fetches user's saved albums and processes each page immediately
func GetUsersSavedAlbumsStreaming(token string, processor func([]*Album) error) error {
	logger.Debug("Attempting to get user's saved albums (streaming)")
	url := fmt.Sprintf("%s/me/albums?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	return fetchAllResultsStreaming[UsersSavedAlbumsResponse](token, url, func(response *UsersSavedAlbumsResponse) error {
		albums := make([]*Album, len(response.Items))
		for i := range response.Items {
			albums[i] = &response.Items[i].Album
		}
		logger.Debug("Processing batch of saved albums", zap.Int("count", len(albums)))
		return processor(albums)
	})
}

// GetAlbumsTracksStreaming fetches album tracks and processes each page immediately
func GetAlbumsTracksStreaming(albumId string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get tracks for album (streaming)", zap.String("albumId", albumId))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for GetAlbumsTracksStreaming", zap.String("albumId", albumId), zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/albums/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, albumId, limitMax, 0)

	// Use audio feature batcher for tracks
	batcher := NewBatchProcessor[*Track](100, func(tracks []*Track) error {
		enrichedTracks, err := getAudioFeatures(tracks)
		if err != nil {
			logger.Error("Error getting audio features for album tracks batch",
				zap.String("albumId", albumId),
				zap.Int("trackCount", len(tracks)),
				zap.Error(err))
		}
		return processor(enrichedTracks)
	})

	err = fetchAllResultsStreaming[AlbumsTracksResponse](token, url, func(response *AlbumsTracksResponse) error {
		for i := range response.Items {
			if err := batcher.Add(&response.Items[i]); err != nil {
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
