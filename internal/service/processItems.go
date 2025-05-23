package service

import (
	"sync"

	"go.uber.org/zap"
)

// TODO: Clean up nested size = 0 checks

// TODO: When to run this function? On register and when else? Cron?
// TODO: Add release radar playlist
func processAll(token string, userId string) {
	logger.Info("Queuing background data processing", zap.String("userId", userId))

	go func() {
		logger.Info("Starting data processing", zap.String("userId", userId))

		numWorkers := 20
		jobQueueSize := 100000

		pool := NewWorkerPool(numWorkers, jobQueueSize)
		tracker := NewProcessedTracker()
		var jobWg sync.WaitGroup

		var allErrors []error
		var errorMu sync.Mutex

		// Error collection goroutine
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

		pool.Start(&jobWg, tracker)

		// Process each data type
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

		jobWg.Wait()
		pool.Stop()
		errorCollectionWg.Wait()

		if len(allErrors) > 0 {
			logger.Error("Background data processing finished with errors",
				zap.String("userId", userId),
				zap.Int("errorCount", len(allErrors)),
				zap.Errors("errors", allErrors),
			)
		} else {
			logger.Info("Background data processing finished successfully",
				zap.String("userId", userId))
		}
	}()
}
