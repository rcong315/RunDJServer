package spotify

import "fmt"

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
