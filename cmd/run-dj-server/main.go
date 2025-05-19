package main

import (
	"os"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/service"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	if os.Getenv("DEBUG") == "true" {
		err := godotenv.Load("../../.env")
		if err != nil {
			logger.Warn("Warning: .env file not found. Using system environment variables.")
		}
	}

	service.InitializeLogger(logger)
	spotify.InitializeLogger(logger)
	db.InitializeLogger(logger)

	router := gin.New()

	// TODO: Set trusted proxies

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))

	router.GET("/", service.HomeHandler)
	router.GET("/thanks", service.ThanksHandler)

	router.POST("/api/spotify/auth/token", spotify.TokenHandler)
	router.POST("/api/spotify/auth/refresh", spotify.RefreshHandler)

	router.POST("/api/user/register", service.RegisterHandler)

	router.GET("/api/songs/preset", service.PresetPlaylistHandler)
	router.GET("/api/songs/recommendations", service.RecommendationsHandler)
	router.GET("/api/songs/bpm/:bpm", service.MatchingTracksHandler)

	router.POST("/api/song/:songId/feedback", service.FeedbackHandler)

	router.POST("/api/playlist/bpm/:bpm", service.CreatePlaylistHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080" // Default port if not specified
		logger.Info("Defaulting to port", zap.String("port", port))
	}

	logger.Info("Server starting", zap.String("port", port))
	err := router.Run(":" + port)
	if err != nil {
		logger.Fatal("Failed to run server", zap.Error(err))
	}
}
