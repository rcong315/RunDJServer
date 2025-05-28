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

func (j *SaveArtistTopTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
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

func (j *SaveArtistAlbumsJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	artistId := j.ArtistId

	var allDbAlbums []*db.Album
	var mu sync.Mutex
	
	err := spotify.GetArtistsAlbumsAndSinglesStreaming(artistId, func(albums []*spotify.Album) error {
		dbAlbums := convertSpotifyAlbumsToDBAlbums(albums)
		
		var albumsToSave []*db.Album
		for _, album := range dbAlbums {
			if !tracker.CheckAndMark("album", album.AlbumId) {
				albumsToSave = append(albumsToSave, album)
			}
		}
		
		if len(albumsToSave) > 0 {
			if err := db.SaveAlbums(albumsToSave); err != nil {
				return fmt.Errorf("saving albums: %w, albums: %d", err, len(albumsToSave))
			}
		}
		
		// Submit jobs for album tracks immediately
		for _, album := range albums {
			pool.SubmitWithStage(&SaveAlbumTracksJob{
				AlbumId: album.Id,
			}, jobWg, stage)
		}
		
		mu.Lock()
		allDbAlbums = append(allDbAlbums, dbAlbums...)
		mu.Unlock()
		
		return nil
	})
	
	if err != nil {
		return fmt.Errorf("getting albums for artist %s: %w", artistId, err)
	}
	
	if len(allDbAlbums) == 0 {
		return nil
	}

	err = db.SaveArtistAlbums(artistId, allDbAlbums)
	if err != nil {
		return fmt.Errorf("saving artist albums: %w", err)
	}

	return nil
}

func processTopArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Getting user's top artists", zap.String("userId", userId))
	
	var allDbArtists []*db.Artist
	var allSpotifyArtists []*spotify.Artist
	var mu sync.Mutex
	
	err := spotify.GetUsersTopArtistsStreaming(token, func(artists []*spotify.Artist) error {
		dbArtists := convertSpotifyArtistsToDBArtists(artists)
		
		var artistsToSave []*db.Artist
		for _, artist := range dbArtists {
			if !tracker.CheckAndMark("artist", artist.ArtistId) {
				artistsToSave = append(artistsToSave, artist)
			}
		}
		
		if len(artistsToSave) > 0 {
			if err := db.SaveArtists(artistsToSave); err != nil {
				logger.Error("Error saving top artists batch to DB",
					zap.String("userId", userId),
					zap.Int("artistsToSaveCount", len(artistsToSave)),
					zap.Error(err))
				return fmt.Errorf("saving top artists: %w", err)
			}
		}
		
		// Submit jobs immediately
		for _, artist := range artists {
			pool.SubmitWithStage(&SaveArtistTopTracksJob{
				ArtistId: artist.Id,
				Type:     TopArtists,
			}, jobWg, stage)
			pool.SubmitWithStage(&SaveArtistAlbumsJob{
				ArtistId: artist.Id,
			}, jobWg, stage)
		}
		
		mu.Lock()
		allDbArtists = append(allDbArtists, dbArtists...)
		allSpotifyArtists = append(allSpotifyArtists, artists...)
		mu.Unlock()
		
		logger.Debug("Processed batch of top artists", 
			zap.String("userId", userId), 
			zap.Int("batchSize", len(artists)),
			zap.Int("savedCount", len(artistsToSave)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting user's top artists", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting top artists: %w", err)
	}
	
	if len(allDbArtists) == 0 {
		return nil
	}

	err = db.SaveUserTopArtists(userId, allDbArtists)
	if err != nil {
		return fmt.Errorf("saving user-top artists relations: %w", err)
	}

	return nil
}

func processFollowedArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Getting user's followed artists", zap.String("userId", userId))
	
	var allDbArtists []*db.Artist
	var allSpotifyArtists []*spotify.Artist
	var mu sync.Mutex
	
	err := spotify.GetUsersFollowedArtistsStreaming(token, func(artists []*spotify.Artist) error {
		dbArtists := convertSpotifyArtistsToDBArtists(artists)
		
		var artistsToSave []*db.Artist
		for _, artist := range dbArtists {
			if !tracker.CheckAndMark("artist", artist.ArtistId) {
				artistsToSave = append(artistsToSave, artist)
			}
		}
		
		if len(artistsToSave) > 0 {
			if err := db.SaveArtists(artistsToSave); err != nil {
				logger.Error("Error saving followed artists batch to DB",
					zap.String("userId", userId),
					zap.Int("artistsToSaveCount", len(artistsToSave)),
					zap.Error(err))
				return fmt.Errorf("saving followed artists: %w", err)
			}
		}
		
		// Submit jobs immediately
		for _, artist := range artists {
			pool.SubmitWithStage(&SaveArtistTopTracksJob{
				ArtistId: artist.Id,
				Type:     FollowedArtists,
			}, jobWg, stage)
			pool.SubmitWithStage(&SaveArtistAlbumsJob{
				ArtistId: artist.Id,
			}, jobWg, stage)
		}
		
		mu.Lock()
		allDbArtists = append(allDbArtists, dbArtists...)
		allSpotifyArtists = append(allSpotifyArtists, artists...)
		mu.Unlock()
		
		logger.Debug("Processed batch of followed artists", 
			zap.String("userId", userId), 
			zap.Int("batchSize", len(artists)),
			zap.Int("savedCount", len(artistsToSave)))
		return nil
	})
	
	if err != nil {
		logger.Error("Error getting user's followed artists", zap.String("userId", userId), zap.Error(err))
		return fmt.Errorf("getting followed artists: %w", err)
	}
	
	if len(allDbArtists) == 0 {
		return nil
	}

	err = db.SaveUserFollowedArtists(userId, allDbArtists)
	if err != nil {
		return fmt.Errorf("saving user-followed artists relations: %w", err)
	}

	return nil
}
