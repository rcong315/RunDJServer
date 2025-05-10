package spotify

import "fmt"

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

type WhoAmIResponse struct {
	Id string `json:"id"`
}

func GetUser(token string) (*User, error) {
	url := fmt.Sprintf("%s/me", spotifyAPIURL)
	response, err := fetchAllResults[User](token, url)
	if err != nil {
		return nil, err
	}
	if len(response) == 0 {
		return nil, fmt.Errorf("no user found")
	}
	return response[0], nil
}
