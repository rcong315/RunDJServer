package service

import (
	"fmt"
	"sync"

	cmap "github.com/orcaman/concurrent-map/v2"
	"go.uber.org/zap"
)

// --- Worker Pool Setup ---

// StageContext tracks jobs belonging to a specific processing stage
type StageContext struct {
	wg   *sync.WaitGroup
	name string
}

// Job represents a task for a worker to execute.
// We use an interface to allow different kinds of tasks.
type Job interface {
	Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error
}

// WorkerPool manages a pool of workers and distributes jobs.
type WorkerPool struct {
	numWorkers  int
	jobsChan    chan *JobWrapper
	resultsChan chan error // Channel to collect errors from jobs
	wg          sync.WaitGroup
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(numWorkers int, jobQueueSize int) *WorkerPool {
	return &WorkerPool{
		numWorkers:  numWorkers,
		jobsChan:    make(chan *JobWrapper, jobQueueSize),   // Buffered channel
		resultsChan: make(chan error, jobQueueSize), // Buffered channel for errors
	}
}

// Start initializes the workers.
func (wp *WorkerPool) Start(jobWg *sync.WaitGroup, tracker *ProcessedTracker) {
	for i := range wp.numWorkers {
		wp.wg.Add(1)
		// Pass necessary context (pool, jobWg, tracker) to the worker
		go wp.worker(i+1, jobWg, tracker)
	}
}

// JobWrapper wraps a job with its stage context
type JobWrapper struct {
	job   Job
	stage *StageContext
}

// worker is the function executed by each worker goroutine.
func (wp *WorkerPool) worker(id int, jobWg *sync.WaitGroup, tracker *ProcessedTracker) {
	defer wp.wg.Done()
	logger.Debug("Worker started", zap.Int("workerId", id))
	for wrapper := range wp.jobsChan {
		jobType := fmt.Sprintf("%T", wrapper.job)
		logger.Debug("Worker processing job", zap.Int("workerId", id), zap.String("jobType", jobType))
		// Pass context down to the job's Execute method
		err := wrapper.job.Execute(wp, jobWg, tracker, wrapper.stage)
		if err != nil {
			select {
			case wp.resultsChan <- err:
				logger.Error("Worker job execution error", zap.Int("workerId", id), zap.String("jobType", jobType), zap.Error(err))
			default:
				logger.Warn("Worker: Error result channel full, discarding error",
					zap.Int("workerId", id),
					zap.String("jobType", jobType),
					zap.Error(err))
			}
		}
		jobWg.Done() // Decrement job wait group *after* job execution completes
		
		// Also decrement stage wait group if present
		if wrapper.stage != nil {
			wrapper.stage.wg.Done()
		}
	}
	logger.Debug("Worker finished", zap.Int("workerId", id))
}

// Submit adds a job to the queue.
// It also increments the job WaitGroup.
func (wp *WorkerPool) Submit(job Job, jobWg *sync.WaitGroup) {
	wp.SubmitWithStage(job, jobWg, nil)
}

// SubmitWithStage adds a job to the queue with stage tracking.
// It increments both the job WaitGroup and the stage WaitGroup if provided.
func (wp *WorkerPool) SubmitWithStage(job Job, jobWg *sync.WaitGroup, stage *StageContext) {
	jobWg.Add(1) // Increment global WG *before* sending to channel
	
	if stage != nil {
		stage.wg.Add(1) // Also increment stage WG
	}
	
	wp.jobsChan <- &JobWrapper{
		job:   job,
		stage: stage,
	}
}

// Stop closes the jobs channel and waits for all workers to finish processing.
func (wp *WorkerPool) Stop() {
	close(wp.jobsChan)    // Signal workers that no more jobs will be sent
	wp.wg.Wait()          // Wait for all worker goroutines to finish
	close(wp.resultsChan) // Close results channel after workers are done
}

// --- Processed Item Tracker (for Deduplication) ---

type ProcessedTracker struct {
	processedTracks    cmap.ConcurrentMap[string, struct{}]
	processedPlaylists cmap.ConcurrentMap[string, struct{}]
	processedArtists   cmap.ConcurrentMap[string, struct{}]
	processedAlbums    cmap.ConcurrentMap[string, struct{}]
	processedSingles   cmap.ConcurrentMap[string, struct{}]
}

func NewProcessedTracker() *ProcessedTracker {
	return &ProcessedTracker{
		processedTracks:    cmap.New[struct{}](),
		processedPlaylists: cmap.New[struct{}](),
		processedArtists:   cmap.New[struct{}](),
		processedAlbums:    cmap.New[struct{}](),
		processedSingles:   cmap.New[struct{}](),
	}
}

// CheckAndMark checks if an ID is processed, marks it if not. Returns true if already processed.
func (pt *ProcessedTracker) CheckAndMark(itemType string, id string) bool {
	var targetMap cmap.ConcurrentMap[string, struct{}]

	switch itemType {
	case "track":
		targetMap = pt.processedTracks
	case "playlist":
		targetMap = pt.processedPlaylists
	case "artist":
		targetMap = pt.processedArtists
	case "album":
		targetMap = pt.processedAlbums
	case "single":
		targetMap = pt.processedSingles
	default:
		logger.Warn("Unknown item type for processed check", zap.String("itemType", itemType), zap.String("itemId", id))
		return false
	}

	return !targetMap.SetIfAbsent(id, struct{}{})
}
