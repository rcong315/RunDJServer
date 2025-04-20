package service

import (
	//TODO: improve logging, different library, log userid automatically?
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

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
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting user: " + err.Error()})
		return
	}
	saveUser(user)

	go func(token, userId string) {
		saveAllTracks(token, userId)
	}(token, user.Id)

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
	seedArtists := c.QueryArray("seed_artists")
	seedGenres := c.QueryArray("seed_genres")
	if len(seedArtists) == 0 && len(seedGenres) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing seed_artists or seed_genres"})
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

	tracks, err := spotify.GetRecommendations(seedArtists, seedGenres, minBPM, maxBPM)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting recommendations: " + err.Error()})
		return
	}

	trackIds := make([]string, len(tracks))
	for i, track := range tracks {
		trackIds[i] = track.Id
	}

	c.JSON(http.StatusOK, trackIds)
}

// func MatchingTracksHandler(c *gin.Context) {
// 	log.Printf("MatchingTracksHandler called")
// 	token := c.Query("access_token")
// 	if token == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
// 		return
// 	}
// 	user, err := spotify.GetUser(token)
// 	if err != nil {
// 		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting user: " + err.Error()})
// 		return
// 	}
// 	userId := user.Id
// 	if userId == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing userId"})
// 		return
// 	}

// 	bpmStr := c.Query("bpm")
// 	if bpmStr == "" {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing bpm"})
// 		return
// 	}
// 	bpm, err := strconv.ParseFloat(bpmStr, 64)
// 	if err != nil {
// 		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bpm: " + err.Error()})
// 		return
// 	}
// 	roundedBPM := int(math.Round(float64(bpm)/5) * 5)
// 	min := roundedBPM - 2
// 	max := roundedBPM + 2

// 	db.GetTracksByBPM(userId, min, max)

// 	c.JSON(http.StatusOK, gin.H{"": []int{roundedBPM - 1, roundedBPM, roundedBPM + 1}})
// }
