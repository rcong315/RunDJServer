package main

import (
	"net/http"
	"os"
	"strings"
	"time"

	ginzap "github.com/gin-contrib/zap"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func main() {
	var logger *zap.Logger
	var err error

	if os.Getenv("DEBUG") == "true" {
		logger, err = zap.NewDevelopment()
		if err != nil {
			panic(err)
		}
		err = godotenv.Load("../../.env")
		if err != nil {
			logger.Warn("Warning: .env file not found. Using system environment variables.")
		}
	} else {
		logger, err = zap.NewProduction()
		if err != nil {
			panic(err)
		}
	}
	defer logger.Sync()

	// Initialize loggers for all packages
	spotify.InitializeLogger(logger)
	db.InitializeLogger(logger)

	router := gin.New()

	router.Use(ginzap.Ginzap(logger, time.RFC3339, true))
	router.Use(ginzap.RecoveryWithZap(logger, true))
	
	// Add CORS headers for public API
	router.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// Public API endpoints - no authentication required
	router.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"service": "RunDJ Public API",
			"version": "1.0.0",
			"status":  "healthy",
		})
	})

	// Health check endpoint
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status": "healthy",
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	})

	// Audio features endpoint
	router.GET("/api/v1/tracks/audio-features", AudioFeaturesHandler)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081" // Different port from main server
		logger.Info("Defaulting to port", zap.String("port", port))
	}

	logger.Info("Public API server starting", zap.String("port", port))
	err = router.Run(":" + port)
	if err != nil {
		logger.Fatal("Failed to run public API server", zap.Error(err))
	}
}

// AudioFeaturesResponse represents the response structure for audio features
type AudioFeaturesResponse struct {
	Tracks []TrackWithAudioFeatures `json:"tracks"`
}

// TrackWithAudioFeatures represents a track with its audio features
type TrackWithAudioFeatures struct {
	ID            string             `json:"id"`
	Name          string             `json:"name"`
	Artists       []ArtistInfo      `json:"artists"`
	Album         AlbumInfo         `json:"album"`
	Popularity    int               `json:"popularity"`
	DurationMS    int               `json:"duration_ms"`
	AudioFeatures *db.AudioFeatures `json:"audio_features"`
}

// ArtistInfo represents basic artist information
type ArtistInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AlbumInfo represents basic album information
type AlbumInfo struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// AudioFeaturesHandler handles requests for track audio features
func AudioFeaturesHandler(c *gin.Context) {
	logger := zap.L() // Get the global logger
	logger.Info("AudioFeaturesHandler called")
	
	// Set API version header
	c.Header("X-API-Version", "1.0.0")

	// Get track IDs from query parameter
	idsParam := c.Query("ids")
	if idsParam == "" {
		logger.Error("AudioFeaturesHandler: Missing ids parameter")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing 'ids' parameter"})
		return
	}

	// Split and validate track IDs
	trackIds := strings.Split(idsParam, ",")
	if len(trackIds) > 10 {
		logger.Error("AudioFeaturesHandler: Too many track IDs", zap.Int("count", len(trackIds)))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 track IDs allowed"})
		return
	}

	// Clean up track IDs (remove empty strings and trim whitespace)
	var cleanIds []string
	for _, id := range trackIds {
		id = strings.TrimSpace(id)
		if id != "" && isValidSpotifyTrackId(id) {
			cleanIds = append(cleanIds, id)
		}
	}

	if len(cleanIds) == 0 {
		logger.Error("AudioFeaturesHandler: No valid track IDs provided", 
			zap.String("originalIds", idsParam))
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "No valid track IDs provided. Track IDs must be 22-character alphanumeric strings.",
		})
		return
	}

	logger.Debug("AudioFeaturesHandler: Processing track IDs", zap.Strings("trackIds", cleanIds))

	// First, try to get tracks from our database
	dbTracks, err := db.GetTracksByIds(cleanIds)
	if err != nil {
		logger.Error("AudioFeaturesHandler: Error getting tracks from database", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Database error"})
		return
	}

	// Create a map of found tracks for quick lookup
	foundTracks := make(map[string]*db.Track)
	for _, track := range dbTracks {
		foundTracks[track.TrackId] = track
	}

	// Find missing track IDs
	var missingIds []string
	for _, id := range cleanIds {
		if _, found := foundTracks[id]; !found {
			missingIds = append(missingIds, id)
		}
	}

	logger.Debug("AudioFeaturesHandler: Database lookup results", 
		zap.Int("foundInDB", len(dbTracks)), 
		zap.Int("missingFromDB", len(missingIds)))

	// If we have missing tracks, fetch them from Spotify
	var spotifyTracks []*spotify.Track
	if len(missingIds) > 0 {
		logger.Debug("AudioFeaturesHandler: Fetching missing tracks from Spotify", zap.Strings("missingIds", missingIds))
		
		spotifyTracks, err = spotify.GetTracksByIds(missingIds)
		if err != nil {
			logger.Error("AudioFeaturesHandler: Error getting tracks from Spotify", zap.Error(err))
			// Continue with what we have from the database
		} else {
			logger.Debug("AudioFeaturesHandler: Retrieved tracks from Spotify", zap.Int("count", len(spotifyTracks)))
			
			// Convert Spotify tracks to DB format and save them
			if len(spotifyTracks) > 0 {
				dbTracksToSave := convertSpotifyTracksToDBTracks(spotifyTracks)
				if err := db.SaveTracks(dbTracksToSave); err != nil {
					logger.Warn("AudioFeaturesHandler: Failed to save tracks to database", zap.Error(err))
				} else {
					logger.Debug("AudioFeaturesHandler: Saved new tracks to database", zap.Int("count", len(dbTracksToSave)))
				}

				// Add the new tracks to our found tracks map
				for _, track := range dbTracksToSave {
					foundTracks[track.TrackId] = track
				}
			}
		}
	}

	// Build response with tracks in the same order as requested
	var responseTracks []TrackWithAudioFeatures
	for _, requestedId := range cleanIds {
		if track, found := foundTracks[requestedId]; found {
			// Convert DB track to response format
			responseTrack := TrackWithAudioFeatures{
				ID:            track.TrackId,
				Name:          track.Name,
				Popularity:    track.Popularity,
				DurationMS:    track.DurationMS,
				AudioFeatures: track.AudioFeatures,
			}

			// Add artist information (simplified)
			for _, artistId := range track.ArtistIds {
				responseTrack.Artists = append(responseTrack.Artists, ArtistInfo{
					ID:   artistId,
					Name: "", // We don't have artist names in the track table
				})
			}

			// Add album information (simplified)
			if track.AlbumId != "" {
				responseTrack.Album = AlbumInfo{
					ID:   track.AlbumId,
					Name: "", // We don't have album names in the track table
				}
			}

			responseTracks = append(responseTracks, responseTrack)
		}
	}

	logger.Info("AudioFeaturesHandler: Successfully processed request", 
		zap.Int("requestedCount", len(cleanIds)), 
		zap.Int("returnedCount", len(responseTracks)))

	c.JSON(http.StatusOK, AudioFeaturesResponse{
		Tracks: responseTracks,
	})
}

