package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"

	"github.com/rcong315/RunDJServer/internal/service"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func main() {
	if os.Getenv("DEBUG") == "true" {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Fatal("Warning: .env file not found. Using system environment variables.")
		}
	}

	router := gin.Default()

	router.GET("/", service.HomeHandler)
	router.GET("/thanks", service.ThanksHandler)

	router.POST("/api/spotify/auth/token", spotify.TokenHandler)
	router.POST("/api/spotify/auth/refresh", spotify.RefreshHandler)
	router.GET("/api/spotify/auth/secret", spotify.SecretTokenHandler)

	router.GET("/api/user/register", service.RegisterHandler)
	router.GET("/api/songs/preset", service.PresetPlaylistHandler)
	router.GET("/api/songs/recommendations", service.RecommendationsHandler)

	port := os.Getenv("PORT")
	log.Printf("Server starting on port %s\n", port)
	router.Run(":" + port)
}
