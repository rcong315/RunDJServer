package db

type AudioFeatures struct {
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

type Track struct {
	TrackId          string        `json:"track_id"`
	UserIds          []string      `json:"user_ids"`
	Name             string        `json:"name"`
	ArtistIds        []string      `json:"artist_ids"`
	AlbumId          string        `json:"album_id"`
	Popularity       int           `json:"popularity"`
	DurationMS       int           `json:"duration_ms"`
	AvailableMarkets []string      `json:"available_markets"`
	AudioFeatures    AudioFeatures `json:"audio_features"`
}

type Album struct {
	AlbumId          string   `json:"album_id"`
	UserIds          []string `json:"user_ids"`
	Name             string   `json:"name"`
	ArtistIds        []string `json:"artist_ids"`
	Genres           []string `json:"genres"`
	Popularity       int      `json:"popularity"`
	AlbumType        string   `json:"album_type"`
	TotalTracks      int      `json:"total_tracks"`
	ReleaseDate      string   `json:"release_date"`
	AvailableMarkets []string `json:"available_markets"`
	ImageURLs        []string `json:"image_urls"`
}

type Artist struct {
	ArtistId   string   `json:"artist_id"`
	UserIds    []string `json:"user_ids"`
	Name       string   `json:"name"`
	Genres     []string `json:"genres"`
	Popularity int      `json:"popularity"`
	Followers  int      `json:"followers"`
	ImageURLs  []string `json:"image_urls"`
}

type Playlist struct {
	PlaylistId  string   `json:"playlist_id"`
	UserIds     []string `json:"user_ids"`
	OwnerId     string   `json:"owner_id"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Public      bool     `json:"public"`
	ImageURLs   []string `json:"image_urls"`
}

type User struct {
	UserId                string   `json:"user_id"`
	Email                 string   `json:"email"`
	DisplayName           string   `json:"name"`
	Country               string   `json:"country"`
	Followers             int      `json:"followers"`
	Product               string   `json:"product"`
	ExplicitFilterEnabled int      `json:"explicit_filter"`
	ImageURLs             []string `json:"image_url"`
}
