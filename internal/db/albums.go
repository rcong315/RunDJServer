package db

import (
	"fmt"

	"go.uber.org/zap"
)

type Album struct {
	AlbumId          string   `json:"album_id"`
	Name             string   `json:"name"`
	ArtistIds        []string `json:"artist_ids"`
	Genres           []string `json:"genres"`
	Popularity       int      `json:"popularity"`
	AlbumType        string   `json:"album_type"`
	TotalTracks      int      `json:"total_tracks"`
	ReleaseDate      string   `json:"release_date"`
	AvailableMarkets []string `json:"available_markets"`
	ImageURLs        []string `json:"image_urls"`
}

func SaveAlbums(albums []*Album) error {
	if len(albums) == 0 {
		logger.Debug("SaveAlbums: No albums to save.")
		return nil
	}
	logger.Info("Attempting to save albums", zap.Int("count", len(albums)))

	err := batchAndSave(albums, "album", func(item any, _ int) []any {
		album := item.(*Album)
		return []any{
			album.AlbumId,
			album.Name,
			album.ArtistIds,
			album.Genres,
			album.Popularity,
			album.AlbumType,
			album.TotalTracks,
			album.ReleaseDate,
			album.AvailableMarkets,
			album.ImageURLs,
		}
	})
	if err != nil {
		logger.Error("Error saving albums batch", zap.Int("count", len(albums)), zap.Error(err))
		return fmt.Errorf("error saving albums: %v", err)
	}

	logger.Info("Successfully saved albums batch", zap.Int("count", len(albums)))
	return nil
}

func SaveUserSavedAlbums(userId string, albums []*Album) error {
	if len(albums) == 0 {
		logger.Debug("SaveUserSavedAlbums: No saved albums to associate for user.", zap.String("userId", userId))
		return nil
	}
	logger.Info("Attempting to save user-saved album associations", zap.String("userId", userId), zap.Int("count", len(albums)))

	err := batchAndSave(albums, "userSavedAlbum", func(item any, _ int) []any {
		album := item.(*Album)
		return []any{
			userId,
			album.AlbumId,
		}
	})
	if err != nil {
		logger.Error("Error saving user-saved album associations batch",
			zap.String("userId", userId),
			zap.Int("count", len(albums)),
			zap.Error(err))
		return fmt.Errorf("error saving user saved albums: %v", err)
	}

	logger.Info("Successfully saved user-saved album associations batch", zap.String("userId", userId), zap.Int("count", len(albums)))
	return nil
}

func SaveAlbumTracks(albumId string, tracks []*Track) error {
	if len(tracks) == 0 {
		logger.Debug("SaveAlbumTracks: No tracks to associate with album.", zap.String("albumId", albumId))
		return nil
	}
	logger.Info("Attempting to save album-track associations", zap.String("albumId", albumId), zap.Int("trackCount", len(tracks)))

	err := batchAndSave(tracks, "albumTrack", func(item any, _ int) []any {
		track := item.(*Track)
		return []any{
			albumId,
			track.TrackId,
		}
	})
	if err != nil {
		logger.Error("Error saving album-track associations batch",
			zap.String("albumId", albumId),
			zap.Int("trackCount", len(tracks)),
			zap.Error(err))
		return fmt.Errorf("error saving album tracks: %v", err)
	}

	logger.Info("Successfully saved album-track associations batch", zap.String("albumId", albumId), zap.Int("trackCount", len(tracks)))
	return nil
}
