package service

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// TODO: Clean up nested size = 0 checks

// TODO: When to run this function? On register and when else? Cron?
// TODO: Add release radar playlist
func processAll(token string, userId string) {
	logger.Info("Queuing background data processing", zap.String("userId", userId))

	go func() {
		startTime := time.Now()

		logger.Info("Starting data processing",
			zap.String("userId", userId),
			zap.Time("startTime", startTime))

		numWorkers := 16
		jobQueueSize := 100 * 1000

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

		processAndCollectErrors := func(name string, processFunc func(string, string, *WorkerPool, *ProcessedTracker, *sync.WaitGroup, *StageContext) error) {
			funcStart := time.Now()

			// Create a stage-specific wait group
			stageWg := &sync.WaitGroup{}
			stageCtx := &StageContext{
				wg:   stageWg,
				name: name,
			}

			if err := processFunc(userId, token, pool, tracker, &jobWg, stageCtx); err != nil {
				errorMu.Lock()
				allErrors = append(allErrors, err)
				errorMu.Unlock()
			}

			// Wait for all jobs in this stage to complete
			stageWg.Wait()

			logger.Info("Processing stage completed",
				zap.String("userId", userId),
				zap.String("stage", name),
				zap.Duration("stageDuration", time.Since(funcStart)))
		}

		var stagesWg sync.WaitGroup
		stagesWg.Add(6)
		go func() {
			defer stagesWg.Done()
			processAndCollectErrors("topTracks", processTopTracks)
		}()
		go func() {
			defer stagesWg.Done()
			processAndCollectErrors("savedTracks", processSavedTracks)
		}()
		go func() {
			defer stagesWg.Done()
			processAndCollectErrors("playlists", processPlaylists)
		}()
		go func() {
			defer stagesWg.Done()
			processAndCollectErrors("topArtists", processTopArtists)
		}()
		go func() {
			defer stagesWg.Done()
			processAndCollectErrors("followedArtists", processFollowedArtists)
		}()
		go func() {
			defer stagesWg.Done()
			processAndCollectErrors("savedAlbums", processSavedAlbums)
		}()
		stagesWg.Wait()

		jobWg.Wait()
		pool.Stop()
		errorCollectionWg.Wait()

		duration := time.Since(startTime)

		if len(allErrors) > 0 {
			logger.Error("Background data processing finished with errors",
				zap.String("userId", userId),
				zap.Int("errorCount", len(allErrors)),
				zap.Duration("duration", duration),
				zap.String("durationFormatted", duration.String()),
				zap.Errors("errors", allErrors))
		} else {
			logger.Info("Background data processing finished successfully",
				zap.String("userId", userId),
				zap.Duration("duration", duration),
				zap.String("durationFormatted", duration.String()))
		}
	}()
}