// Helper function to convert Spotify tracks to DB tracks (reused from service package)
func convertSpotifyTracksToDBTracks(tracks []*spotify.Track) []*db.Track {
	var dbTracks []*db.Track
	for _, track := range tracks {
		if track == nil || track.Id == "" {
			continue
		}

		artistIds := make([]string, len(track.Artists))
		for i, artist := range track.Artists {
			artistIds[i] = artist.Id
		}

		var albumId string
		if track.Album == nil {
			albumId = ""
		} else {
			albumId = track.Album.Id
		}

		audioFeatures := track.AudioFeatures
		var dbAudioFeatures *db.AudioFeatures
		if track.AudioFeatures == nil {
			dbAudioFeatures = &db.AudioFeatures{}
		} else {
			dbAudioFeatures = &db.AudioFeatures{
				Danceability:      audioFeatures.Danceability,
				Energy:            audioFeatures.Energy,
				Key:               audioFeatures.Key,
				Loudness:          audioFeatures.Loudness,
				Mode:              audioFeatures.Mode,
				Speechiness:       audioFeatures.Speechiness,
				Acousticness:      audioFeatures.Acousticness,
				Instrumentallness: audioFeatures.Instrumentallness,
				Liveness:          audioFeatures.Liveness,
				Valence:           audioFeatures.Valence,
				Tempo:             audioFeatures.Tempo,
				Duration:          audioFeatures.Duration,
				TimeSignature:     audioFeatures.TimeSignature,
			}
		}

		dbTrack := &db.Track{
			TrackId:          track.Id,
			Name:             track.Name,
			ArtistIds:        artistIds,
			AlbumId:          albumId,
			Popularity:       track.Popularity,
			DurationMS:       track.DurationMS,
			AvailableMarkets: track.AvailableMarkets,
			AudioFeatures:    dbAudioFeatures,
			TimeSignature:    dbAudioFeatures.TimeSignature,
		}
		
		// Set BPM from audio features tempo
		if dbAudioFeatures.Tempo > 0 {
			dbTrack.BPM = dbAudioFeatures.Tempo
		}
		
		dbTracks = append(dbTracks, dbTrack)
	}

	return dbTracks
}

// isValidSpotifyTrackId validates that a string looks like a valid Spotify track ID
func isValidSpotifyTrackId(id string) bool {
	// Spotify track IDs are 22 characters long and use base62 encoding
	if len(id) != 22 {
		return false
	}
	
	// Check if all characters are valid base62 (a-z, A-Z, 0-9)
	for _, char := range id {
		if !((char >= 'a' && char <= 'z') || 
			 (char >= 'A' && char <= 'Z') || 
			 (char >= '0' && char <= '9')) {
			return false
		}
	}
	
	return true
}