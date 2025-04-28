package service

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

func saveUser(user *spotify.User) {
	dbUser := convertSpotifyUserToDBUser(user)
	err := db.SaveUser(dbUser)
	if err != nil {
		log.Print(err)
	} else {
		log.Printf("User saved: %s", user.Id)
	}
}

// TODO: Use generics

func processTracks(userId string, items any, source string) error {
	tracks, ok := items.([]*spotify.Track)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Track")
	}
	if len(tracks) == 0 {
		log.Printf("No tracks to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d tracks for user %s from source %s", len(tracks), userId, source)
	var trackData []*db.Track
	for _, track := range tracks {
		if track != nil && track.Id != "" {
			dbTrack := convertSpotifyTrackToDBTrack(track)
			trackData = append(trackData, dbTrack)
		}
	}

	if len(trackData) == 0 {
		log.Printf("No valid tracks found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SaveTracks(userId, trackData, source)
	if err != nil {
		log.Printf("Error saving %d tracks for user %s from source %s: %v", len(trackData), userId, source, err)
		return fmt.Errorf("saving %d tracks from %s: %w", len(trackData), source, err)
	}
	log.Printf("Saved %d tracks for user %s from source %s", len(trackData), userId, source)
	return nil
}

func processPlaylists(userId string, items any, source string) error {
	playlists, ok := items.([]*spotify.Playlist)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Playlist")
	}
	if len(playlists) == 0 {
		log.Printf("No playlists to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d playlists for user %s from source %s", len(playlists), userId, source)
	var playlistData []*db.Playlist
	for _, playlist := range playlists {
		if playlist != nil && playlist.Id != "" {
			dbPlaylist := convertSpotifyPlaylistToDBPlaylist(playlist)
			playlistData = append(playlistData, dbPlaylist)
		}
	}

	if len(playlistData) == 0 {
		log.Printf("No valid playlists found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SavePlaylists(userId, playlistData, source)
	if err != nil {
		log.Printf("Error saving %d playlists for user %s from source %s: %v", len(playlistData), userId, source, err)
		return fmt.Errorf("saving %d playlists from %s: %w", len(playlistData), source, err)
	}
	log.Printf("Saved %d playlists for user %s from source %s", len(playlistData), userId, source)
	return nil
}

func processArtists(userId string, items any, source string) error {
	artists, ok := items.([]*spotify.Artist)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Artist")
	}
	if len(artists) == 0 {
		log.Printf("No artists to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d artists for user %s from source %s", len(artists), userId, source)
	var artistData []*db.Artist
	for _, artist := range artists {
		if artist != nil && artist.Id != "" {
			dbPlaylist := convertSpotifyArtistToDBArtist(artist)
			artistData = append(artistData, dbPlaylist)
		}
	}

	if len(artistData) == 0 {
		log.Printf("No valid artists found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SaveArtists(userId, artistData, source)
	if err != nil {
		log.Printf("Error saving %d artists for user %s from source %s: %v", len(artistData), userId, source, err)
		return fmt.Errorf("saving %d artists from %s: %w", len(artistData), source, err)
	}
	log.Printf("Saved %d artists for user %s from source %s", len(artistData), userId, source)
	return nil
}

func processAlbums(userId string, items any, source string) error {
	albums, ok := items.([]*spotify.Album)
	if !ok {
		return errors.New("invalid type: expected []*spotify.Album")
	}
	if len(albums) == 0 {
		log.Printf("No albums to save for user %s from source %s", userId, source)
		return nil
	}

	log.Printf("Saving %d albums for user %s from source %s", len(albums), userId, source)
	var albumData []*db.Album
	for _, album := range albums {
		if album != nil && album.Id != "" {
			dbAlbum := convertSpotifyAlbumToDBAlbum(album)
			albumData = append(albumData, dbAlbum)
		}
	}

	if len(albumData) == 0 {
		log.Printf("No valid albums found to save for user %s from source %s", userId, source)
		return nil
	}

	err := db.SaveAlbums(userId, albumData, source)
	if err != nil {
		log.Printf("Error saving %d albums for user %s from source %s: %v", len(albumData), userId, source, err)
		return fmt.Errorf("saving %d albums from %s: %w", len(albumData), source, err)
	}
	log.Printf("Saved %d albums for user %s from source %s", len(albumData), userId, source)
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
	var allErrors []error            // Slice to collect errors
	var errorMu sync.Mutex           // Mutex to protect allErrors slice

	// Goroutine to collect errors from the results channel safely
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

	// User's top tracks
	log.Print("Getting user's top tracks")
	usersTopTracks, err := spotify.GetUsersTopTracks(token)
	if err != nil {
		log.Printf("Error getting user's top tracks: %v", err)
		errorMu.Lock()
		allErrors = append(allErrors, fmt.Errorf("getting top tracks: %w", err))
		errorMu.Unlock()
	} else if len(usersTopTracks) > 0 {
		var tracksToProcess []*spotify.Track
		for _, track := range usersTopTracks {
			if track != nil && track.Id != "" && !tracker.CheckAndMark("track", track.Id) {
				tracksToProcess = append(tracksToProcess, track)
			}
		}
		if len(tracksToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "top tracks",
				DataType:  "tracks",
				Items:     tracksToProcess,
				ProcessFn: processTracks,
			}, &jobWg)
		}
	}

	// User's saved tracks
	log.Printf("Getting user's saved tracks")
	usersSavedTracks, err := spotify.GetUsersSavedTracks(token)
	if err != nil {
		log.Printf("Error getting user's saved tracks: %v", err)
		errorMu.Lock()
		allErrors = append(allErrors, fmt.Errorf("getting saved tracks: %w", err))
		errorMu.Unlock()
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
				Source:    "saved tracks",
				DataType:  "tracks",
				Items:     tracksToProcess,
				ProcessFn: processTracks,
			}, &jobWg)
		}
	}

	// User's playlists
	log.Printf("Getting user's playlists")
	usersPlaylists, err := spotify.GetUsersPlaylists(token)
	if err != nil {
		log.Printf("Error getting user's playlists: %v", err)
		errorMu.Lock()
		allErrors = append(allErrors, fmt.Errorf("getting playlists: %w", err))
		errorMu.Unlock()
	} else {
		if len(usersPlaylists) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "playlists",
				DataType:  "playlists",
				Items:     usersPlaylists,
				ProcessFn: processPlaylists,
			}, &jobWg)
		}
		for _, playlist := range usersPlaylists {
			if playlist != nil && playlist.Id != "" {
				pool.Submit(&FetchAndProcessPlaylistTracksJob{
					UserID:     userId,
					Token:      token,
					PlaylistID: playlist.Id,
					Source:     "playlists",
				}, &jobWg)
			}
		}
	}

	//TODO: split artists and albums
	// User's top and followed artists
	log.Printf("Getting user's top and followed artists")
	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		log.Printf("Error getting user's top artists: %v", err)
		errorMu.Lock()
		allErrors = append(allErrors, fmt.Errorf("getting top artists: %w", err))
		errorMu.Unlock()
	}
	usersFollowedArtists, err := spotify.GetUsersFollowedArtists(token)
	if err != nil {
		log.Printf("Error getting user's followed artists: %v", err)
		errorMu.Lock()
		allErrors = append(allErrors, fmt.Errorf("getting followed artists: %w", err))
		errorMu.Unlock()
	}
	combinedArtists := append(usersTopArtists, usersFollowedArtists...)
	uniqueArtists := removeDuplicateArtists(combinedArtists)

	if len(uniqueArtists) > 0 {
		var artistsToProcess []*spotify.Artist
		var artistIdsForSubDataFetch []string
		for _, artist := range uniqueArtists {
			if artist != nil && artist.Id != "" {
				if !tracker.CheckAndMark("artist", artist.Id) {
					artistsToProcess = append(artistsToProcess, artist)
					artistIdsForSubDataFetch = append(artistIdsForSubDataFetch, artist.Id)
				} else {
					log.Printf("Skipping already processed artist %s", artist.Id)
				}
			}
		}

		if len(artistsToProcess) > 0 {
			pool.Submit(&ProcessDataJob{
				UserID:    userId,
				Source:    "artists",
				DataType:  "artists",
				Items:     artistsToProcess,
				ProcessFn: processArtists,
			}, &jobWg)
		}

		for _, artistId := range artistIdsForSubDataFetch {
			pool.Submit(&FetchArtistSubDataJob{
				UserID:   userId,
				ArtistID: artistId,
			}, &jobWg)
		}
	}

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
				Source:    "saved albums",
				DataType:  "albums",
				Items:     albumsToProcess,
				ProcessFn: processAlbums,
			}, &jobWg)
		}

		for _, albumId := range albumIdsForTrackFetch {
			pool.Submit(&FetchAndProcessAlbumTracksJob{
				UserID:  userId,
				AlbumID: albumId,
				Source:  "saved albums",
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
