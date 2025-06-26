package service

import (
	"fmt"
	"sync"
	"time"

	cmap "github.com/orcaman/concurrent-map/v2"
	"go.uber.org/zap"
)

// --- Worker Pool Setup ---

// StageContext tracks jobs belonging to a specific processing stage
type StageContext struct {
	Wg   *sync.WaitGroup
	Name string
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
	
	// Simple queue monitoring
	queueHighWaterMark int64
	lastLoggedHigh     int64
	mu                 sync.Mutex
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
			wrapper.stage.Wg.Done()
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
	// Calculate current queue size before attempting to queue
	currentSize := len(wp.jobsChan)
	
	// Update high water mark
	wp.mu.Lock()
	if int64(currentSize) > wp.queueHighWaterMark {
		wp.queueHighWaterMark = int64(currentSize)
		
		// Only log if it's significant: >20% of queue capacity AND >100 more than last logged
		capacity := cap(wp.jobsChan)
		if int64(currentSize) > int64(capacity/5) && 
		   int64(currentSize) > wp.lastLoggedHigh+100 {
			wp.lastLoggedHigh = int64(currentSize)
			
			// Estimate memory usage
			memoryMB := wp.estimateQueueMemoryMB(currentSize)
			
			logger.Info("New significant queue high water mark",
				zap.Int64("maxQueueSize", wp.queueHighWaterMark),
				zap.Int("currentSize", currentSize),
				zap.Int("capacity", capacity),
				zap.Float64("percentFull", float64(currentSize)/float64(capacity)*100),
				zap.Float64("estimatedMemoryMB", memoryMB))
		}
	}
	wp.mu.Unlock()
	
	// Warn if queue is getting full
	capacity := cap(wp.jobsChan)
	if currentSize > capacity*80/100 {
		logger.Warn("Queue is nearly full!",
			zap.Int("currentSize", currentSize),
			zap.Int("capacity", capacity),
			zap.Float64("percentFull", float64(currentSize)/float64(capacity)*100))
	}
	
	// Only increment waitgroups after successfully queuing
	jobWg.Add(1) // Increment global WG
	
	if stage != nil {
		stage.Wg.Add(1) // Also increment stage WG
	}
	
	// This will block if queue is full
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
	
	// Calculate peak memory usage
	peakMemoryMB := wp.estimateQueueMemoryMB(int(wp.queueHighWaterMark))
	
	logger.Info("WorkerPool stopped",
		zap.Int64("maxQueueSizeReached", wp.queueHighWaterMark),
		zap.Int("queueCapacity", cap(wp.jobsChan)),
		zap.Float64("peakMemoryUsageMB", peakMemoryMB),
		zap.Float64("percentOfCapacityUsed", float64(wp.queueHighWaterMark)/float64(cap(wp.jobsChan))*100))
}

// GetQueueSize returns the current number of jobs in the queue
func (wp *WorkerPool) GetQueueSize() int {
	return len(wp.jobsChan)
}

// GetMaxQueueSize returns the maximum queue size observed
func (wp *WorkerPool) GetMaxQueueSize() int64 {
	wp.mu.Lock()
	defer wp.mu.Unlock()
	return wp.queueHighWaterMark
}

// GetResultsChan returns the results channel for error collection
func (wp *WorkerPool) GetResultsChan() <-chan error {
	return wp.resultsChan
}

