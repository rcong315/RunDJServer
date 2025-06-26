package crawler

import (
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/service"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type ProcessPlaylistJob struct {
	logger     *zap.Logger
	playlistID string
	deep       bool // If true, process all artists from the playlist
}

func (j *ProcessPlaylistJob) Execute(pool *service.WorkerPool, jobWg *sync.WaitGroup, tracker *service.ProcessedTracker, stage *service.StageContext) error {
	defer jobWg.Done()
	defer stage.Wg.Done()

	j.logger.Info("Starting ProcessPlaylistJob", 
		zap.String("playlistID", j.playlistID),
		zap.Bool("deep", j.deep))

	// Get secret token
	token, err := spotify.GetSecretToken()
	if err != nil {
		return fmt.Errorf("getting secret token: %w", err)
	}

	// Create batch processor for tracks
	trackBatcher := spotify.NewBatchProcessor(100, func(tracks []*spotify.Track) error {
		// Get audio features
		enrichedTracks, err := spotify.GetAudioFeatures(token, tracks)
		if err != nil {
			j.logger.Error("Failed to get audio features", zap.Error(err))
			// Continue processing without audio features
			enrichedTracks = tracks
		}

		// Convert and save tracks
		dbTracks := service.ConvertSpotifyTracksToDBTracks(enrichedTracks)
		if err := db.SaveTracks(dbTracks); err != nil {
			return fmt.Errorf("saving tracks: %w", err)
		}

		// Save playlist-track relationships
		if err := j.savePlaylistTracks(dbTracks); err != nil {
			return fmt.Errorf("saving playlist tracks: %w", err)
		}

		// If deep processing, queue artist jobs
		if j.deep {
			j.queueArtistJobs(enrichedTracks, pool, jobWg, tracker, stage)
		}

		return nil
	})

	// Fetch playlist tracks
	err = spotify.GetPlaylistTracks(token, j.playlistID, func(tracks []*spotify.Track) error {
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				j.logger.Error("Failed to add track to batch", zap.Error(err))
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching playlist tracks: %w", err)
	}

	// Flush remaining tracks
	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing track batcher: %w", err)
	}

	j.logger.Info("Completed ProcessPlaylistJob", zap.String("playlistID", j.playlistID))
	return nil
}

func (j *ProcessPlaylistJob) savePlaylistTracks(tracks []*db.Track) error {
	// Save playlist if not exists
	playlist := &db.Playlist{
		PlaylistId: j.playlistID,
		Name:       "Weekly Crawled Playlist", // Default name
		OwnerId:    "crawler",
	}
	if err := db.SavePlaylists([]*db.Playlist{playlist}); err != nil {
		j.logger.Warn("Failed to save playlist", zap.Error(err))
		// Continue anyway, playlist might already exist
	}

	// Save playlist-track relationships
	return db.SavePlaylistTracks(j.playlistID, tracks)
}

func (j *ProcessPlaylistJob) queueArtistJobs(tracks []*spotify.Track, pool *service.WorkerPool, 
	jobWg *sync.WaitGroup, tracker *service.ProcessedTracker, stage *service.StageContext) {
	
	// Collect unique artist IDs
	artistIDs := make(map[string]bool)
	for _, track := range tracks {
		for _, artist := range track.Artists {
			if artist.Id != "" {
				artistIDs[artist.Id] = true
			}
		}
	}

	// Queue jobs for each artist
	for artistID := range artistIDs {
		if !tracker.CheckAndMark("artist_deep", artistID) {
			// Queue top tracks job
			topTracksJob := &service.SaveArtistTopTracksJob{
				ArtistId: artistID,
				Type:     service.TopArtists,
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(topTracksJob, jobWg, stage)

			// Queue albums job
			albumsJob := &service.SaveArtistAlbumsJob{
				ArtistId: artistID,
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(albumsJob, jobWg, stage)
		}
	}

	j.logger.Debug("Queued artist jobs", zap.Int("artistCount", len(artistIDs)))
}