package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/rcong315/RunDJServer/internal/spotify"
)

func processTopTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Print("Getting user's top tracks")
	usersTopTracks, err := spotify.GetUsersTopTracks(token)
	if err != nil {
		return fmt.Errorf("getting top tracks: %w", err)
	}
	if len(usersTopTracks) == 0 {
		return nil
	}

	pool.Submit(&ProcessDataJob{
		UserID:    userId,
		Source:    "top_tracks",
		DataType:  "tracks",
		Items:     usersTopTracks,
		ProcessFn: saveTracks,
	}, jobWg)

	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's saved tracks")
	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
	if err != nil {
		return fmt.Errorf("getting saved tracks: %w", err)
	}
	if len(usersSavedTracks) == 0 {
		return nil
	}

	pool.Submit(&ProcessDataJob{
		UserID:    userId,
		Source:    "saved_tracks",
		DataType:  "tracks",
		Items:     usersSavedTracks,
		ProcessFn: saveTracks,
	}, jobWg)

	return nil
}

// TODO: Clean up nested size = 0 checks

func processPlaylists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's playlists")
	usersPlaylists, err := spotify.GetUsersPlaylists(token)
	if err != nil {
		return fmt.Errorf("getting playlists: %w", err)
	}
	if len(usersPlaylists) == 0 {
		return nil
	}

	err = savePlaylists(userId, usersPlaylists, "playlists", tracker)
	if err != nil {
		return fmt.Errorf("saving playlists: %w", err)
	}

	for _, playlist := range usersPlaylists {
		if playlist != nil && playlist.Id != "" {
			pool.Submit(&FetchAndProcessPlaylistTracksJob{
				UserID:     userId,
				Token:      token,
				PlaylistID: playlist.Id,
				Source:     "playlists",
			}, jobWg)
		}
	}

	return nil
}

func processTopArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's top and followed artists")
	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		return fmt.Errorf("getting top artists: %w", err)
	}
	if len(usersTopArtists) == 0 {
		return nil
	}

	err = saveArtists(userId, usersTopArtists, "top_artists", tracker)
	if err != nil {
		return fmt.Errorf("saving top artists: %w", err)
	}

	for _, artist := range usersTopArtists {
		pool.Submit(&FetchArtistSubDataJob{
			UserID:   userId,
			ArtistID: artist.Id,
			Source:   "top_artists",
		}, jobWg)
	}

	return nil
}

func processFollowedArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's followed artists")
	usersFollowedArtists, err := spotify.GetUsersFollowedArtists(token)
	if err != nil {
		return fmt.Errorf("getting followed artists: %w", err)
	}
	if len(usersFollowedArtists) == 0 {
		return nil
	}

	err = saveArtists(userId, usersFollowedArtists, "followed_artists", tracker)
	if err != nil {
		return fmt.Errorf("saving followed artists: %w", err)
	}

	for _, artist := range usersFollowedArtists {
		pool.Submit(&FetchArtistSubDataJob{
			UserID:   userId,
			ArtistID: artist.Id,
			Source:   "followed_artists",
		}, jobWg)
	}

	return nil
}

func processSavedAlbums(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's saved albums")
	usersSavedAlbums, err := spotify.GetUsersSavedAlbums(token)
	if err != nil {
		return fmt.Errorf("getting saved albums: %w", err)
	}
	if len(usersSavedAlbums) == 0 {
		return nil
	}

	err = saveAlbums(userId, usersSavedAlbums, "saved_albums", tracker)
	if err != nil {
		return fmt.Errorf("saving albums: %w", err)
	}

	for _, album := range usersSavedAlbums {
		pool.Submit(&FetchAndProcessAlbumTracksJob{
			UserID:  userId,
			AlbumID: album.Id,
			Source:  "saved_albums",
		}, jobWg)
	}

	return nil
}

func processPlaylistTracks(userId string, token string, playlistId string, source string, tracker *ProcessedTracker) error {
	log.Printf("Job: Getting tracks from playlist: %s", playlistId)
	playlistTracks, err := spotify.GetPlaylistsTracks(token, playlistId)
	if err != nil {
		return fmt.Errorf("getting tracks for playlist %s: %w", playlistId, err)
	}

	if len(playlistTracks) == 0 {
		return nil
	}

	log.Printf("Job: Submitting processing for %d tracks from playlist %s", len(playlistTracks), playlistId)
	err = saveTracks(userId, playlistTracks, source, tracker)
	if err != nil {
		return fmt.Errorf("processing tracks for playlist %s: %w", playlistId, err)
	}

	return nil
}

// TODO: When to run this function? On register and when else? Cron?
// TODO: Add release radar playlist
// TODO: Try seperating files by track, playlist, artist, etc.?
func processAll(token string, userId string) {
	log.Printf("Starting data processing for user %s", userId)

	numWorkers := 10       // Adjust based on resources and API limits
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

	log.Printf("All initial fetches done, waiting for processing jobs to complete...")
	jobWg.Wait()

	log.Printf("All processing jobs completed, stopping worker pool...")
	pool.Stop()

	errorCollectionWg.Wait()
	log.Printf("Worker pool stopped.")

	errorMu.Lock()
	defer errorMu.Unlock()
	if len(allErrors) > 0 {
		finalError := fmt.Errorf("processing failed with %d errors", len(allErrors))
		for i, e := range allErrors {
			finalError = fmt.Errorf("%w; [%d]: %v", finalError, i+1, e)
			log.Printf("Error %d: %v", i+1, e)
		}
		log.Printf("Finished processing for user %s with %d errors.", userId, len(allErrors))
	} else {
		log.Printf("Finished processing for user %s with 0 errors", userId)
	}

	// TODO: fix deadlock issue
}
