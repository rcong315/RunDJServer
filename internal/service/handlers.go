package service

import (
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type Message struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HomeHandler(c *gin.Context) {
	logger.Info("HomeHandler called", zap.String("path", c.Request.URL.Path))
	c.String(http.StatusOK, "RunDJ Backend")
}

func RegisterHandler(c *gin.Context) {
	// TODO: check when last updated to see if need to run processAll again
	logger.Info("RegisterHandler called")
	token := c.Query("access_token")
	if token == "" {
		logger.Error("RegisterHandler: Missing access_token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}

	user, err := spotify.GetUser(token)
	if err != nil {
		logger.Error("RegisterHandler: Error getting user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	logger.Debug("RegisterHandler: User retrieved", zap.String("userId", user.Id), zap.String("displayName", user.DisplayName))
	saveUser(user) // Assuming saveUser has its own logging if necessary

	processAll(token, user.Id) // Assuming processAll has its own logging

	logger.Info("RegisterHandler: User registered successfully, processing tracks", zap.String("userId", user.Id))
	c.JSON(http.StatusOK, Message{
		Status:  "success",
		Message: "User registered successfully, processing tracks",
	})
}

func PresetPlaylistHandler(c *gin.Context) {
	logger.Info("PresetPlaylistHandler called")
	bpmStr := c.Query("bpm")
	if bpmStr == "" {
		logger.Error("PresetPlaylistHandler: Missing bpm")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}

	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		logger.Error("PresetPlaylistHandler: Invalid bpm", zap.String("bpmStr", bpmStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}

	roundedBPM := int(math.Round(float64(bpm)/5) * 5)
	logger.Debug("PresetPlaylistHandler: BPM processed", zap.Float64("originalBPM", bpm), zap.Int("roundedBPM", roundedBPM))

	playlistId, exists := presetPlaylists[roundedBPM]
	if !exists {
		logger.Error("PresetPlaylistHandler: Playlist not found for BPM", zap.Int("roundedBPM", roundedBPM))
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found for the given BPM"})
		return
	}

	logger.Info("PresetPlaylistHandler: Playlist found", zap.Int("roundedBPM", roundedBPM), zap.String("playlistId", playlistId))
	c.String(http.StatusOK, playlistId)
}

func RecommendationsHandler(c *gin.Context) {
	logger.Info("RecommendationsHandler called")
	token := c.Query("access_token")
	if token == "" {
		logger.Error("RecommendationsHandler: Missing access_token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}

	// Attempt to get user ID for logging, even if not strictly required by the handler's logic yet
	// This helps in correlating logs if an error occurs early.
	var userIdForLogging string
	userForLog, errUser := spotify.GetUser(token)
	if errUser == nil && userForLog != nil {
		userIdForLogging = userForLog.Id
	}

	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		logger.Error("RecommendationsHandler: Error getting user's top artists", zap.String("userId", userIdForLogging), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user's top artists: " + err.Error(),
		})
		return
	}
	if len(usersTopArtists) == 0 {
		logger.Error("RecommendationsHandler: No top artists found", zap.String("userId", userIdForLogging))
		c.JSON(http.StatusBadRequest, gin.H{"error": "No top artists found"})
		return
	}
	logger.Debug("RecommendationsHandler: User's top artists retrieved", zap.String("userId", userIdForLogging), zap.Int("count", len(usersTopArtists)))

	numSeedArtists := 5
	if len(usersTopArtists) < 5 {
		numSeedArtists = len(usersTopArtists)
	}

	seedArtists := make([]string, numSeedArtists)
	for i := range numSeedArtists {
		seedArtists[i] = usersTopArtists[i].Id
	}

	var seedGenres []string // Assuming seedGenres might be populated from elsewhere in a future state
	if len(seedArtists) == 0 && len(seedGenres) == 0 {
		logger.Error("RecommendationsHandler: Missing seeds", zap.String("userId", userIdForLogging))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing seeds"})
		return
	}

	bpmStr := c.Query("bpm")
	if bpmStr == "" {
		logger.Error("RecommendationsHandler: Missing bpm", zap.String("userId", userIdForLogging))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}
	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		logger.Error("RecommendationsHandler: Invalid bpm", zap.String("userId", userIdForLogging), zap.String("bpmStr", bpmStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}

	minBPM := bpm - 2
	maxBPM := bpm + 2

	logger.Debug("RecommendationsHandler: Getting recommendations",
		zap.String("userId", userIdForLogging),
		zap.Strings("seedArtists", seedArtists),
		zap.Strings("seedGenres", seedGenres),
		zap.Float64("minBPM", minBPM),
		zap.Float64("maxBPM", maxBPM))
	tracks, err := spotify.GetRecommendations(seedArtists, seedGenres, minBPM, maxBPM)
	if err != nil {
		logger.Error("RecommendationsHandler: Error getting recommendations", zap.String("userId", userIdForLogging), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting recommendations: " + err.Error(),
		})
		return
	}

	trackIds := make([]string, len(tracks))
	for i, track := range tracks {
		trackIds[i] = track.Id
	}
	logger.Info("RecommendationsHandler: Recommendations retrieved", zap.String("userId", userIdForLogging), zap.Int("count", len(trackIds)))
	c.JSON(http.StatusOK, trackIds)
}

func MatchingTracksHandler(c *gin.Context) {
	logger.Info("MatchingTracksHandler called")
	token := c.Query("access_token")
	if token == "" {
		logger.Error("MatchingTracksHandler: Missing access_token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}
	user, err := spotify.GetUser(token)
	if err != nil {
		logger.Error("MatchingTracksHandler: Error getting user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	userId := user.Id
	if userId == "" {
		// This case should ideally not be reached if GetUser returns a user object
		logger.Error("MatchingTracksHandler: Missing userId after GetUser call")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
		return
	}
	logger.Debug("MatchingTracksHandler: User identified", zap.String("userId", userId))

	bpmStr := c.Param("bpm")
	if bpmStr == "" {
		logger.Error("MatchingTracksHandler: Missing bpm", zap.String("userId", userId))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}
	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		logger.Error("MatchingTracksHandler: Invalid bpm", zap.String("userId", userId), zap.String("bpmStr", bpmStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}
	min := bpm - 1.5
	max := bpm + 1.5
	logger.Debug("MatchingTracksHandler: BPM parameters set", zap.String("userId", userId), zap.Float64("targetBPM", bpm), zap.Float64("minBPM", min), zap.Float64("maxBPM", max))

	sourcesStr := c.Query("sources")
	sources := strings.Split(sourcesStr, ",")
	logger.Debug("MatchingTracksHandler: Sources for tracks", zap.String("userId", userId), zap.Strings("sources", sources))

	tracks, err := db.GetTracksByBPM(userId, min, max, sources)
	if err != nil {
		logger.Error("MatchingTracksHandler: Error getting tracks by BPM", zap.String("userId", userId), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting tracks by BPM: " + err.Error(),
		})
		return
	}
	logger.Info("MatchingTracksHandler: Tracks retrieved by BPM", zap.String("userId", userId), zap.Int("count", len(tracks)))

	c.JSON(http.StatusOK, gin.H{
		"count":  len(tracks),
		"user":   userId,
		"min":    min,
		"max":    max,
		"tracks": tracks,
	})
}

func CreatePlaylistHandler(c *gin.Context) {
	logger.Info("CreatePlaylistHandler called")
	token := c.Query("access_token")
	if token == "" {
		logger.Error("CreatePlaylistHandler: Missing access_token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}
	user, err := spotify.GetUser(token)
	if err != nil {
		logger.Error("CreatePlaylistHandler: Error getting user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	userId := user.Id
	if userId == "" {
		logger.Error("CreatePlaylistHandler: Missing userId after GetUser call")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
		return
	}
	logger.Debug("CreatePlaylistHandler: User identified", zap.String("userId", userId))

	bpmStr := c.Param("bpm")
	if bpmStr == "" {
		logger.Error("CreatePlaylistHandler: Missing bpm", zap.String("userId", userId))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}
	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		logger.Error("CreatePlaylistHandler: Invalid bpm", zap.String("userId", userId), zap.String("bpmStr", bpmStr), zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}
	min := bpm - 1.5
	max := bpm + 1.5
	logger.Debug("CreatePlaylistHandler: BPM parameters set", zap.String("userId", userId), zap.Float64("targetBPM", bpm), zap.Float64("minBPM", min), zap.Float64("maxBPM", max))

	sourcesStr := c.Query("sources")
	sources := strings.Split(sourcesStr, ",")
	logger.Debug("CreatePlaylistHandler: Sources for tracks", zap.String("userId", userId), zap.Strings("sources", sources))

	tracks, err := db.GetTracksByBPM(userId, min, max, sources)
	if err != nil {
		logger.Error("CreatePlaylistHandler: Error getting tracks by BPM", zap.String("userId", userId), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting tracks by BPM: " + err.Error(),
		})
		return
	}
	var ids []string
	for key := range tracks {
		ids = append(ids, key)
	}
	logger.Debug("CreatePlaylistHandler: Tracks for playlist retrieved", zap.String("userId", userId), zap.Int("count", len(ids)))

	logger.Debug("Creating playlist",
		zap.String("userId", userId),
		zap.Float64("minBPM", min),
		zap.Float64("maxBPM", max),
		zap.Int("songCount", len(tracks)))
	playlist, err := spotify.CreatePlaylist(token, userId, bpm, min, max, ids)
	if err != nil || playlist.Id == "" { // Check playlist.Id as well, as CreatePlaylist might return partial success
		logger.Error("CreatePlaylistHandler: Error creating playlist", zap.String("userId", userId), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error creating playlist: " + err.Error(),
		})
		return
	}
	logger.Info("CreatePlaylistHandler: Playlist created successfully", zap.String("userId", userId), zap.String("playlistId", playlist.Id))
	c.JSON(http.StatusOK, playlist)
}

func FeedbackHandler(c *gin.Context) {
	logger.Info("FeedbackHandler called")
	token := c.Query("access_token")
	if token == "" {
		logger.Error("FeedbackHandler: Missing access_token")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}
	user, err := spotify.GetUser(token)
	if err != nil {
		logger.Error("FeedbackHandler: Error getting user", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	userId := user.Id
	if userId == "" {
		logger.Error("FeedbackHandler: Missing userId after GetUser call")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
		return
	}
	logger.Debug("FeedbackHandler: User identified", zap.String("userId", userId))

	songId := c.Param("songId")
	if songId == "" {
		logger.Error("FeedbackHandler: Missing songId", zap.String("userId", userId))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing songId"})
		return
	}
	logger.Debug("FeedbackHandler: Song identified", zap.String("userId", userId), zap.String("songId", songId))

	feedback := c.Query("feedback")
	if feedback == "" {
		logger.Error("FeedbackHandler: Missing feedback", zap.String("userId", userId), zap.String("songId", songId))
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing feedback"})
		return
	}
	var feedbackInt int
	switch feedback {
	case "LIKE":
		feedbackInt = 1
	case "DISLIKE":
		feedbackInt = -1
	default:
		feedbackInt = 0 // Or handle as an error if feedback must be LIKE/DISLIKE
		logger.Warn("FeedbackHandler: Invalid feedback value", zap.String("userId", userId), zap.String("songId", songId), zap.String("feedbackValue", feedback))
	}
	logger.Debug("FeedbackHandler: Feedback processed", zap.String("userId", userId), zap.String("songId", songId), zap.String("feedback", feedback), zap.Int("feedbackInt", feedbackInt))

	err = db.SaveFeedback(userId, songId, feedbackInt) // Assuming SaveFeedback has its own logging if necessary
	if err != nil {
		logger.Error("FeedbackHandler: Error saving feedback", zap.String("userId", userId), zap.String("songId", songId), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error saving feedback: " + err.Error(),
		})
		return
	}
	logger.Info("FeedbackHandler: Feedback saved successfully", zap.String("userId", userId), zap.String("songId", songId))
	c.JSON(http.StatusOK, Message{
		Status:  "success",
		Message: "Feedback saved successfully",
	})
}
