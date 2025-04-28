package service

import (
	//TODO: improve logging, different library, log userid automatically?

	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

const spotifyAPIURL = "https://api.spotify.com/v1"

type Message struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HomeHandler(c *gin.Context) {
	log.Printf("HomeHandler called")
	c.String(http.StatusOK, "RunDJ Backend")
}

func ThanksHandler(c *gin.Context) {
	log.Printf("ThanksHandler called")
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, "<html><body><a href=\"https://getsongbpm.com\">getsongbpm.com</a></body></html>")
}

func RegisterHandler(c *gin.Context) {
	log.Printf("RegisterHandler called")
	token := c.Query("access_token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}

	user, err := spotify.GetUser(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	saveUser(user)

	processAll(token, user.Id)

	c.JSON(http.StatusOK, Message{
		Status:  "success",
		Message: "User registered successfully, processing tracks",
	})
}

func PresetPlaylistHandler(c *gin.Context) {
	log.Printf("PresetPlaylistHandler called")
	bpmStr := c.Query("bpm")
	if bpmStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}

	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}

	roundedBPM := int(math.Round(float64(bpm)/5) * 5)

	playlistId, exists := presetPlaylists[roundedBPM]
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Playlist not found for the given BPM"})
		return
	}

	c.String(http.StatusOK, playlistId)
}

func RecommendationsHandler(c *gin.Context) {
	log.Printf("RecommendationsHandler called")
	token := c.Query("access_token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}

	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user's top artists: " + err.Error(),
		})
		return
	}
	if len(usersTopArtists) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "No top artists found"})
		return
	}

	numSeedArtists := 5
	if len(usersTopArtists) < 5 {
		numSeedArtists = len(usersTopArtists)
	}

	seedArtists := make([]string, numSeedArtists)
	for i := range numSeedArtists {
		seedArtists[i] = usersTopArtists[i].Id
	}

	var seedGenres []string
	if len(seedArtists) == 0 && len(seedGenres) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing seeds"})
		return
	}

	bpmStr := c.Query("bpm")
	if bpmStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}
	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}

	minBPM := bpm - 2
	maxBPM := bpm + 2

	log.Printf("Getting recommendations with seed artists: %v, seed genres: %v, minBPM: %f, maxBPM: %f", seedArtists, seedGenres, minBPM, maxBPM)
	tracks, err := spotify.GetRecommendations(seedArtists, seedGenres, minBPM, maxBPM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting recommendations: " + err.Error(),
		})
		return
	}

	trackIds := make([]string, len(tracks))
	for i, track := range tracks {
		trackIds[i] = track.Id
	}

	c.JSON(http.StatusOK, trackIds)
}

func MatchingTracksHandler(c *gin.Context) {
	log.Printf("MatchingTracksHandler called")
	token := c.Query("access_token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}
	user, err := spotify.GetUser(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	userId := user.Id
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
		return
	}

	bpmStr := c.Param("bpm")
	if bpmStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}
	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}

	min := bpm - 1.5
	max := bpm + 1.5

	tracks, err := db.GetTracksByBPM(userId, min, max)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting tracks by BPM: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(tracks),
		"user":   userId,
		"min":    min,
		"max":    max,
		"tracks": tracks,
	})
}

func CreatePlaylistHandler(c *gin.Context) {
	log.Printf("CreatePlaylistHandler called")
	token := c.Query("access_token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}

	user, err := spotify.GetUser(token)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting user: " + err.Error(),
		})
		return
	}
	userId := user.Id
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
		return
	}

	bpmStr := c.Param("bpm")
	if bpmStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
		return
	}
	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
		return
	}

	min := bpm - 1.5
	max := bpm + 1.5

	tracks, err := db.GetTracksByBPM(userId, min, max)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error getting tracks by BPM: " + err.Error(),
		})
		return
	}

	log.Printf("Creating playlist for user %s for the bpm range %f-%f with %d songs", userId, min, max, len(tracks))
	err = spotify.CreatePlaylist(token, userId, bpm, min, max, tracks)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Error creating playlist: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, Message{
		Status:  "success",
		Message: "Playlist created successfully",
	})
}
