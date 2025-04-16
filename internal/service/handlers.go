package service

import (
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
	c.String(http.StatusOK, "RunDJ Backend")
}

func ThanksHandler(c *gin.Context) {
	c.Header("Content-Type", "text/html")
	c.String(http.StatusOK, "<html><body><a href=\"https://getsongbpm.com\">getsongbpm.com</a></body></html>")
}

func RegisterHandler(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Missing access_token"})
		return
	}

	user, err := spotify.GetUser(accessToken)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error getting user: " + err})
		return
	}

	err = saveUser(user)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error saving user: " + err.Error()})
		return
	}
	log.Printf("Created new user: Id=%s, Email=%s, DisplayName=%s\n",
		user.Id, user.Email, user.DisplayName)

	go func(accessToken, userId string) {
		log.Printf("Getting all tracks for user %s\n", userId)
		tracks, _ := spotify.GetUsersTopTracks(accessToken)
		err = saveTracks(userId, tracks, "top tracks")
		if err != nil {
			log.Printf("Error saving tracks: %v", err)
		}
		// log.Printf("Finished getting %d tracks for user %s\n", len(ids), userI)
		// log.Print("Getting and saving song BPMs, this might take a while...\n")
		// bpm.SaveBPMs(userId, ids)
		// log.Printf("Finished saving BPMs for user %s\n", userId)
	}(accessToken, user.Id)

	c.JSON(http.StatusOK, Message{
		Status:  "success",
		Message: "User registered successfully, processing tracks",
	})
}

func PresetPlaylistHandler(c *gin.Context) {
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

func MatchingTracksHandler(c *gin.Context) {
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

	min := roundedBPM - 2
	max := roundedBPM + 2

	db.GetTracksByBPM(userId, min, max)

	c.JSON(http.StatusOK, gin.H{"": roundedBPM - 1, roundedBPM, roundedBPM + 1})
}
