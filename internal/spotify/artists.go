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

func GetUsersTopArtists(token string) ([]*Artist, error) {
	logger.Debug("Attempting to get user's top artists")
	url := fmt.Sprintf("%s/me/top/artists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersTopArtistsResponse](token, url)
	if err != nil {
		logger.Error("Error fetching user's top artists", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allArtists []*Artist
	for _, response := range responses {
		for i := range response.Items {
			allArtists = append(allArtists, &response.Items[i])
		}
	}

	logger.Debug("Successfully retrieved user's top artists", zap.Int("count", len(allArtists)))
	return allArtists, nil
}

func GetUsersFollowedArtists(token string) ([]*Artist, error) {
	logger.Debug("Attempting to get user's followed artists")
	url := fmt.Sprintf("%s/me/following?type=artist&limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	responses, err := fetchAllResults[UsersFollowedArtistsResponse](token, url)
	if err != nil {
		logger.Error("Error fetching user's followed artists", zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allArtists []*Artist
	for _, response := range responses {
		for i := range response.Artists.Items {
			allArtists = append(allArtists, &response.Artists.Items[i])
		}
	}

	logger.Debug("Successfully retrieved user's followed artists", zap.Int("count", len(allArtists)))
	return allArtists, nil
}

func GetArtistsTopTracks(artistId string) ([]*Track, error) {
	logger.Debug("Attempting to get top tracks for artist", zap.String("artistId", artistId))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for GetArtistsTopTracks", zap.String("artistId", artistId), zap.Error(err))
		return nil, err
	}

	url := fmt.Sprintf("%s/artists/%s/top-tracks", spotifyAPIURL, artistId)
	logger.Debug("Fetching artist top tracks from URL", zap.String("artistId", artistId), zap.String("url", url))

	responses, err := fetchAllResults[ArtistsTopTracksResponse](token, url)
	if err != nil {
		logger.Error("Error fetching artist top tracks", zap.String("artistId", artistId), zap.Error(err), zap.String("url", url))
		return nil, err
	}

	var allTracks []*Track
	for _, response := range responses {
		for i := range response.Tracks {
			allTracks = append(allTracks, &response.Tracks[i])
		}
	}
	logger.Debug("Successfully retrieved initial artist top tracks list", zap.String("artistId", artistId), zap.Int("count", len(allTracks)))

	allTracks, err = getAudioFeatures(allTracks)
	if err != nil {
		logger.Error("Error getting audio features for artist top tracks", zap.String("artistId", artistId), zap.Int("trackCount", len(allTracks)), zap.Error(err))
		// Return tracks even if audio features fail for some
	}

	logger.Debug("Successfully retrieved artist top tracks with audio features", zap.String("artistId", artistId), zap.Int("count", len(allTracks)))
	return allTracks, err
}

func GetArtistsAlbumsAndSingles(artistId string) ([]*Album, error) {
	logger.Debug("Attempting to get albums and singles for artist", zap.String("artistId", artistId))
	albumsAndSingles, err := getArtistsAlbums(artistId, "album,single")
	if err != nil {
		// Error already logged in getArtistsAlbums
		return nil, err
	}
	logger.Debug("Successfully retrieved albums and singles for artist", zap.String("artistId", artistId), zap.Int("count", len(albumsAndSingles)))
	return albumsAndSingles, nil
}

func GetArtistsCompilations(artistId string) ([]*Album, error) {
	logger.Debug("Attempting to get compilations for artist", zap.String("artistId", artistId))
	albums, err := getArtistsAlbums(artistId, "compilation")
	if err != nil {
		return nil, err
	}
	logger.Debug("Successfully retrieved compilations for artist", zap.String("artistId", artistId), zap.Int("count", len(albums)))
	return albums, nil
}

func GetArtistsAppearsOn(artistId string) ([]*Album, error) {
	logger.Debug("Attempting to get 'appears on' albums for artist", zap.String("artistId", artistId))
	albums, err := getArtistsAlbums(artistId, "appears_on")
	if err != nil {
		return nil, err
	}
	logger.Debug("Successfully retrieved 'appears on' albums for artist", zap.String("artistId", artistId), zap.Int("count", len(albums)))
	return albums, nil
}

func getArtistsAlbums(artistId string, include_groups string) ([]*Album, error) {
	logger.Debug("Getting artist albums (helper)", zap.String("artistId", artistId), zap.String("include_groups", include_groups))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for getArtistsAlbums",
			zap.String("artistId", artistId),
			zap.String("include_groups", include_groups),
			zap.Error(err))
		return nil, err
	}

	url := fmt.Sprintf("%s/artists/%s/albums?include_groups=%s&limit=%d&offset=%d", spotifyAPIURL, artistId, include_groups, limitMax, 0)
	logger.Debug("Fetching artist albums from URL", zap.String("artistId", artistId), zap.String("include_groups", include_groups), zap.String("url", url))

	responses, err := fetchAllResults[ArtistsAlbumsResponse](token, url)
	if err != nil {
		logger.Error("Error fetching artist albums",
			zap.String("artistId", artistId),
			zap.String("include_groups", include_groups),
			zap.Error(err),
			zap.String("url", url))
		return nil, err
	}

	var allAlbums []*Album
	for _, response := range responses {
		for i := range response.Items {
			allAlbums = append(allAlbums, &response.Items[i])
		}
	}
	logger.Debug("Successfully retrieved artist albums (helper)",
		zap.String("artistId", artistId),
		zap.String("include_groups", include_groups),
		zap.Int("count", len(allAlbums)))
	return allAlbums, nil
}

// GetUsersTopArtistsStreaming fetches user's top artists and processes each page immediately
func GetUsersTopArtistsStreaming(token string, processor func([]*Artist) error) error {
	logger.Debug("Attempting to get user's top artists (streaming)")
	url := fmt.Sprintf("%s/me/top/artists/?limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	return fetchAllResultsStreaming[UsersTopArtistsResponse](token, url, func(response *UsersTopArtistsResponse) error {
		artists := make([]*Artist, len(response.Items))
		for i := range response.Items {
			artists[i] = &response.Items[i]
		}
		logger.Debug("Processing batch of top artists", zap.Int("count", len(artists)))
		return processor(artists)
	})
}

// GetUsersFollowedArtistsStreaming fetches user's followed artists and processes each page immediately
func GetUsersFollowedArtistsStreaming(token string, processor func([]*Artist) error) error {
	logger.Debug("Attempting to get user's followed artists (streaming)")
	url := fmt.Sprintf("%s/me/following?type=artist&limit=%d&offset=%d", spotifyAPIURL, limitMax, 0)

	return fetchAllResultsStreaming[UsersFollowedArtistsResponse](token, url, func(response *UsersFollowedArtistsResponse) error {
		artists := make([]*Artist, len(response.Artists.Items))
		for i := range response.Artists.Items {
			artists[i] = &response.Artists.Items[i]
		}
		logger.Debug("Processing batch of followed artists", zap.Int("count", len(artists)))
		return processor(artists)
	})
}

// GetArtistsAlbumsAndSinglesStreaming fetches artist's albums/singles and processes each page immediately
func GetArtistsAlbumsAndSinglesStreaming(artistId string, processor func([]*Album) error) error {
	logger.Debug("Attempting to get albums and singles for artist (streaming)", zap.String("artistId", artistId))
	return getArtistsAlbumsStreaming(artistId, "album,single", processor)
}

// getArtistsAlbumsStreaming is the streaming version of getArtistsAlbums
func getArtistsAlbumsStreaming(artistId string, include_groups string, processor func([]*Album) error) error {
	logger.Debug("Getting artist albums (streaming)", zap.String("artistId", artistId), zap.String("include_groups", include_groups))
	token, err := getSecretToken()
	if err != nil {
		logger.Error("Error getting secret token for getArtistsAlbumsStreaming",
			zap.String("artistId", artistId),
			zap.String("include_groups", include_groups),
			zap.Error(err))
		return err
	}

	url := fmt.Sprintf("%s/artists/%s/albums?include_groups=%s&limit=%d&offset=%d", spotifyAPIURL, artistId, include_groups, limitMax, 0)

	return fetchAllResultsStreaming[ArtistsAlbumsResponse](token, url, func(response *ArtistsAlbumsResponse) error {
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
}
