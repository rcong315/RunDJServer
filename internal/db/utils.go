package db

import (
	"context"
	"fmt"
	"os"
	"reflect"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var (
	dbPool    *pgxpool.Pool
	dbOnce    sync.Once
	initError error
)

const BatchSize = 100

func initDB() error {
	dbHost := os.Getenv("DB_HOST")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")

	// Construct connection string - remove prefer_simple_protocol=true if you added it
	connString := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		dbUser, dbPassword, dbHost, dbPort, dbName)

	config, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return fmt.Errorf("unable to parse connection string: %w", err)
	}

	// Tell the pool to use the simple protocol by default for Exec/Query calls
	config.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol

	// Use a context with timeout for the initial connection attempt
	connectCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	pool, err := pgxpool.NewWithConfig(connectCtx, config) // Use NewWithConfig
	if err != nil {
		return fmt.Errorf("unable to create connection pool: %w", err)
	}

	// Optional: Ping the database to ensure connectivity before returning
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := pool.Ping(pingCtx); err != nil {
		pool.Close() // Close pool if ping fails
		return fmt.Errorf("unable to ping database pool: %w", err)
	}

	dbPool = pool
	return nil
}

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
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %v", err)
	}
	// Use Rollback with error checking (though often ignored in defer)
	defer func() {
		_ = tx.Rollback(ctx) // Ensure rollback is attempted on any exit path
	}()

	slice := reflect.ValueOf(items)
	if slice.Kind() != reflect.Slice {
		return fmt.Errorf("items must be a slice") // No need to explicitly rollback here, defer handles it
	}

	batch := &pgx.Batch{}
	sliceLen := slice.Len() // Get length once
	for i := range sliceLen {
		item := slice.Index(i).Interface()
		params := paramConverter(item)
		batch.Queue(insertQuery, params...)

		if batch.Len() >= BatchSize {
			br := tx.SendBatch(ctx, batch)
			// Pass the actual batch length to processBatchResults
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
		// Pass the actual batch length
		if err := processBatchResults(br, batch.Len()); err != nil {
			return fmt.Errorf("final batch execution error: %v", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		// Commit failed, Rollback was already deferred but might also fail
		return fmt.Errorf("transaction commit error: %v", err)
	}

	// If Commit succeeds, the deferred Rollback will harmlessly return pgx.ErrTxCommitSuccess
	return nil
}
