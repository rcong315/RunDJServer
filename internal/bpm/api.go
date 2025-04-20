// *** DEPRECATED ***
// This file is no longer used. Functionality is replaced by the Spotify API.

package bpm

// import (
// 	"encoding/json"
// 	"fmt"
// 	"log"
// 	"net/http"
// 	"os"
// 	"time"

// 	"github.com/joho/godotenv"
// 	"github.com/rcong315/RunDJServer/internal/db"
// )

// const bpmAPIURL = "https://soundstat.info/api/v1/track/"

// type Song struct {
// 	Id         string   `json:"id"`
// 	Name       string   `json:"name"`
// 	Artists    []string `json:"artists"`
// 	Genre      string   `json:"genre"`
// 	Popularity int      `json:"popularity"`
// 	Features   struct {
// 		Tempo            float64 `json:"tempo"`
// 		Key              int     `json:"key"`
// 		Mode             int     `json:"mode"`
// 		KeyConfidence    float64 `json:"key_confidence"`
// 		Energy           float64 `json:"energy"`
// 		Danceability     float64 `json:"danceability"`
// 		Valence          float64 `json:"valence"`
// 		Instrumentalness float64 `json:"instrumentalness"`
// 		Acousticness     float64 `json:"acousticness"`
// 		Loudness         float64 `json:"loudness"`
// 	} `json:"features"`
// }

// func getBPM(key string, id string) (*Song, error) {
// 	url := fmt.Sprintf("%s%s", bpmAPIURL, id)
// 	req, err := http.NewRequest("GET", url, nil)
// 	if err != nil {
// 		return nil, fmt.Errorf("error creating request: %v", err)
// 	}
// 	req.Header.Set("x-api-key", key)

// 	client := &http.Client{}
// 	resp, err := client.Do(req)
// 	if err != nil {
// 		return nil, fmt.Errorf("error making GET request: %v", err)
// 	}
// 	defer resp.Body.Close()

// 	if resp.StatusCode != http.StatusOK {
// 		if resp.StatusCode == http.StatusNotFound {
// 			return nil, fmt.Errorf("track not found: %s", id)
// 		} else if resp.StatusCode == http.StatusTooManyRequests {
// 			log.Printf("Rate limited, waiting 5 seconds")
// 			time.Sleep(5 * time.Second)
// 			return nil, fmt.Errorf("rate limited")
// 		} else {
// 			return nil, fmt.Errorf("error from BPM server: %d", resp.StatusCode)
// 		}
// 	}

// 	var song Song
// 	if err := json.NewDecoder(resp.Body).Decode(&song); err != nil {
// 		return nil, fmt.Errorf("error from BPM server: %d", resp.StatusCode)
// 	}

// 	songBytes, _ := json.MarshalIndent(song, "", "  ")
// 	log.Printf("Song details: %s", string(songBytes))
// 	return &song, nil
// }

// func SaveBPMs(userId string, ids []string) []string {
// 	if os.Getenv("DEBUG") == "true" {
// 		err := godotenv.Load("../../.env")
// 		if err != nil {
// 			log.Fatal("Error loading .env file")
// 		}
// 	}

// 	bpmAPIKey := os.Getenv("BPM_API_KEY")
// 	currentIds := ids

// 	for len(currentIds) > 0 {
// 		var nextRetryIds []string

// 		for _, id := range currentIds {
// 			song, err := getBPM(bpmAPIKey, id)
// 			if err != nil {
// 				log.Printf("Error getting BPM for track %s: %v", id, err)
// 				nextRetryIds = append(nextRetryIds, id)
// 			} else {
// 				if err := db.SaveSong(song.Id, userId, song.Name, song.Artists, song.Genre, song.Features.Tempo); err != nil {
// 					log.Printf("Error saving song %s: %v", song.Id, err)
// 					nextRetryIds = append(nextRetryIds, id)
// 				}
// 			}
// 		}

// 		if len(nextRetryIds) > 0 {
// 			log.Printf("Retrying %d tracks", len(nextRetryIds))
// 			currentIds = nextRetryIds
// 		} else {
// 			break
// 		}
// 	}

// 	return nil
// }

// func getSongs(bpm int) []string {
// 	var ids []string
// 	return ids
// }