// TrySubmitWithStage attempts to submit without blocking
// Returns true if job was queued, false if queue is full
func (wp *WorkerPool) TrySubmitWithStage(job Job, jobWg *sync.WaitGroup, stage *StageContext) bool {
	select {
	case wp.jobsChan <- &JobWrapper{job: job, stage: stage}:
		// Successfully queued - now increment waitgroups
		jobWg.Add(1)
		if stage != nil {
			stage.Wg.Add(1)
		}
		
		// Update monitoring
		currentSize := len(wp.jobsChan)
		wp.mu.Lock()
		if int64(currentSize) > wp.queueHighWaterMark {
			wp.queueHighWaterMark = int64(currentSize)
			
			// Only log if it's significant
			capacity := cap(wp.jobsChan)
			if int64(currentSize) > int64(capacity/5) && 
			   int64(currentSize) > wp.lastLoggedHigh+100 {
				wp.lastLoggedHigh = int64(currentSize)
				
				// Estimate memory usage
				memoryMB := wp.estimateQueueMemoryMB(currentSize)
				
				logger.Info("New significant queue high water mark",
					zap.Int64("maxQueueSize", wp.queueHighWaterMark),
					zap.Int("currentSize", currentSize),
					zap.Int("capacity", capacity),
					zap.Float64("percentFull", float64(currentSize)/float64(capacity)*100),
					zap.Float64("estimatedMemoryMB", memoryMB))
			}
		}
		wp.mu.Unlock()
		
		return true
		
	default:
		// Queue is full, would block
		logger.Error("Queue full - job rejected!",
			zap.String("jobType", fmt.Sprintf("%T", job)),
			zap.Int("queueSize", len(wp.jobsChan)),
			zap.Int("queueCapacity", cap(wp.jobsChan)))
		return false
	}
}

// SubmitWithRetry attempts to submit with exponential backoff
func (wp *WorkerPool) SubmitWithRetry(job Job, jobWg *sync.WaitGroup, stage *StageContext, maxRetries int) error {
	backoff := 100 * time.Millisecond
	
	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			time.Sleep(backoff)
			backoff *= 2
			if backoff > 5*time.Second {
				backoff = 5 * time.Second
			}
		}
		
		if wp.TrySubmitWithStage(job, jobWg, stage) {
			return nil
		}
		
		logger.Warn("Retrying job submission",
			zap.Int("attempt", attempt+1),
			zap.Int("maxRetries", maxRetries),
			zap.Duration("nextBackoff", backoff))
	}
	
	return fmt.Errorf("failed to submit job after %d retries - queue is full", maxRetries)
}

// estimateQueueMemoryMB estimates memory usage of the queue in MB
func (wp *WorkerPool) estimateQueueMemoryMB(queueSize int) float64 {
	// Base sizes in bytes
	const (
		// JobWrapper struct overhead
		jobWrapperSize = 16 // two pointers (8 bytes each on 64-bit)
		
		// Estimated average job size (varies by job type)
		// Most jobs contain: token string, IDs, function pointers
		avgJobSize = 200 // conservative estimate
		
		// Channel overhead per item
		channelOverhead = 8
	)
	
	totalBytes := queueSize * (jobWrapperSize + avgJobSize + channelOverhead)
	return float64(totalBytes) / (1024 * 1024)
}

// GetQueueStats returns current queue statistics including memory usage
func (wp *WorkerPool) GetQueueStats() (current int, max int64, memoryMB float64) {
	current = len(wp.jobsChan)
	
	wp.mu.Lock()
	max = wp.queueHighWaterMark
	wp.mu.Unlock()
	
	memoryMB = wp.estimateQueueMemoryMB(current)
	return
}

// GetDetailedStats returns comprehensive queue statistics
func (wp *WorkerPool) GetDetailedStats() map[string]interface{} {
	current := len(wp.jobsChan)
	capacity := cap(wp.jobsChan)
	
	wp.mu.Lock()
	max := wp.queueHighWaterMark
	wp.mu.Unlock()
	
	return map[string]interface{}{
		"current_size":        current,
		"capacity":           capacity,
		"max_size_reached":   max,
		"percent_full":       float64(current) / float64(capacity) * 100,
		"percent_of_max":     float64(current) / float64(max) * 100,
		"current_memory_mb":  wp.estimateQueueMemoryMB(current),
		"peak_memory_mb":     wp.estimateQueueMemoryMB(int(max)),
		"capacity_memory_mb": wp.estimateQueueMemoryMB(capacity),
		"num_workers":        wp.numWorkers,
	}
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
