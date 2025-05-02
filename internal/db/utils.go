package db

import (
	"context"
	"embed"
	"fmt"
	"log"
	"os"
	"path/filepath"
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

//go:embed sql/*.sql
var sqlFiles embed.FS // Variable to hold embedded SQL files
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

func batchAndSave(items any, queryFilename string, paramConverter func(item any) []any) error {
	sqlQuery, err := getQueryString(queryFilename)
	if err != nil {
		return fmt.Errorf("failed to get SQL query string: %w", err)
	}

	db, err := getDB()
	if err != nil {
		return fmt.Errorf("database connection error: %w", err)
	}

	ctx := context.Background()
	tx, err := db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("error starting transaction: %w", err)
	}
	// Defer Rollback guarantees an attempt to roll back if Commit fails or panic occurs.
	// pgx's Rollback handles cases where Commit already succeeded gracefully.
	defer tx.Rollback(ctx)

	slice := reflect.ValueOf(items)
	if slice.Kind() != reflect.Slice {
		// No explicit rollback needed, defer handles it.
		return fmt.Errorf("items must be a slice, got %v", slice.Kind())
	}

	batch := &pgx.Batch{}
	sliceLen := slice.Len()
	for i := range sliceLen { // Corrected loop iteration
		item := slice.Index(i).Interface()
		params := paramConverter(item)

		// Use the query read from the file
		batch.Queue(sqlQuery, params...)

		// Send batch if it reaches BatchSize
		if batch.Len() >= BatchSize {
			br := tx.SendBatch(ctx, batch)
			// Get the exact count sent before clearing the batch
			sentCount := batch.Len()
			// It's often safer to reset the batch *before* processing results
			batch = &pgx.Batch{}
			if err := processBatchResults(br, sentCount); err != nil {
				// Rollback handled by defer
				log.Printf("Error processing final batch: %v", item)
				return fmt.Errorf("batch execution error (batch size %d): %w", sentCount, err)
			}
		}
	}

	if batch.Len() > 0 {
		br := tx.SendBatch(ctx, batch)
		sentCount := batch.Len()
		if err := processBatchResults(br, sentCount); err != nil {
			// Rollback handled by defer
			log.Printf("Error processing final batch: %v", batch)
			return fmt.Errorf("final batch execution error (batch size %d): %w", sentCount, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		// Commit failed, deferred Rollback will attempt cleanup.
		return fmt.Errorf("transaction commit error: %w", err)
	}

	// Commit succeeded. Deferred Rollback will return pgx.ErrTxCommitSuccess, which is ignored.
	return nil
}

func executeSelect(queryFilename string, args ...any) (pgx.Rows, error) {
	sqlQuery, err := getQueryString(queryFilename)
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL query string: %w", err)
	}

	db, err := getDB()
	if err != nil {
		return nil, fmt.Errorf("database connection error: %w", err)
	}

	ctx := context.Background()
	rows, err := db.Query(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("error executing query: %w", err)
	}
	return rows, nil
}

func getQueryString(queryFilename string) (string, error) {
	sqlFilePathInEmbedFS := filepath.Join("sql", queryFilename+".sql") // Path *inside* the embed FS
	sqlBytes, err := sqlFiles.ReadFile(sqlFilePathInEmbedFS)           // Read from the embed.FS variable
	if err != nil {
		return "", fmt.Errorf("failed to read embedded SQL file %q: %w", sqlFilePathInEmbedFS, err)
	}
	return string(sqlBytes), nil
}
