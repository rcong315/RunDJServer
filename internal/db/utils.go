package db

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool    *pgxpool.Pool
	dbOnce    sync.Once
	initError error
)

const BatchSize = 100

// --- initDB function remains the same logically ---
func initDB() error {
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	// Construct connection string - remove prefer_simple_protocol=true if you added it
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	// Use v5 pgxpool.Connect (API is compatible)
	pool, err := pgxpool.New(context.Background(), connString) // pgxpool.New is preferred in v5 over Connect
	if err != nil {
		return fmt.Errorf("unable to connect to database: %v", err)
	}

	dbPool = pool
	return nil
}

// --- getDB function remains the same logically ---
func getDB() (*pgxpool.Pool, error) {
	dbOnce.Do(func() {
		initError = initDB()
	})

	if initError != nil {
		return nil, initError
	}

	return dbPool, nil
}

// Note: br.Exec() in v5 returns (pgconn.CommandTag, error)
// We can ignore the command tag if not needed.
func processBatchResults(br pgx.BatchResults, count int) error {
	for i := range count { // Corrected loop range
		_, err := br.Exec()
		if err != nil {
			_ = br.Close() // Attempt to close even on error
			return fmt.Errorf("error executing batch item %d: %w", i, err)
		}
	}
	return br.Close()
}

func batchAndSave(items any, insertQuery string, paramConverter func(item any) []any) error {
	db, err := getDB()
	if err != nil {
		return fmt.Errorf("database connection error: %v", err)
	}

	ctx := context.Background()
	// Use v5 db.Begin (API is compatible)
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	// Use v5 tx.Rollback (API is compatible)
	defer tx.Rollback(ctx) // Rollback is a safety net if Commit isn't reached

	slice := reflect.ValueOf(items)
	if slice.Kind() != reflect.Slice {
		// Rollback explicitly here as Commit won't be reached
		_ = tx.Rollback(ctx)
		return fmt.Errorf("items must be a slice")
	}

	// Use v5 pgx.Batch (API is compatible)
	batch := &pgx.Batch{}
	sliceLen := slice.Len() // Get length once
	for i := range sliceLen {
		item := slice.Index(i).Interface()
		params := paramConverter(item)
		// Use v5 batch.Queue (API is compatible)
		batch.Queue(insertQuery, params...)

		// Use v5 batch.Len() (API is compatible)
		if batch.Len() >= BatchSize {
			// Use v5 tx.SendBatch (API is compatible)
			br := tx.SendBatch(ctx, batch)
			// processBatchResults uses v5 APIs internally now via br
			if err := processBatchResults(br, batch.Len()); err != nil {
				// Rollback already deferred, but error occurred during batch
				return fmt.Errorf("batch execution error: %v", err)
			}
			// Reset batch for the next set
			batch = &pgx.Batch{}
		}
	}

	// Process any remaining items
	if batch.Len() > 0 {
		br := tx.SendBatch(ctx, batch)
		if err := processBatchResults(br, batch.Len()); err != nil {
			// Rollback already deferred
			return fmt.Errorf("final batch execution error: %v", err)
		}
	}

	// Use v5 tx.Commit (API is compatible)
	if err := tx.Commit(ctx); err != nil {
		// Commit failed, Rollback was already deferred but might also fail
		return fmt.Errorf("transaction commit error: %v", err)
	}

	// If Commit succeeds, the deferred Rollback will return pgx.ErrTxCommitSuccess
	// which is ignored by convention.
	return nil
}
