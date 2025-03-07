package spotify

type Track struct {
	Id               string   `json:"id"`
	Name             string   `json:"name"`
	Album            Album    `json:"album"`
	Artists          []Artist `json:"artists"`
	Popularity       int      `json:"popularity"`
	DurationMS       int      `json:"duration_ms"`
	AvailableMarkets []string `json:"available_markets"`
}

type Album struct {
	Id               string   `json:"id"`
	Artists          Artist   `json:"artists"`
	TotalTracks      int      `json:"total_tracks"`
	ReleaseDate      string   `json:"release_date"`
	AvailableMarkets []string `json:"available_markets"`
	Images           []Image  `json:"images"`
}

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

type Playlist struct {
	Id    string `json:"id"`
	Owner struct {
		Id          string `json:"id"`
		DisplayName string `json:"display_name"`
	} `json:"owner"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Public      bool    `json:"public"`
	Images      []Image `json:"images"`
}

type Image struct {
	URL string `json:"url"`
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

type UsersPlaylistsResponse struct {
	Items []Playlist `json:"items"`
	Next  string     `json:"next"`
}

type PlaylistsTracksResponse struct {
	Items []UsersSavedTrackItem `json:"items"`
	Next  string                `json:"next"`
}

type UsersTopArtistsResponse struct {
	Items []Artist `json:"items"`
	Next  string   `json:"next"`
}

type UsersFollowedArtists struct {
	Artists struct {
		Items []Artist `json:"items"`
		Next  string   `json:"next"`
	} `json:"artists"`
}

type ArtistsTopTrackResponse struct {
	Tracks []Track `json:"tracks"`
}

type ArtistsAlbumsResponse struct {
	Items []Album `json:"items"`
	Next  string  `json:"next"`
}

type AlbumsTracksResponse struct {
	Items []Track `json:"items"`
	Next  string  `json:"next"`
}

type AudioFeaturesResponse struct {
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
