package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

// --- Worker Pool Setup ---

// Job represents a task for a worker to execute.
// We use an interface to allow different kinds of tasks.
type Job interface {
	Execute(pool *WorkerPool, jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) error
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
func (wp *WorkerPool) Start(jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) {
	for i := range wp.numWorkers {
		wp.wg.Add(1)
		// Pass necessary context (pool, jobWg, tracker) to the worker
		go wp.worker(i+1, jobWg, processedTracker)
	}
}

// worker is the function executed by each worker goroutine.
func (wp *WorkerPool) worker(id int, jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) {
	defer wp.wg.Done()
	log.Printf("Worker %d started", id)
	for job := range wp.jobsChan {
		log.Printf("Worker %d processing job: %T", id, job)
		// Pass context down to the job's Execute method
		err := job.Execute(wp, jobWg, processedTracker)
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
	mu               sync.Mutex
	processedTracks  map[string]struct{}
	processedArtists map[string]struct{}
	processedAlbums  map[string]struct{}
	processedSingles map[string]struct{}
}

func NewProcessedTracker() *ProcessedTracker {
	return &ProcessedTracker{
		processedTracks:  make(map[string]struct{}),
		processedArtists: make(map[string]struct{}),
		processedAlbums:  make(map[string]struct{}),
		processedSingles: make(map[string]struct{}),
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

// --- Job Implementations ---

// Define concrete job types for each task

type SaveUserJob struct {
	User *spotify.User
}

type ProcessDataJob struct {
	UserID    string
	Source    string
	DataType  string
	Items     any // Use interface{} or generics (Go 1.18+)
	ProcessFn func(userId string, items any, source string, tracker *ProcessedTracker) error
}

type FetchAndProcessPlaylistTracksJob struct {
	UserID     string
	Token      string
	PlaylistID string
	Source     string
}

type FetchArtistSubDataJob struct {
	UserID   string
	ArtistID string
	Source   string
}

type FetchAndProcessAlbumTracksJob struct {
	UserID  string
	AlbumID string
	Source  string
}

func (j *SaveUserJob) Execute() error {
	dbUser := convertSpotifyUserToDBUser(j.User)
	err := db.SaveUser(dbUser)
	if err != nil {
		log.Printf("Error saving user %s: %v", j.User.Id, err)
		return fmt.Errorf("saving user %s: %w", j.User.Id, err) // Wrap error
	}
	log.Printf("User saved: %s", j.User.Id)
	return nil
}

func (j *ProcessDataJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) error {
	// Note: This job structure assumes the data (Items) is already fetched.
	// You might need different job types if the job itself needs to fetch data.
	err := j.ProcessFn(j.UserID, j.Items, j.Source, processedTracker)
	if err != nil {
		// Error is already logged in ProcessFn, just return it
		return fmt.Errorf("processing %s for user %s from source %s: %w", j.DataType, j.UserID, j.Source, err)
	}
	return nil
}

func (j *FetchAndProcessPlaylistTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) error {
	return processPlaylistTracks(j.UserID, j.Token, j.PlaylistID, j.Source, processedTracker)
}

// TODO: move to processItems
func (j *FetchArtistSubDataJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) error {
	var artistErrors []error

	log.Printf("Job: Getting top tracks for artist: %s", j.ArtistID)
	artistTopTracks, err := spotify.GetArtistsTopTracks(j.ArtistID)
	if err != nil {
		log.Printf("Error getting top tracks for artist %s: %v", j.ArtistID, err)
		artistErrors = append(artistErrors, fmt.Errorf("getting top tracks for artist %s: %w", j.ArtistID, err))
	} else if len(artistTopTracks) > 0 {
		log.Printf("Submitting job for %d top tracks from artist %s", len(artistTopTracks), j.ArtistID)
		pool.Submit(&ProcessDataJob{
			UserID:    j.UserID,
			Source:    j.Source + "_top_tracks",
			DataType:  "tracks",
			Items:     artistTopTracks,
			ProcessFn: saveTracks,
		}, jobWg)
	}

	log.Printf("Job: Getting albums for artist: %s", j.ArtistID)
	albumsAndSingles, err := spotify.GetArtistsAlbumsAndSingles(j.ArtistID)
	if err != nil {
		log.Printf("Error getting albums for artist %s: %v", j.ArtistID, err)
		artistErrors = append(artistErrors, fmt.Errorf("getting albums for artist %s: %w", j.ArtistID, err))
	}

	var artistAlbums, artistSingles []*spotify.Album
	for _, item := range albumsAndSingles {
		if item.AlbumType == "album" {
			artistAlbums = append(artistAlbums, item)
		} else if item.AlbumType == "single" {
			artistSingles = append(artistSingles, item)
		}
	}

	if len(artistAlbums) > 0 {
		log.Printf("Submitting job for %d albums metadata from artist %s", len(artistAlbums), j.ArtistID)
		pool.Submit(&ProcessDataJob{
			UserID:    j.UserID,
			Source:    j.Source + "_album",
			DataType:  "albums",
			Items:     artistAlbums,
			ProcessFn: saveAlbums,
		}, jobWg)

		for _, album := range artistAlbums {
			log.Printf("Submitting job to fetch tracks for album %s (from artist %s)", album.Id, j.ArtistID)
			pool.Submit(&FetchAndProcessAlbumTracksJob{
				UserID:  j.UserID,
				AlbumID: album.Id,
				Source:  j.Source + "_album",
			}, jobWg)
		}
	}

	if len(artistSingles) > 0 {
		log.Printf("Submitting job for %d singles metadata from artist %s", len(artistSingles), j.ArtistID)
		pool.Submit(&ProcessDataJob{
			UserID:    j.UserID,
			Source:    j.Source + "_singles",
			DataType:  "singles",
			Items:     artistSingles,
			ProcessFn: saveAlbums,
		}, jobWg)

		for _, single := range artistSingles {
			log.Printf("Submitting job to fetch tracks for single %s (from artist %s)", single.Id, j.ArtistID)
			pool.Submit(&FetchAndProcessAlbumTracksJob{
				UserID:  j.UserID,
				AlbumID: single.Id,
				Source:  j.Source + "_singles",
			}, jobWg)
		}
	}

	// Aggregate errors from this artist's sub-tasks
	if len(artistErrors) > 0 {
		// TODO: Use errors.Join (Go 1.20+) or simple wrapping
		return fmt.Errorf("failed fetching sub-data for artist %s: %v", j.ArtistID, artistErrors) // Basic wrapping
	}
	return nil
}

func (j *FetchAndProcessAlbumTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, processedTracker *ProcessedTracker) error {
	albumId := j.AlbumID
	log.Printf("Job: Getting tracks from album: %s", albumId)
	albumTracks, err := spotify.GetAlbumsTracks(albumId)
	if err != nil {
		return fmt.Errorf("getting tracks for album %s: %w", albumId, err)
	}

	if len(albumTracks) == 0 {
		log.Printf("No tracks found for album %s", albumId)
		return nil
	}

	err = saveTracks(j.UserID, albumTracks, j.Source, processedTracker)
	if err != nil {
		return fmt.Errorf("saving tracks for album %s: %w", albumId, err)
	}

	return nil
}
