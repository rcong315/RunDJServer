package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/crawler"
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		fmt.Printf("Error loading .env file: %v\n", err)
	}

	// Initialize logger
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Printf("Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize database
	if err := db.InitDB(); err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Verify Spotify configuration
	if _, err := spotify.GetConfig(); err != nil {
		logger.Fatal("Failed to load Spotify configuration", zap.Error(err))
	}

	// Create and start crawler
	crawlerInstance := crawler.New(logger)
	
	// Start the crawler
	crawlerInstance.Start()
	logger.Info("Crawler started successfully")

	// Set up graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	
	<-sigChan
	logger.Info("Shutdown signal received, stopping crawler...")
	
	crawlerInstance.Stop()
	logger.Info("Crawler stopped successfully")
}