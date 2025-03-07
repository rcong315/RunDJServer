package db

import (
	"context"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/joho/godotenv"
)

var (
	dbPool    *pgxpool.Pool
	dbOnce    sync.Once
	initError error
)

func initDB() error {
	if os.Getenv("DEBUG") == "true" {
		err := godotenv.Load("../../.env")
		if err != nil {
			log.Printf("Error loading .env file: %v", err)
		}
	}

	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	pool, err := pgxpool.Connect(context.Background(), connString)
	if err != nil {
		return fmt.Errorf("unable to connect to database: %v", err)
	}

	dbPool = pool
	return nil
}

func GetDB() (*pgxpool.Pool, error) {
	dbOnce.Do(func() {
		initError = initDB()
	})

	if initError != nil {
		return nil, initError
	}

	return dbPool, nil
}

func SaveUser(userId string, email string, displayName string) error {
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	_, err = db.Exec(context.Background(),
		`INSERT INTO "user" (user_id, email, display_name) VALUES ($1, $2, $3) 
			ON CONFLICT (user_id) DO NOTHING`, userId, email, displayName)
	if err != nil {
		return fmt.Errorf("error creating user record: %v", err)
	}

	return nil
}

func SaveSong(id string, userId string, title string, artists []string, genre string, bpm float64) error {
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	_, err = db.Exec(context.Background(),
		`INSERT INTO "song" (spotify_id, user_id, title, artists, genre, bpm) 
			VALUES ($1, ARRAY[$2], $3, $4, $5, $6)
			ON CONFLICT (spotify_id) DO UPDATE 
			SET user_id = song.user_id || ARRAY[$2], bpm = $6
			WHERE NOT (song.user_id @> ARRAY[$2])`, id, userId, title, artists, genre, bpm)
	if err != nil {
		return fmt.Errorf("error creating song record: %v", err)
	}

	return nil
}
