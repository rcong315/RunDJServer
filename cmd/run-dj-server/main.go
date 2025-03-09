package main

import (
	"log"
	"net/http"
	"os"

	"github.com/joho/godotenv"
)

func main() {
	if os.Getenv("DEBUG") == "true" {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Fatal("Error loading .env file")
		}
	}

	http.HandleFunc("/", service.HomeHandler)
	http.HandleFunc("/thanks", service.ThanksHandler)

	http.HandleFunc("/api/user/register", service.RegisterHandler)
	http.HandleFunc("/api/songs/preset", service.PresetPlaylistHandler)

	port := ":8080"
	log.Printf("Server starting on port %s\n", port)
	if err := http.ListenAndServe(port, nil); err != nil {
		log.Fatal(err)
	}
}
