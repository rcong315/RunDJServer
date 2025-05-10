package service

import (
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

//TODO: Logging and error formatting

func saveUser(user *spotify.User) error {
	dbUser := convertSpotifyUserToDBUser(user)
	return db.SaveUser(dbUser)
}
