package spotify

import (
	"fmt"

	"go.uber.org/zap"
)

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
	logger.Debug("Attempting to get user details")
	url := fmt.Sprintf("%s/me", spotifyAPIURL)
	logger.Debug("Fetching user details from URL", zap.String("url", url))

	responses, err := fetchAllResults[User](token, url)
	if err != nil {
		return nil, fmt.Errorf("fetching user details: %w", err)
	}
	if len(responses) == 0 || responses[0] == nil {
		return nil, fmt.Errorf("no user found")
	}

	user := responses[0]
	logger.Debug("Successfully retrieved user details",
		zap.String("userId", user.Id),
		zap.String("displayName", user.DisplayName),
		zap.String("email", user.Email))
	return user, nil
}
