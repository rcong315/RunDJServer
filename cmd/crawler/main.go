package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/crawler"
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func main() {
	var (
		crawlInterval = flag.Duration("crawl-interval", 5*time.Minute, "Interval between crawl cycles")
		workers       = flag.Int("workers", 8, "Number of worker goroutines")
		logLevel      = flag.String("log-level", getEnv("LOG_LEVEL", "info"), "Log level (debug, info, warn, error)")
		metricsPort   = flag.String("metrics-port", getEnv("METRICS_PORT", "9090"), "Metrics server port")
	)
	flag.Parse()

	// Load environment variables
	if os.Getenv("DEBUG") == "true" {
		err := godotenv.Load("../../.env")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: .env file not found. Using system environment variables.\n")
		}
	}

	// Setup logger
	logger, err := setupLogger(*logLevel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error setting up logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize package loggers
	db.InitializeLogger(logger)
	spotify.InitializeLogger(logger)

	logger.Info("Starting RunDJ Crawler",
		zap.Duration("crawlInterval", *crawlInterval),
		zap.Int("workers", *workers),
		zap.String("logLevel", *logLevel),
		zap.String("metricsPort", *metricsPort))

	// Create crawler config
	config := &crawler.Config{
		CrawlInterval: *crawlInterval,
		Workers:       *workers,
		MetricsPort:   *metricsPort,
		Logger:        logger,
	}

	// Create and start crawler
	c, err := crawler.New(config)
	if err != nil {
		logger.Fatal("Failed to create crawler", zap.Error(err))
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle shutdown signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
	}()

	// Start crawler
	logger.Info("Starting crawler...")
	if err := c.Start(ctx); err != nil {
		logger.Fatal("Crawler failed", zap.Error(err))
	}

	logger.Info("Crawler shutdown complete")
}

func setupLogger(level string) (*zap.Logger, error) {
	var config zap.Config

	if level == "debug" {
		config = zap.NewDevelopmentConfig()
	} else {
		config = zap.NewProductionConfig()
	}

	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}

	return config.Build()
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}