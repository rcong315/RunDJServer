package spotify

type User struct {
	Id          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
	Country     string `json:"country"`
	Followers   struct {
		Total int `json:"total"`
	} `json:"followers"`
	Product   string  `json:"product"`
	ImageURLs []Image `json:"images"`
}

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
	Name        string `json:"name"`
	Description string `json:"description"`
	Public      bool   `json:"public"`
	Followers   struct {
		Total int `json:"total"`
	} `json:"followers"`
	Images []Image `json:"images"`
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

type Image struct {
	URL string `json:"url"`
}

type RecommendationsResponse struct {
	Seeds []struct {
		InitialPoolSize     int    `json:"initialPoolSize"`
		AfterFilteringSize  int    `json:"afterFilteringSize"`
		AlfterRelinkingSize int    `json:"afterRelinkingSize"`
		Id                  string `json:"id"`
		Type                string `json:"type"`
		Href                string `json:"href"`
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

type UsersFollowedArtistsResponse struct {
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
	AudioFeatures []AudioFeatures `json:"audio_features"`
}

type WhoAmIResponse struct {
	Id string `json:"id"`
}

type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	Scope        string `json:"scope"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type TokenRequest struct {
	Code string `json:"code" form:"code"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}

type Config struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
	FrontendURI  string
	Port         string
}
