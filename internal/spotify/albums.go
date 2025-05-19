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
	logger.Info("Attempting to get user's saved albums")
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

	logger.Info("Successfully retrieved user's saved albums", zap.Int("count", len(allAlbums)))
	return allAlbums, nil
}

func GetAlbumsTracks(albumId string) ([]*Track, error) {
	logger.Info("Attempting to get tracks for album", zap.String("albumId", albumId))
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
	logger.Info("Successfully retrieved initial album tracks list", zap.String("albumId", albumId), zap.Int("count", len(allTracks)))

	allTracks, err = getAudioFeatures(allTracks)
	if err != nil {
		logger.Error("Error getting audio features for album tracks", zap.String("albumId", albumId), zap.Int("trackCount", len(allTracks)), zap.Error(err))
		return allTracks, err // Return tracks even if audio features fail for some
	}

	logger.Info("Successfully retrieved album tracks with audio features", zap.String("albumId", albumId), zap.Int("count", len(allTracks)))
	return allTracks, nil
}
