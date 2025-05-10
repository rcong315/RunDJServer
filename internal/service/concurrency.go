package service

import (
	"log"
	"sync"
)

// --- Worker Pool Setup ---

// Job represents a task for a worker to execute.
// We use an interface to allow different kinds of tasks.
type Job interface {
	Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker) error
}

// WorkerPool manages a pool of workers and distributes jobs.
type WorkerPool struct {
	numWorkers  int
	jobsChan    chan Job
	resultsChan chan error // Channel to collect errors from jobs
	wg          sync.WaitGroup
}

// NewWorkerPool creates a new worker pool.
func NewWorkerPool(numWorkers int, jobQueueSize int) *WorkerPool {
	return &WorkerPool{
		numWorkers:  numWorkers,
		jobsChan:    make(chan Job, jobQueueSize),   // Buffered channel
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

// worker is the function executed by each worker goroutine.
func (wp *WorkerPool) worker(id int, jobWg *sync.WaitGroup, tracker *ProcessedTracker) {
	defer wp.wg.Done()
	log.Printf("Worker %d started", id)
	for job := range wp.jobsChan {
		log.Printf("Worker %d processing job: %T", id, job)
		// Pass context down to the job's Execute method
		err := job.Execute(wp, jobWg, tracker)
		if err != nil {
			select {
			case wp.resultsChan <- err:
			default:
				log.Printf("Worker %d: Error result channel full, discarding error: %v", id, err)
			}
		}
		jobWg.Done() // Decrement job wait group *after* job execution completes
	}
	log.Printf("Worker %d finished", id)
}

// Submit adds a job to the queue.
// It also increments the job WaitGroup.
func (wp *WorkerPool) Submit(job Job, jobWg *sync.WaitGroup) {
	jobWg.Add(1) // Increment WG *before* sending to channel
	wp.jobsChan <- job
}

// Stop closes the jobs channel and waits for all workers to finish processing.
func (wp *WorkerPool) Stop() {
	close(wp.jobsChan)    // Signal workers that no more jobs will be sent
	wp.wg.Wait()          // Wait for all worker goroutines to finish
	close(wp.resultsChan) // Close results channel after workers are done
}

// --- Processed Item Tracker (for Deduplication) ---

type ProcessedTracker struct {
	mu                 sync.Mutex
	processedTracks    map[string]struct{}
	processedPlaylists map[string]struct{}
	processedArtists   map[string]struct{}
	processedAlbums    map[string]struct{}
	processedSingles   map[string]struct{}
}

func NewProcessedTracker() *ProcessedTracker {
	return &ProcessedTracker{
		processedTracks:    make(map[string]struct{}),
		processedPlaylists: make(map[string]struct{}),
		processedArtists:   make(map[string]struct{}),
		processedAlbums:    make(map[string]struct{}),
		processedSingles:   make(map[string]struct{}),
	}
}

// CheckAndMark checks if an ID is processed, marks it if not. Returns true if already processed.
func (pt *ProcessedTracker) CheckAndMark(itemType string, id string) bool {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	var targetMap map[string]struct{}
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
		log.Printf("WARN: Unknown item type '%s' for processed check", itemType)
		return false // Don't block unknown types, but log it
	}

	if _, exists := targetMap[id]; exists {
		return true // Already processed
	}
	targetMap[id] = struct{}{} // Mark as processed
	return false               // Was not processed before
}
