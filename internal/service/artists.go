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

func createArtistBatcher(userId string, tracker *ProcessedTracker, saveRelation func(string, []*db.Artist) error) *spotify.BatchProcessor[*spotify.Artist] {
	return spotify.NewBatchProcessor(100, func(artists []*spotify.Artist) error {
		dbArtists := convertSpotifyArtistsToDBArtists(artists)
		var artistsToSave []*db.Artist
		for _, artist := range dbArtists {
			if !tracker.CheckAndMark("artist", artist.ArtistId) {
				artistsToSave = append(artistsToSave, artist)
			}
		}

		if len(artistsToSave) > 0 {
			if err := db.SaveArtists(artistsToSave); err != nil {
				return fmt.Errorf("saving artists batch: %w", err)
			}
			logger.Debug("Saved batch of artists to DB",
				zap.String("userId", userId))
		}

		if err := saveRelation(userId, dbArtists); err != nil {
			return fmt.Errorf("saving user-artist relations: %w", err)
		}

		logger.Debug("Saved user-artist relations to DB",
			zap.String("userId", userId))
		return nil
	})
}

func (j *SaveArtistTopTracksJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	artistId := j.ArtistId

	logger.Debug("Executing SaveArtistTopTracksJob",
		zap.String("artistId", artistId))

	trackBatcher := createTrackBatcher("artist", artistId, tracker, db.SaveArtistTopTracks)

	err := spotify.GetArtistsTopTracks(artistId, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				return fmt.Errorf("adding track to batch: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting top tracks for artist %s: %w", artistId, err)
	}

	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing track batcher: %w", err)
	}

	logger.Debug("Executed SaveArtistTopTracksJob",
		zap.String("artistId", artistId))
	return nil
}

func (j *SaveArtistAlbumsJob) Execute(pool *WorkerPool, jobWg *sync.WaitGroup, tracker *ProcessedTracker, stage *StageContext) error {
	artistId := j.ArtistId

	logger.Debug("Executing SaveArtistAlbumsJob",
		zap.String("artistId", artistId))

	albumBatcher := createAlbumBatcher("artist", artistId, tracker, db.SaveArtistAlbums)

	err := spotify.GetArtistsAlbumsAndSingles(artistId, func(albums []*spotify.Album) error {
		for _, album := range albums {
			if err := albumBatcher.Add(album); err != nil {
				return fmt.Errorf("adding album to batch: %w", err)
			}
			pool.SubmitWithStage(&SaveAlbumTracksJob{
				AlbumId: album.Id,
			}, jobWg, stage)
		}

		logger.Debug("Processed batch of artist albums",
			zap.String("artistId", artistId))
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting albums for artist %s: %w", artistId, err)
	}

	if err := albumBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining albums for artist %s: %w", artistId, err)
	}

	logger.Debug("Executed SaveArtistAlbumsJob",
		zap.String("artistId", artistId))
	return nil
}

func processTopArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Getting user's top artists",
		zap.String("userId", userId))

	artistBatcher := createArtistBatcher(userId, tracker, db.SaveUserTopArtists)

	err := spotify.GetUsersTopArtists(token, func(artists []*spotify.Artist) error {
		for _, artist := range artists {
			if err := artistBatcher.Add(artist); err != nil {
				return fmt.Errorf("adding artist to batch: %w", err)
			}
			pool.SubmitWithStage(&SaveArtistTopTracksJob{
				ArtistId: artist.Id,
				Type:     TopArtists,
			}, jobWg, stage)
			pool.SubmitWithStage(&SaveArtistAlbumsJob{
				ArtistId: artist.Id,
			}, jobWg, stage)
		}

		logger.Debug("Processed batch of top artists",
			zap.String("userId", userId))
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting top artists for user %s: %w", userId, err)
	}

	if err := artistBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining artists: %w", err)
	}

	logger.Debug("Processed user's top artists",
		zap.String("userId", userId))
	return nil
}

func processFollowedArtists(userId string, token string, pool *WorkerPool, tracker *ProcessedTracker, jobWg *sync.WaitGroup, stage *StageContext) error {
	logger.Debug("Getting user's followed artists",
		zap.String("userId", userId))

	artistBatcher := createArtistBatcher(userId, tracker, db.SaveUserTopArtists)

	err := spotify.GetUsersFollowedArtists(token, func(artists []*spotify.Artist) error {
		for _, artist := range artists {
			if err := artistBatcher.Add(artist); err != nil {
				return fmt.Errorf("adding artist to batch: %w", err)
			}
			pool.SubmitWithStage(&SaveArtistTopTracksJob{
				ArtistId: artist.Id,
				Type:     TopArtists,
			}, jobWg, stage)
			pool.SubmitWithStage(&SaveArtistAlbumsJob{
				ArtistId: artist.Id,
			}, jobWg, stage)
		}

		logger.Debug("Processed batch of followed artists",
			zap.String("userId", userId))
		return nil
	})
	if err != nil {
		return fmt.Errorf("getting followed artists for user %s: %w", userId, err)
	}

	if err := artistBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing remaining artists: %w", err)
	}

	logger.Debug("Processed user's followed artists",
		zap.String("userId", userId))
	return nil
}
