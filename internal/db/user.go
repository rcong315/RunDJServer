package db

import (
	"context"
	"fmt"

	"go.uber.org/zap"
)

type User struct {
	UserId      string   `json:"user_id"`
	Email       string   `json:"email"`
	DisplayName string   `json:"display_name"`
	Country     string   `json:"country"`
	Followers   int      `json:"followers"`
	Product     string   `json:"product"`
	ImageURLs   []string `json:"image_urls"`
}

func SaveUser(user *User) error {
	// TODO: Save tokens
	logger.Info("Attempting to save user", zap.String("userId", user.UserId), zap.String("displayName", user.DisplayName))

	sqlQuery, err := getQueryString("insert", "user")
	if err != nil {
		logger.Error("SaveUser: Error getting query string for 'user'", zap.Error(err))
		return fmt.Errorf("error getting query string: %v", err)
	}

	db, err := getDB()
	if err != nil {
		logger.Error("SaveUser: Database connection error", zap.Error(err), zap.String("userId", user.UserId))
		return fmt.Errorf("database connection error: %v", err)
	}

	_, err = db.Exec(context.Background(), sqlQuery,
		user.UserId,
		user.Email,
		user.DisplayName,
		user.Country,
		user.Followers,
		user.Product,
		user.ImageURLs,
	)
	if err != nil {
		logger.Error("SaveUser: Error creating user record",
			zap.String("userId", user.UserId),
			zap.String("email", user.Email), // Be mindful of PII in logs, consider redacting or not logging sensitive fields
			zap.String("displayName", user.DisplayName),
			zap.Error(err))
		return fmt.Errorf("error creating user record: %v", err)
	}

	logger.Info("Successfully saved user", zap.String("userId", user.UserId))
	return nil
}
