package db

import (
	"context"
	"fmt"
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
	sqlQuery, err := getQueryString("insert", "user")
	if err != nil {
		return fmt.Errorf("error getting query string: %v", err)
	}

	db, err := getDB()
	if err != nil {
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
		return fmt.Errorf("error creating user record: %v", err)
	}

	return nil
}
