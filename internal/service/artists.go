package service

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type ArtistType int

const (
	TopArtists ArtistType = iota
	FollowedArtists
)

type SaveArtistTopTracksJob struct {
	ArtistId string
	Type     ArtistType
}

type SaveArtistAlbumsJob struct {
	ArtistId string
}

func (j *SaveArtistTopTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker) error {
	artistId := j.ArtistId

	artistTopTracks, err := spotify.GetArtistsTopTracks(artistId)
	if err != nil {
		return fmt.Errorf("getting top tracks for artist %s: %w", artistId, err)
	}
	if len(artistTopTracks) == 0 {
		return nil
	}

	dbTracks := convertSpotifyTracksToDBTracks(artistTopTracks)
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

	err = db.SaveArtistTopTracks(artistId, dbTracks)
	if err != nil {
		return fmt.Errorf("saving artist top tracks: %w", err)
	}

	return nil
}

func (j *SaveArtistAlbumsJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker) error {
	artistId := j.ArtistId

	artistAlbums, err := spotify.GetArtistsAlbumsAndSingles(artistId)
	if err != nil {
		return fmt.Errorf("getting albums for artist %s: %w", artistId, err)
	}
	if len(artistAlbums) == 0 {
		return nil
	}

	dbAlbums := convertSpotifyAlbumsToDBAlbums(artistAlbums)
	var albumsToSave []*db.Album
	for _, album := range dbAlbums {
		if !tracker.CheckAndMark("album", album.AlbumId) {
			albumsToSave = append(albumsToSave, album)
		}
	}

	err = db.SaveAlbums(albumsToSave)
	if err != nil {
		return fmt.Errorf("saving albums: %w, albums: %d", err, len(albumsToSave))
	}

	err = db.SaveArtistAlbums(artistId, dbAlbums)
	if err != nil {
		return fmt.Errorf("saving artist albums: %w", err)
	}

	for _, album := range artistAlbums {
		pool.Submit(&SaveAlbumTracksJob{
			AlbumId: album.Id,
		}, jobWg)
	}

	return nil
}

func processTopArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	logger.Info("Getting user's top artists", zap.String("userId", userId))
	usersTopArtists, err := spotify.GetUsersTopArtists(token)
	if err != nil {
		logger.Error("Error getting user's top artists", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting top artists: %w", err)
	}
	if len(usersTopArtists) == 0 {
		return nil
	}

	dbArtists := convertSpotifyArtistsToDBArtists(usersTopArtists)
	var artistsToSave []*db.Artist
	for _, artist := range dbArtists {
		if !tracker.CheckAndMark("artist", artist.ArtistId) {
			artistsToSave = append(artistsToSave, artist)
		}
	}

	err = db.SaveArtists(artistsToSave)
	if err != nil {
		return fmt.Errorf("saving top artists: %w", err)
	}

	err = db.SaveUserTopArtists(userId, dbArtists)
	if err != nil {
		return fmt.Errorf("saving user-top artists relations: %w", err)
	}

	for _, artist := range usersTopArtists {
		pool.Submit(&SaveArtistTopTracksJob{
			ArtistId: artist.Id,
			Type:     TopArtists,
		}, jobWg)
		pool.Submit(&SaveArtistAlbumsJob{
			ArtistId: artist.Id,
		}, jobWg)
	}

	return nil
}

func processFollowedArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup) error {
	logger.Info("Getting user's followed artists", zap.String("userId", userId))
	usersFollowedArtists, err := spotify.GetUsersFollowedArtists(token)
	if err != nil {
		logger.Error("Error getting user's followed artists", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting followed artists: %w", err)
	}
	if len(usersFollowedArtists) == 0 {
		return nil
	}

	dbArtists := convertSpotifyArtistsToDBArtists(usersFollowedArtists)
	var artistsToSave []*db.Artist
	for _, artist := range dbArtists {
		if !tracker.CheckAndMark("artist", artist.ArtistId) {
			artistsToSave = append(artistsToSave, artist)
		}
	}

	err = db.SaveArtists(artistsToSave)
	if err != nil {
		return fmt.Errorf("saving followed artists: %w", err)
	}

	err = db.SaveUserFollowedArtists(userId, dbArtists)
	if err != nil {
		return fmt.Errorf("saving user-followed artists relations: %w", err)
	}

	for _, artist := range usersFollowedArtists {
		pool.Submit(&SaveArtistTopTracksJob{
			ArtistId: artist.Id,
			Type:     FollowedArtists,
		}, jobWg)
		pool.Submit(&SaveArtistAlbumsJob{
			ArtistId: artist.Id,
		}, jobWg)
	}

	return nil
}
