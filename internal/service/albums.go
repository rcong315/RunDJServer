package service

import (
	"fmt"
	"log"
	"sync"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type SaveAlbumTracksJob struct {
	AlbumId string
}

func (j *SaveAlbumTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker) error {
	albumId := j.AlbumId

	albumTracks, err := spotify.GetAlbumsTracks(albumId)
	if err != nil {
		return fmt.Errorf("getting tracks for album %s: %w", albumId, err)
	}
	if len(albumTracks) == 0 {
		return nil
	}

	dbTracks := convertSpotifyTracksToDBTracks(albumTracks)
	var tracksToSave []*db.Track
	for _, track := range dbTracks {
		if !tracker.CheckAndMark("track", track.TrackId) {
			tracksToSave = append(tracksToSave, track)
		}
	}

	err = db.SaveTracks(tracksToSave)
	if err != nil {
		return fmt.Errorf("saving tracks: %w, tracks: %d", err, len(tracksToSave))
	}

	err = db.SaveAlbumTracks(albumId, dbTracks)
	if err != nil {
		return fmt.Errorf("saving album tracks: %w", err)
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

	dbAlbums := convertSpotifyAlbumsToDBAlbums(usersSavedAlbums)
	var albumsToSave []*db.Album
	for _, album := range dbAlbums {
		if !tracker.CheckAndMark("album", album.AlbumId) {
			albumsToSave = append(albumsToSave, album)
		}
	}

	err = db.SaveAlbums(albumsToSave)
	if err != nil {
		return fmt.Errorf("saving albums: %w", err)
	}

	err = db.SaveUserSavedAlbums(userId, dbAlbums)
	if err != nil {
		return fmt.Errorf("saving user-album relation: %w", err)
	}

	for _, album := range usersSavedAlbums {
		pool.Submit(&SaveAlbumTracksJob{
			AlbumId: album.Id,
		}, jobWg)
	}

	return nil
}
