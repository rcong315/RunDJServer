package service

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"strconv"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

const spotifyAPIURL = "https://api.spotify.com/v1"

var presetPlaylists = map[int]string{
	105: "56cgN0YoqzPjmNBBuiVo6b",
	110: "2pX7htNxQUGZSObonznRyn",
	115: "78qmqXAefQPCbQ5JqfwWgz",
	120: "2rzL3ZFSz87245ljAic93z",
	125: "37i9dQZF1EIgsxtEuT3KWN",
	130: "37i9dQZF1EIdJGESPytB8N",
	135: "37i9dQZF1EIdnGKfcfozNo",
	140: "37i9dQZF1EIgOKtiospcqN",
	145: "37i9dQZF1EIcB36Vij2P5d",
	150: "37i9dQZF1EIgrZKdA44WQK",
	155: "37i9dQZF1EIeGfmJObJDc0",
	160: "37i9dQZF1EIdYV92VKrjuC",
	165: "37i9dQZF1EIcNylL4dr08W",
	170: "37i9dQZF1EIgfIackHptHl",
	175: "37i9dQZF1EIfnhoQIQxMqH",
	180: "37i9dQZF1EIgUYhklBpeMG",
	185: "37i9dQZF1EIhy9qfhxNEnX",
	190: "37i9dQZF1EIcID9rq1OAoH",
}

type Message struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

func HomeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	fmt.Fprintf(w, "RunDJ Backend")
}

func thanksHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, "<html><body><a href=\"https://getsongbpm.com>getsongbpm.com</a></body></html>")
}

func registerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	accessToken := r.URL.Query().Get("access_token")
	apiURL := fmt.Sprintf("%s/me", spotifyAPIURL)
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		http.Error(w, "Error creating request", http.StatusInternalServerError)
		return
	}
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", accessToken))

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, "Error making GET request", http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, "Error from Spotify server", resp.StatusCode)
		return
	}

	var user db.User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Error(w, "Error decoding response", http.StatusInternalServerError)
		return
	}

	err = db.SaveUser(user)
	if err != nil {
		http.Error(w, "Error saving user", http.StatusInternalServerError)
		log.Printf("Error saving user: %v\n", err)
		return
	}
	log.Printf("Created new user: Id=%s, Email=%s, DisplayName=%s\n",
		user.UserId, user.Email, user.DisplayName)

	go func(accessToken, userId string) {
		log.Printf("Getting all tracks for user %s\n", userId)
		tracks, _, _, _ := spotify.GetAllTracks(accessToken)
		save(userId, tracks)
		// log.Printf("Finished getting %d tracks for user %s\n", len(ids), userI)
		// log.Print("Getting and saving song BPMs, this might take a while...\n")
		// bpm.SaveBPMs(userId, ids)
		// log.Printf("Finished saving BPMs for user %s\n", userId)
	}(accessToken, user.UserId)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(Message{Status: "success", Message: "User registered successfully, processing tracks"})
}

func presetPlaylistHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bpmStr := r.URL.Query().Get("bpm")
	if bpmStr == "" {
		http.Error(w, "Missing bpm", http.StatusBadRequest)
		return
	}

	bpm, err := strconv.ParseFloat(bpmStr, 64)
	if err != nil {
		http.Error(w, "Invalid bpm", http.StatusBadRequest)
		return
	}

	roundedBPM := int(math.Round(float64(bpm)/5) * 5)

	playlistId, exists := presetPlaylists[roundedBPM]
	if !exists {
		http.Error(w, "Playlist not found for the given BPM", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, playlistId)
}
