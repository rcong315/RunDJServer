package crawler

import (
	"time"

	"go.uber.org/zap"
)

// Config holds all configuration for the crawler
type Config struct {
	// Crawler behavior
	CrawlInterval time.Duration
	Workers       int

	// Monitoring
	MetricsPort string

	// Logging
	Logger *zap.Logger
}

// CrawlJobType represents different types of crawl jobs
type CrawlJobType string

const (
	JobTypeMissingAudioFeatures CrawlJobType = "missing_audio_features"
	JobTypeStaleRefresh         CrawlJobType = "stale_refresh"
	JobTypeDiscoveryArtists     CrawlJobType = "discovery_artists"
	JobTypeDiscoveryAlbums      CrawlJobType = "discovery_albums"
)

// Priority levels for crawl jobs
const (
	PriorityHigh   = 1 // Missing audio features
	PriorityMedium = 2 // Stale data refresh
	PriorityLow    = 3 // Discovery crawling
)

// CrawlJob represents a unit of work for the crawler
type CrawlJob struct {
	Type     CrawlJobType `json:"type"`
	ID       string       `json:"id"`        // Track ID, Artist ID, or Album ID
	Priority int          `json:"priority"`
	Retries  int          `json:"retries"`
	UserID   string       `json:"user_id,omitempty"` // For user-specific jobs
}

// Crawler configuration constants
const (
	// Spotify API limits
	MaxAudioFeaturesBatch = 100
	MaxTracksPerRequest   = 50
	MaxAlbumsPerRequest   = 20

	// Rate limiting
	DefaultRateLimit = 100 // requests per minute
	
	// Retry configuration
	MaxRetries      = 3
	RetryBackoffMin = 1 * time.Second
	RetryBackoffMax = 30 * time.Second

	// Data freshness
	StaleDataThreshold = 30 * 24 * time.Hour // 30 days
	ArtistCrawlInterval = 7 * 24 * time.Hour  // 7 days

	// Queue configuration
	JobQueueSize = 10000
)