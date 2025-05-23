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
	logger.Debug("Attempting to get user details") // Token itself should not be logged
	url := fmt.Sprintf("%s/me", spotifyAPIURL)
	logger.Debug("Fetching user details from URL", zap.String("url", url))

	// fetchAllResults expects a slice of T, but /me returns a single User object.
	// This will require fetchAllResults to be flexible or use a direct fetch method.
	// Assuming fetchAllResults can handle a single object response by wrapping it or similar.
	// If User is T in fetchAllResults[T], then responses will be []*User.
	responses, err := fetchAllResults[User](token, url)
	if err != nil {
		logger.Error("Error fetching user details", zap.Error(err), zap.String("url", url))
		return nil, err
	}
	if len(responses) == 0 || responses[0] == nil {
		logger.Error("No user found in response", zap.String("url", url))
		return nil, fmt.Errorf("no user found")
	}

	user := responses[0]
	logger.Debug("Successfully retrieved user details",
		zap.String("userId", user.Id),
		zap.String("displayName", user.DisplayName),
		zap.String("email", user.Email)) // Be mindful of PII (email)
	return user, nil
}
