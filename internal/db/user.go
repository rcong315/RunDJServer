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

func SaveUser(user *User) (bool, error) {
	// TODO: Save tokens
	logger.Debug("Attempting to save user", zap.String("userId", user.UserId), zap.String("displayName", user.DisplayName))

	exists, err := UserExists(user.UserId)
	if err != nil {
		return false, fmt.Errorf("error checking if user exists: %v", err)
	}

	sqlQuery, err := getQueryString("insert", "user")
	if err != nil {
		return false, fmt.Errorf("error getting query string: %v", err)
	}

	db, err := getDB()
	if err != nil {
		return false, fmt.Errorf("database connection error: %v", err)
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
		return false, fmt.Errorf("error creating user record: %v", err)
	}

	logger.Debug("Successfully saved user", zap.String("userId", user.UserId), zap.Bool("isNewUser", !exists))
	return !exists, nil
}

func UserExists(userId string) (bool, error) {
	logger.Debug("Checking if user exists", zap.String("userId", userId))

	db, err := getDB()
	if err != nil {
		return false, fmt.Errorf("database connection error: %v", err)
	}

	var count int
	err = db.QueryRow(context.Background(), `SELECT COUNT(*) FROM "user" WHERE user_id = $1`, userId).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("error checking user existence: %v", err)
	}

	return count > 0, nil
}
