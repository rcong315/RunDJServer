package spotify

import (
	"fmt"

	"go.uber.org/zap"
)

type Artist struct {
	Id         string   `json:"id"`
	Name       string   `json:"name"`
	Genres     []string `json:"genres"`
	Popularity int      `json:"popularity"`
	Followers  struct {
		Total int `json:"total"`
	} `json:"followers"`
	Images []Image `json:"images"`
}

type UsersTopArtistsResponse struct {
	Items []Artist `json:"items"`
	Next  string   `json:"next"`
}

type UsersFollowedArtistsResponse struct {
	Artists struct {
		Items []Artist `json:"items"`
		Next  string   `json:"next"`
	} `json:"artists"`
}

type ArtistsTopTracksResponse struct {
	Tracks []Track `json:"tracks"`
}

type ArtistsAlbumsResponse struct {
	Items []Album `json:"items"`
	Next  string  `json:"next"`
}

func GetUsersTopArtists(token string, processor func([]*Artist) error) error {
	logger.Debug("Attempting to get user's top artists")
	url := fmt.Sprintf("%s/me/top/artists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	err := fetchAllResultsStreaming(token, url, func(response *UsersTopArtistsResponse) error {
		artists := make([]*Artist, len(response.Items))
		for i := range response.Items {
			artists[i] = &response.Items[i]
		}
		logger.Debug("Processing batch of top artists", zap.Int("count", len(artists)))
		return processor(artists)
	})
	if err != nil {
		return fmt.Errorf("fetching top artists: %w", err)
	}

	logger.Debug("Retrieved user's top artists")
	return nil
}

func GetUsersFollowedArtists(token string, processor func([]*Artist) error) error {
	logger.Debug("Attempting to get user's followed artists")
	url := fmt.Sprintf("%s/me/following?type=artist&limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	err := fetchAllResultsStreaming(token, url, func(response *UsersFollowedArtistsResponse) error {
		artists := make([]*Artist, len(response.Artists.Items))
		for i := range response.Artists.Items {
			artists[i] = &response.Artists.Items[i]
		}
		logger.Debug("Processing batch of followed artists", zap.Int("count", len(artists)))
		return processor(artists)
	})
	if err != nil {
		return fmt.Errorf("fetching followed artists: %w", err)
	}

	logger.Debug("Retrieved user's followed artists")
	return nil
}

func GetArtistsAlbumsAndSingles(artistId string, processor func([]*Album) error) error {
	if err := getArtistsAlbums(artistId, "album,single", processor); err != nil {
		return fmt.Errorf("getting albums and singles for artist %s: %w", artistId, err)
	}
	return nil
}

func getArtistsAlbums(artistId string, include_groups string, processor func([]*Album) error) error {
	logger.Debug("Getting artist albums", zap.String("artistId", artistId), zap.String("include_groups", include_groups))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for getArtistsAlbums",
			zap.String("artistId", artistId),
			zap.String("include_groups", include_groups),
			zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/artists/%s/albums?include_groups=%s&limit=%d&offset=%d", spotifyAPIURL, artistId, include_groups, limitMax, 0)

	err = fetchAllResultsStreaming(token, url, func(response *ArtistsAlbumsResponse) error {
		albums := make([]*Album, len(response.Items))
		for i := range response.Items {
			albums[i] = &response.Items[i]
		}
		logger.Debug("Processing batch of artist albums",
			zap.String("artistId", artistId),
			zap.String("include_groups", include_groups),
			zap.Int("count", len(albums)))
		return processor(albums)
	})
	if err != nil {
		return fmt.Errorf("fetching albums for artist %s: %w", artistId, err)
	}

	logger.Debug("Retrieved artist albums")
	return nil
}

func GetArtistsTopTracks(artistId string, processor func([]*Track) error) error {
	logger.Debug("Attempting to get top tracks for artist",
		zap.String("artistId", artistId))
	token, err := getSecretToken()
	if err != nil {
		return fmt.Errorf("getting secret token for artist %s: %w", artistId, err)
	}

	url := fmt.Sprintf("%s/artists/%s/top-tracks", spotifyAPIURL, artistId)
	logger.Debug("Fetching artist top tracks from URL",
		zap.String("artistId", artistId),
		zap.String("url", url))

	err = fetchAllResultsStreaming(token, url, func(response *ArtistsTopTracksResponse) error {
		tracks := make([]*Track, len(response.Tracks))
		for i := range response.Tracks {
			tracks[i] = &response.Tracks[i]
		}
		logger.Debug("Processing batch of artist top tracks",
			zap.String("artistId", artistId))
		return processor(tracks)
	})
	if err != nil {
		return fmt.Errorf("fetching top tracks for artist %s: %w", artistId, err)
	}

	logger.Debug("Retrieved artist top tracks")
	return nil
}

// func GetArtistsCompilations(artistId string) ([]*Album, error) {
// 	logger.Debug("Attempting to get compilations for artist", zap.String("artistId", artistId))
// 	albums, err := getArtistsAlbums(artistId, "compilation")
// 	if err != nil {
// 		return nil, err
// 	}
// 	logger.Debug("Successfully retrieved compilations for artist", zap.String("artistId", artistId), zap.Int("count", len(albums)))
// 	return albums, nil
// }

// func GetArtistsAppearsOn(artistId string) ([]*Album, error) {
// 	logger.Debug("Attempting to get 'appears on' albums for artist", zap.String("artistId", artistId))
// 	albums, err := getArtistsAlbums(artistId, "appears_on")
// 	if err != nil {
// 		return nil, err
// 	}
// 	logger.Debug("Successfully retrieved 'appears on' albums for artist", zap.String("artistId", artistId), zap.Int("count", len(albums)))
// 	return albums, nil
// }
