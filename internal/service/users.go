package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func saveUser(user *spotify.User) (bool, error) {
	if user == nil {
		return false, fmt.Errorf("cannot save nil user")
	}
	logger.Debug("Attempting to save user to DB", zap.String("spotifyUserId", user.Id), zap.String("displayName", user.DisplayName))

	dbUser := convertSpotifyUserToDBUser(user)
	isNewUser, err := db.SaveUser(dbUser)
	if err != nil {
		return false, fmt.Errorf("error saving user %s to DB: %w", user.Id, err)
	}

	logger.Debug("Successfully saved user to DB", zap.String("spotifyUserId", user.Id), zap.String("dbUserId", dbUser.UserId), zap.Bool("isNewUser", isNewUser))
	return isNewUser, nil
}
