package service

import (
	"fmt"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func saveUser(user *spotify.User) error {
	if user == nil {
		logger.Error("Attempted to save a nil user")
		return fmt.Errorf("cannot save nil user")
	}
	logger.Info("Attempting to save user to DB", zap.String("spotifyUserId", user.Id), zap.String("displayName", user.DisplayName))

	dbUser := convertSpotifyUserToDBUser(user)
	err := db.SaveUser(dbUser) // db.SaveUser should have its own detailed logging
	if err != nil {
		logger.Error("Error saving user to DB",
			zap.String("spotifyUserId", user.Id),
			zap.String("dbUserId", dbUser.UserId), // dbUser.UserId might be same as spotifyUserId
			zap.Error(err))
		return fmt.Errorf("error saving user %s to DB: %w", user.Id, err)
	}

	logger.Info("Successfully saved user to DB", zap.String("spotifyUserId", user.Id), zap.String("dbUserId", dbUser.UserId))
	return nil
}
