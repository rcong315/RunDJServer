package db

import (
	"fmt"

	"go.uber.org/zap"
)

type Artist struct {
	ArtistId   string   `json:"artist_id"`
	Name       string   `json:"name"`
	Genres     []string `json:"genres"`
	Popularity int      `json:"popularity"`
	Followers  int      `json:"followers"`
	ImageURLs  []string `json:"image_urls"`
}

func SaveArtists(artists []*Artist) error {
	if len(artists) == 0 {
		logger.Debug("SaveArtists: No artists to save.")
		return nil
	}
	logger.Debug("Attempting to save artists", zap.Int("count", len(artists)))

	err := batchAndSave(artists, "artist", func(item any, _ int) []any {
		artist := item.(*Artist)
		return []any{
			artist.ArtistId,
			artist.Name,
			artist.Genres,
			artist.Popularity,
			artist.Followers,
			artist.ImageURLs,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving artists: %v", err)
	}

	logger.Debug("Successfully saved artists batch", zap.Int("count", len(artists)))
	return nil
}

func SaveUserTopArtists(userId string, artists []*Artist) error {
	if len(artists) == 0 {
		logger.Debug("SaveUserTopArtists: No top artists to associate for user.", zap.String("userId", userId))
		return nil
	}
	logger.Debug("Attempting to save user-top artist associations", zap.String("userId", userId), zap.Int("count", len(artists)))

	err := batchAndSave(artists, "userTopArtist", func(item any, rank int) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
			rank,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user top artists: %v", err)
	}

	logger.Debug("Successfully saved user-top artist associations batch", zap.String("userId", userId), zap.Int("count", len(artists)))
	return nil
}

func SaveUserFollowedArtists(userId string, artists []*Artist) error {
	if len(artists) == 0 {
		logger.Debug("SaveUserFollowedArtists: No followed artists to associate for user.", zap.String("userId", userId))
		return nil
	}
	logger.Debug("Attempting to save user-followed artist associations", zap.String("userId", userId), zap.Int("count", len(artists)))

	err := batchAndSave(artists, "userFollowedArtist", func(item any, _ int) []any {
		artist := item.(*Artist)
		return []any{
			userId,
			artist.ArtistId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving user followed artists: %v", err)
	}

	logger.Debug("Successfully saved user-followed artist associations batch", zap.String("userId", userId), zap.Int("count", len(artists)))
	return nil
}

func SaveArtistTopTracks(artistId string, tracks []*Track) error {
	if len(tracks) == 0 {
		logger.Debug("SaveArtistTopTracks: No top tracks to associate with artist.", zap.String("artistId", artistId))
		return nil
	}
	logger.Debug("Attempting to save artist-top track associations", zap.String("artistId", artistId), zap.Int("trackCount", len(tracks)))

	err := batchAndSave(tracks, "artistTopTrack", func(item any, rank int) []any {
		track := item.(*Track)
		return []any{
			artistId,
			track.TrackId,
			rank,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving artist top tracks: %v", err)
	}

	logger.Debug("Successfully saved artist-top track associations batch", zap.String("artistId", artistId), zap.Int("trackCount", len(tracks)))
	return nil
}

func SaveArtistAlbums(artistId string, albums []*Album) error {
	if len(albums) == 0 {
		logger.Debug("SaveArtistAlbums: No albums to associate with artist.", zap.String("artistId", artistId))
		return nil
	}
	logger.Debug("Attempting to save artist-album associations", zap.String("artistId", artistId), zap.Int("albumCount", len(albums)))

	err := batchAndSave(albums, "artistAlbum", func(item any, _ int) []any {
		album := item.(*Album)
		return []any{
			artistId,
			album.AlbumId,
		}
	})
	if err != nil {
		return fmt.Errorf("error saving artist albums: %v", err)
	}

	logger.Debug("Successfully saved artist-album associations batch", zap.String("artistId", artistId), zap.Int("albumCount", len(albums)))
	return nil
}
