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

func GetUsersSavedAlbums(token string, processor func([]*Album) error) error {
	logger.Debug("Attempting to get user's saved albums")
	url := fmt.Sprintf("%s/me/albums?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	err := fetchAllResultsStreaming(token, url, func(response *UsersSavedAlbumsResponse) error {
		albums := make([]*Album, len(response.Items))
		for i := range response.Items {
			albums[i] = &response.Items[i].Album
		}

		if err := processor(albums); err != nil {
			return fmt.Errorf("processing saved albums batch: %w", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching saved albums: %w", err)
	}

	logger.Debug("Retrieved user's saved albums")
	return nil
}

func GetAlbumsTracks(albumId string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get tracks for album", zap.String("albumId", albumId))
	url := fmt.Sprintf("%s/albums/%s/tracks?limit=%d&offset=%d", spotifyAPIURL, albumId, limitMax, 0)
	token, err := getSecretToken()
	if err != nil {
		return fmt.Errorf("getting secret token: %w", err)
	}

	audioFeaturesBatcher := createAudioFeaturesBatcher(processor)

	err = fetchAllResultsStreaming(token, url, func(response *AlbumsTracksResponse) error {
		for i := range response.Items {
			if err := audioFeaturesBatcher.Add(&response.Items[i]); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching tracks for album %s: %w", albumId, err)
	}

	if err := audioFeaturesBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining tracks for album %s: %w", albumId, err)
	}

	return nil
}
