package crawler

import (
	"context"
	"fmt"
	"sync"

	"go.uber.org/zap"

	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/service"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

type DiscoverMissingEntitiesJob struct {
	logger *zap.Logger
}

func (j *DiscoverMissingEntitiesJob) Execute(pool *service.WorkerPool, jobWg *sync.WaitGroup, tracker *service.ProcessedTracker, stage *service.StageContext) error {
	defer jobWg.Done()
	defer stage.Wg.Done()

	j.logger.Info("Starting DiscoverMissingEntitiesJob")

	// Find tracks with missing artists
	missingArtists, err := j.getTracksWithMissingArtists()
	if err != nil {
		return fmt.Errorf("getting tracks with missing artists: %w", err)
	}

	// Find tracks with missing albums
	missingAlbums, err := j.getTracksWithMissingAlbums()
	if err != nil {
		return fmt.Errorf("getting tracks with missing albums: %w", err)
	}

	j.logger.Info("Found missing entities",
		zap.Int("missingArtists", len(missingArtists)),
		zap.Int("missingAlbums", len(missingAlbums)))

	// Queue jobs to process missing artists
	for _, artistID := range missingArtists {
		if !tracker.CheckAndMark("artist", artistID) {
			job := &service.SaveArtistTopTracksJob{
				ArtistId: artistID,
				Type:     service.TopArtists, // Default type
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(job, jobWg, stage)

			// Also queue album processing for the artist
			albumJob := &service.SaveArtistAlbumsJob{
				ArtistId: artistID,
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(albumJob, jobWg, stage)
		}
	}

	// Queue jobs to process missing albums
	for _, albumID := range missingAlbums {
		if !tracker.CheckAndMark("album", albumID) {
			job := &ProcessAlbumJob{
				logger:  j.logger,
				albumID: albumID,
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(job, jobWg, stage)
		}
	}

	j.logger.Info("Completed DiscoverMissingEntitiesJob",
		zap.Int("queuedArtists", len(missingArtists)),
		zap.Int("queuedAlbums", len(missingAlbums)))
	return nil
}

func (j *DiscoverMissingEntitiesJob) getTracksWithMissingArtists() ([]string, error) {
	query := `
		SELECT DISTINCT unnest(t.artist_ids) AS artist_id
		FROM track t
		WHERE NOT EXISTS (
			SELECT 1 FROM artist a
			WHERE a.artist_id = ANY(t.artist_ids)
		)
		AND array_length(t.artist_ids, 1) > 0
		LIMIT 500
	`

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("getting database connection: %w", err)
	}

	rows, err := database.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("querying missing artists: %w", err)
	}
	defer rows.Close()

	var artistIDs []string
	for rows.Next() {
		var artistID string
		if err := rows.Scan(&artistID); err != nil {
			j.logger.Error("Failed to scan artist ID", zap.Error(err))
			continue
		}
		artistIDs = append(artistIDs, artistID)
	}

	return artistIDs, nil
}

func (j *DiscoverMissingEntitiesJob) getTracksWithMissingAlbums() ([]string, error) {
	query := `
		SELECT DISTINCT t.album_id
		FROM track t
		WHERE t.album_id IS NOT NULL
		AND t.album_id != ''
		AND NOT EXISTS (
			SELECT 1 FROM album a
			WHERE a.album_id = t.album_id
		)
		LIMIT 500
	`

	database, err := db.GetDB()
	if err != nil {
		return nil, fmt.Errorf("getting database connection: %w", err)
	}

	rows, err := database.Query(context.Background(), query)
	if err != nil {
		return nil, fmt.Errorf("querying missing albums: %w", err)
	}
	defer rows.Close()

	var albumIDs []string
	for rows.Next() {
		var albumID string
		if err := rows.Scan(&albumID); err != nil {
			j.logger.Error("Failed to scan album ID", zap.Error(err))
			continue
		}
		albumIDs = append(albumIDs, albumID)
	}

	return albumIDs, nil
}

// ProcessAlbumJob processes a single album
type ProcessAlbumJob struct {
	logger  *zap.Logger
	albumID string
}

func (j *ProcessAlbumJob) Execute(pool *service.WorkerPool, jobWg *sync.WaitGroup, tracker *service.ProcessedTracker, stage *service.StageContext) error {
	defer jobWg.Done()
	defer stage.Wg.Done()

	j.logger.Debug("Processing album", zap.String("albumID", j.albumID))

	// Get secret token
	token, err := spotify.GetSecretToken()
	if err != nil {
		return fmt.Errorf("getting secret token: %w", err)
	}

	// Fetch album details
	album, err := spotify.GetAlbum(token, j.albumID)
	if err != nil {
		return fmt.Errorf("fetching album details: %w", err)
	}

	// Convert and save album
	dbAlbums := j.convertSpotifyAlbumsToDBAlbums([]*spotify.Album{album})
	if len(dbAlbums) > 0 {
		if err := db.SaveAlbums(dbAlbums); err != nil {
			return fmt.Errorf("saving album: %w", err)
		}
		j.logger.Debug("Saved album to database", 
			zap.String("albumID", j.albumID),
			zap.String("albumName", album.Name))
	}

	// Process album artists that might be missing
	for _, artist := range album.Artists {
		if artist.Id != "" && !tracker.CheckAndMark("artist", artist.Id) {
			// Queue artist processing jobs
			artistJob := &service.SaveArtistTopTracksJob{
				ArtistId: artist.Id,
				Type:     service.TopArtists,
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(artistJob, jobWg, stage)

			// Also queue artist albums
			albumsJob := &service.SaveArtistAlbumsJob{
				ArtistId: artist.Id,
			}
			jobWg.Add(1)
			stage.Wg.Add(1)
			pool.SubmitWithStage(albumsJob, jobWg, stage)
		}
	}

	// Create track batcher for processing album tracks
	trackBatcher := spotify.NewBatchProcessor(100, func(tracks []*spotify.Track) error {
		// Enrich tracks with audio features
		enrichedTracks, err := spotify.GetAudioFeatures(token, tracks)
		if err != nil {
			j.logger.Error("Failed to get audio features for album tracks", 
				zap.String("albumID", j.albumID),
				zap.Error(err))
			// Continue with tracks without audio features
			enrichedTracks = tracks
		}

		// Convert and save tracks
		dbTracks := service.ConvertSpotifyTracksToDBTracks(enrichedTracks)
		if err := db.SaveTracks(dbTracks); err != nil {
			return fmt.Errorf("saving tracks: %w", err)
		}

		// Save album-track relationships
		if err := db.SaveAlbumTracks(j.albumID, dbTracks); err != nil {
			return fmt.Errorf("saving album-track relationships: %w", err)
		}

		j.logger.Debug("Saved album tracks batch", 
			zap.String("albumID", j.albumID),
			zap.Int("trackCount", len(dbTracks)))
		return nil
	})

	// Fetch and process album tracks
	err = spotify.GetAlbumsTracks(j.albumID, func(tracks []*spotify.Track) error {
		// Set album ID for tracks (might not be set in simplified track objects)
		for _, track := range tracks {
			if track.Album == nil {
				track.Album = album
			}
		}

		// Add tracks to batcher
		for _, track := range tracks {
			if err := trackBatcher.Add(track); err != nil {
				j.logger.Error("Failed to add track to batch", 
					zap.String("trackID", track.Id),
					zap.Error(err))
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("fetching album tracks: %w", err)
	}

	// Flush remaining tracks
	if err := trackBatcher.Flush(); err != nil {
		return fmt.Errorf("flushing track batcher: %w", err)
	}

	j.logger.Info("Successfully processed album", 
		zap.String("albumID", j.albumID),
		zap.String("albumName", album.Name),
		zap.Int("totalTracks", album.TotalTracks))
	return nil
}

// convertSpotifyAlbumsToDBAlbums converts Spotify albums to database format
func (j *ProcessAlbumJob) convertSpotifyAlbumsToDBAlbums(albums []*spotify.Album) []*db.Album {
	var dbAlbums []*db.Album
	for _, album := range albums {
		if album == nil || album.Id == "" {
			continue
		}

		artistIds := make([]string, len(album.Artists))
		for i, artist := range album.Artists {
			artistIds[i] = artist.Id
		}

		imageURLs := make([]string, len(album.Images))
		for i, img := range album.Images {
			imageURLs[i] = img.URL
		}

		dbAlbum := &db.Album{
			AlbumId:          album.Id,
			Name:             album.Name,
			ArtistIds:        artistIds,
			Genres:           album.Genres,
			Popularity:       album.Popularity,
			AlbumType:        album.AlbumType,
			TotalTracks:      album.TotalTracks,
			ReleaseDate:      album.ReleaseDate,
			AvailableMarkets: album.AvailableMarkets,
			ImageURLs:        imageURLs,
		}
		dbAlbums = append(dbAlbums, dbAlbum)
	}
	return dbAlbums
}