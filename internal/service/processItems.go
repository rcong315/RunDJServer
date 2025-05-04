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

func processPlaylistTracks(userId string, token string, playlistId string, source string, tracker *ProcessedTracker) error {
	log.Printf("Job: Getting tracks from playlist: %s", playlistId)
	playlistTracks, err := spotify.GetPlaylistsTracks(token, playlistId)
	if err != nil {
		// Log specific error, but return a wrapped error for aggregation
		log.Printf("Error getting tracks from playlist %s: %v", playlistId, err)
		return fmt.Errorf("getting tracks for playlist %s: %w", playlistId, err)
	}

	if len(playlistTracks) == 0 {
		log.Printf("No tracks found for playlist %s", playlistId)
		return nil // Not an error if playlist is empty
	}

	// --- Deduplication before processing ---
	var tracksToProcess []*spotify.Track
	for _, track := range playlistTracks {
		if track != nil && track.Id != "" {
			if !tracker.CheckAndMark("track", track.Id) {
				tracksToProcess = append(tracksToProcess, track)
			} else {
				log.Printf("Skipping already processed track %s from playlist %s", track.Id, playlistId)
			}
		}
	}

	if len(tracksToProcess) > 0 {
		log.Printf("Job: Submitting processing for %d tracks from playlist %s", len(tracksToProcess), playlistId)
		// Process the filtered tracks (could call processTracks directly or submit another job)
		// Calling directly is simpler here as data is already fetched.
		err := saveTracks(userId, tracksToProcess, source) // Pass filtered list
		if err != nil {
			return fmt.Errorf("processing tracks for playlist %s: %w", playlistId, err)
		}

		err = saveTrackPlaylistRelation(playlistId, tracksToProcess, source)
		if err != nil {
			return fmt.Errorf("saving track-playlist relation for playlist %s: %w", playlistId, err)
		}
	} else {
		log.Printf("No new tracks to process for playlist %s after deduplication", playlistId)
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
				Source:   "top_artists",
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
				Source:    "followed_artists",
				DataType:  "artists",
				Items:     artistsToProcess,
				ProcessFn: saveArtists,
			}, jobWg)
		}

		for _, artist := range artistsToProcess {
			pool.Submit(&FetchArtistSubDataJob{
				UserID:   userId,
				ArtistID: artist.Id,
				Source:   "followed_artists",
			}, jobWg)
		}
	}

	return nil
}

func processSavedAlbums(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	log.Printf("Getting user's saved albums")
	usersSavedAlbums, err := spotify.GetUsersSavedAlbums(token)
	if err != nil {
		log.Printf("Error getting user's saved albums: %v", err)
		return fmt.Errorf("getting saved albums: %w", err)
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
				ProcessFn: saveAlbums,
			}, jobWg)
		}

		for _, albumId := range albumIdsForTrackFetch {
			pool.Submit(&FetchAndProcessAlbumTracksJob{
				UserID:  userId,
				AlbumID: albumId,
				Source:  "saved_albums",
			}, jobWg)
		}
	}

	return nil
}

// TODO: When to run this function? On register and when else? Cron?
// TODO: Add release radar playlist
// TODO: Try seperating files by track, playlist, artist, etc.?
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
}
