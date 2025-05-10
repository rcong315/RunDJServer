package spotify

import "fmt"

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

func GetArtistsAlbumsAndSingles(id string) ([]*Album, error) {
	albumsAndSingles, err := getArtistsAlbums(id, "album,single")
	if err != nil {
		return nil, err
	}

	return albumsAndSingles, nil
}

func GetArtistsCompilations(id string) ([]*Album, error) {
	return getArtistsAlbums(id, "compilation")
}

func GetArtistsAppearsOn(id string) ([]*Album, error) {
	return getArtistsAlbums(id, "appears_on")
}

func getArtistsAlbums(id string, include_groups string) ([]*Album, error) {
	token, err := getSecretToken()
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("%s/artists/%s/albums?include_groups=%s&limit=%d&offset=%d", spotifyAPIURL, id, include_groups, limitMax, 0)

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
