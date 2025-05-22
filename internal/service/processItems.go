package service

import (
	"sync"

	"go.uber.org/zap"
)

// TODO: Clean up nested size = 0 checks

// TODO: When to run this function? On register and when else? Cron?
// TODO: Add release radar playlist
// TODO: Try seperating files by track, playlist, artist, etc.?
func processAll(token string, userId string) {
	logger.Info("Starting data processing", zap.String("userId", userId))

	numWorkers := 20       // Adjust based on resources and API limits
	jobQueueSize := 100000 // Adjust based on expected number of jobs

	pool := NewWorkerPool(numWorkers, jobQueueSize)
	tracker := NewProcessedTracker() // Create tracker for deduplication
	var jobWg sync.WaitGroup         // WaitGroup to track submitted jobs

	var allErrors []error  // Slice to collect errors
	var errorMu sync.Mutex // Mutex to protect allErrors slice

	// Goroutine to collect errors from the results channel
	errorCollectionWg := sync.WaitGroup{}
	errorCollectionWg.Add(1)
	go func() {
		defer errorCollectionWg.Done()
		for err := range pool.resultsChan {
			if err != nil {
				errorMu.Lock()
				allErrors = append(allErrors, err)
				errorMu.Unlock()
			}
		}
	}()

	pool.Start(&jobWg, tracker) // Start the worker pool, pass context

	// Process each data type in its own function
	processAndCollectErrors := func(processFunc func(string, string, *WorkerPool, *ProcessedTracker, *sync.WaitGroup) error) {
		if err := processFunc(userId, token, pool, tracker, &jobWg); err != nil {
			errorMu.Lock()
			allErrors = append(allErrors, err)
			errorMu.Unlock()
		}
	}

	processAndCollectErrors(processTopTracks)
	processAndCollectErrors(processSavedTracks)
	processAndCollectErrors(processPlaylists)
	processAndCollectErrors(processTopArtists)
	processAndCollectErrors(processFollowedArtists)
	processAndCollectErrors(processSavedAlbums)

	// Launch a goroutine to handle graceful shutdown and error reporting
	// This allows processAll to return immediately after queueing.
	go func(currentUserId string, currentPool *WorkerPool, currentJobWg *sync.WaitGroup, currentErrorCollectionWg *sync.WaitGroup, currentAllErrors *[]error, currentErrorMu *sync.Mutex) {
		logger.Info("Background shutdown handler started", zap.String("userId", currentUserId))

		currentJobWg.Wait() // Wait for all submitted jobs to complete
		logger.Info("All processing jobs completed, stopping worker pool", zap.String("userId", currentUserId))

		currentPool.Stop() // Stop the worker pool (closes jobsChan, waits for workers, closes resultsChan)
		logger.Info("Worker pool stop signal sent", zap.String("userId", currentUserId))

		currentErrorCollectionWg.Wait() // Wait for the error collection goroutine to finish
		logger.Info("Error collection finished, worker pool fully stopped", zap.String("userId", currentUserId))

		currentErrorMu.Lock()
		defer currentErrorMu.Unlock()
		if len(*currentAllErrors) > 0 {
			logger.Error("Background data processing finished with errors",
				zap.String("userId", currentUserId),
				zap.Int("errorCount", len(*currentAllErrors)),
				zap.Errors("errors", *currentAllErrors),
			)
		} else {
			logger.Info("Background data processing finished successfully", zap.String("userId", currentUserId))
		}
	}(userId, pool, &jobWg, &errorCollectionWg, &allErrors, &errorMu)

	logger.Info("All processing functions queued, processing will continue in the background. processAll is returning.", zap.String("userId", userId))
}
