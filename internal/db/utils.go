package db

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
)

var (
	dbPool    *pgxpool.Pool
	dbOnce    sync.Once
	initError error
)

const BatchSize = 100

func initDB() error {
	// if os.Getenv("DEBUG") == "true" {
	// 	err := godotenv.Load("../../.env")
	// 	if err != nil {
	// 		log.Println("Warning: .env file not found. Using system environment variables.")
	// 	}
	// }

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

func processBatchResults(br pgx.BatchResults) error {
	for {
		_, err := br.Exec()
		if err != nil {
			if err == pgx.ErrNoRows {
				break
			}
			_ = br.Close()
			return err
		}
	}
	return br.Close()
}

func batchAndSave(userId string, items any, insertQuery string, paramConverter func(userId string, item any) []any) error {
	db, err := GetDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	defer tx.Rollback(ctx)

	// Create a slice value based on the items parameter
	slice := reflect.ValueOf(items)
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("items must be a slice")
	}

	batch := &pgx.Batch{}
	for i := range slice.Len() {
		item := slice.Index(i).Interface()
		params := paramConverter(userId, item)
		batch.Queue(insertQuery, params...)

		if batch.Len() >= BatchSize {
			br := tx.SendBatch(ctx, batch)
			if err := processBatchResults(br); err != nil {
				return fmt.Errorf("batch execution error: %v", err)
			}
			batch = &pgx.Batch{}
		}
	}

	if batch.Len() > 0 {
		br := tx.SendBatch(ctx, batch)
		if err := processBatchResults(br); err != nil {
			return fmt.Errorf("batch execution error: %v", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("transaction commit error: %v", err)
	}

	return nil
}
