package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"

	"github.com/jackc/pgx/v4"
	"github.com/joho/godotenv"

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

type SpotifyUser struct {
	ID          string `json:"id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

func main() {
	if os.Getenv("DEBUG") == "true" {
		err := godotenv.Load()
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	http.HandleFunc("/", homeHandler)
	http.HandleFunc("/thanks", thanksHandler)

	http.HandleFunc("/api/user/register", registerHandler)
	http.HandleFunc("/api/songs/preset", presetPlaylistHandler)

	port := ":8080"
	fmt.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
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

	var user SpotifyUser
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		http.Error(w, "Error decoding response", http.StatusInternalServerError)
		return
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	dbName := os.Getenv("DB_NAME")
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Connect to the database
	conn, err := pgx.Connect(context.Background(), connString)
	if err != nil {
		log.Fatalf("Unable to connect to database: %v", err)
	}
	defer conn.Close(context.Background())

	// Create a new user record
	_, err = conn.Exec(context.Background(),
		`INSERT INTO "user" (user_id, email, display_name) VALUES ($1, $2, $3) ON CONFLICT (user_id) DO NOTHING`, user.ID, user.Email, user.DisplayName)
	if err != nil {
		log.Printf("Error creating user record: %v", err)
	}

	fmt.Printf("Created new user: ID=%s, Email=%s, DisplayName=%s\n",
		user.ID, user.Email, user.DisplayName)

	var ids []string = spotify.GetAllTracks(accessToken)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ids)
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

	playlistID, exists := presetPlaylists[roundedBPM]
	if !exists {
		http.Error(w, "Playlist not found for the given BPM", http.StatusNotFound)
		return
	}

	fmt.Fprint(w, playlistID)
}
