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
		log.Printf("Error getting user's top tracks: %v", err)
		return fmt.Errorf("getting top tracks: %w", err)
	}

	if len(usersTopTracks) > 0 {
		var tracksToProcess []*spotify.Track
		for _, track := range usersTopTracks {
			if track != nil && track.Id != "" && !tracker.CheckAndMark("track", track.Id) {
				tracksToProcess = append(tracksToProcess, track)
			}
		}
		if len(tracksToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "top_tracks",
				DataType:  "tracks",
				Items:     tracksToProcess,
				ProcessFn: saveTracks,
			}, jobWg)
		}
	}

	return nil
}

func processSavedTracks(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's saved tracks")
	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
	if err != nil {
		log.Printf("Error getting user's saved tracks: %v", err)
		return fmt.Errorf("getting saved tracks: %w", err)
	} else if len(usersSavedTracks) > 0 {
		var tracksToProcess []*spotify.Track
		for _, track := range usersSavedTracks {
			if track != nil && track.Id != "" && !tracker.CheckAndMark("track", track.Id) {
				tracksToProcess = append(tracksToProcess, track)
			}
		}
		if len(tracksToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "saved_tracks",
				DataType:  "tracks",
				Items:     tracksToProcess,
				ProcessFn: saveTracks,
			}, jobWg)
		}
	}

	return nil
}

func processPlaylists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's playlists")
	usersPlaylists, err := spotify.GetUsersPlaylists(token)
	if err != nil {
		log.Printf("Error getting user's playlists: %v", err)
		return fmt.Errorf("getting playlists: %w", err)
	} else {
		if len(usersPlaylists) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "playlists",
				DataType:  "playlists",
				Items:     usersPlaylists,
				ProcessFn: savePlaylists,
			}, jobWg)
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
	}

	return nil
}

func processTopArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's top and followed artists")
	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		log.Printf("Error getting user's top artists: %v", err)
		return fmt.Errorf("getting top artists: %w", err)
	}

	if len(usersTopArtists) > 0 {
		var artistsToProcess []*spotify.Artist
		for _, artist := range usersTopArtists {
			if artist != nil && artist.Id != "" {
				if !tracker.CheckAndMark("artist", artist.Id) {
					artistsToProcess = append(artistsToProcess, artist)
				} else {
					log.Printf("Skipping already processed artist %s", artist.Id)
				}
			}
		}

		if len(artistsToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "top_artists",
				DataType:  "artists",
				Items:     artistsToProcess,
				ProcessFn: saveArtists,
			}, jobWg)
		}

		for _, artist := range artistsToProcess {
			pool.Submit(&FetchArtistSubDataJob{
				UserID:   userId,
				ArtistID: artist.Id,
			}, jobWg)
		}
	}

	return nil
}

func processFollowedArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's followed artists")
	usersFollowedArtists, err := spotify.GetUsersFollowedArtists(token)
	if err != nil {
		log.Printf("Error getting user's followed artists: %v", err)
		return fmt.Errorf("getting followed artists: %w", err)
	}

	if len(usersFollowedArtists) > 0 {
		var artistsToProcess []*spotify.Artist
		for _, artist := range usersFollowedArtists {
			if artist != nil && artist.Id != "" {
				if !tracker.CheckAndMark("artist", artist.Id) {
					artistsToProcess = append(artistsToProcess, artist)
				} else {
					log.Printf("Skipping already processed artist %s", artist.Id)
				}
			}
		}

		if len(artistsToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "saved_artists",
				DataType:  "artists",
				Items:     artistsToProcess,
				ProcessFn: saveArtists,
			}, jobWg)
		}

		for _, artist := range artistsToProcess {
			pool.Submit(&FetchArtistSubDataJob{
				UserID:   userId,
				ArtistID: artist.Id,
			}, jobWg)
		}
	}

	return nil
}

// TODO: When to run this function? On register and when else? Cron?
// TODO: Add release radar playlist
func processAll(token string, userId string) {
	log.Printf("Starting data processing for user %s", userId)

	numWorkers := 25       // Adjust based on resources and API limits
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

	//TODO: split artists and albums

	// User's saved albums
	log.Printf("Getting user's saved albums")
	usersSavedAlbums, err := spotify.GetUsersSavedAlbums(token)
	if err != nil {
		log.Printf("Error getting user's saved albums: %v", err)
		errorMu.Lock()
		allErrors = append(allErrors, fmt.Errorf("getting saved albums: %w", err))
		errorMu.Unlock()
	} else if len(usersSavedAlbums) > 0 {
		var albumsToProcess []*spotify.Album
		var albumIdsForTrackFetch []string
		for _, album := range usersSavedAlbums {
			if album != nil && album.Id != "" {
				if !tracker.CheckAndMark("album", album.Id) {
					albumsToProcess = append(albumsToProcess, album)
					albumIdsForTrackFetch = append(albumIdsForTrackFetch, album.Id)
				} else {
					log.Printf("Skipping already processed saved album %s", album.Id)
				}
			}
		}

		if len(albumsToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "saved_albums",
				DataType:  "albums",
				Items:     albumsToProcess,
				ProcessFn: processAlbums,
			}, &jobWg)
		}

		for _, albumId := range albumIdsForTrackFetch {
			pool.Submit(&FetchAndProcessAlbumTracksJob{
				UserID:  userId,
				AlbumID: albumId,
				Source:  "saved_albums",
			}, &jobWg)
		}
	}

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
	}

	log.Printf("Finished processing for user %s", userId)
}
