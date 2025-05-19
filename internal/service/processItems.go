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

	logger.Info("All initial fetches done, waiting for processing jobs to complete", zap.String("userId", userId))
	jobWg.Wait()

	logger.Info("All processing jobs completed, stopping worker pool", zap.String("userId", userId))
	pool.Stop()

	errorCollectionWg.Wait()
	logger.Info("Worker pool stopped", zap.String("userId", userId))

	errorMu.Lock()
	defer errorMu.Unlock()
	if len(allErrors) > 0 {
		logger.Error("Data processing finished with errors",
			zap.String("userId", userId),
			zap.Int("errorCount", len(allErrors)),
			zap.Errors("errors", allErrors),
		)
	} else {
		logger.Info("Finished data processing successfully", zap.String("userId", userId))
	}
}
