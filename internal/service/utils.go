package service

import (
	"github.com/rcong315/RunDJServer/internal/db"
	"github.com/rcong315/RunDJServer/internal/spotify"
)

var presetPlaylists = map[int]string{
	105: "56cgN0YoqzPjmNBBuiVo6b",
	110: "2pX7htNxQUGZSObonznRyn",
	115: "78qmqXAefQPCbQ5JqfwWgz",
	120: "2rzL3ZFSz87245ljAic93z",
	125: "37i9dQZF1EIgsxtEuT3KWN",
	130: "37i9dQZF1EIdJGESPytB8N",
	135: "37i9dQZF1EIdnGKfcfozNo",
	140: "37i9dQZF1EIgOKtiospcqN",
	145: "37i9dQZF1EIcB36Vij2P5d",
	150: "37i9dQZF1EIgrZKdA44WQK",
	155: "37i9dQZF1EIeGfmJObJDc0",
	160: "37i9dQZF1EIdYV92VKrjuC",
	165: "37i9dQZF1EIcNylL4dr08W",
	170: "37i9dQZF1EIgfIackHptHl",
	175: "37i9dQZF1EIfnhoQIQxMqH",
	180: "37i9dQZF1EIgUYhklBpeMG",
	185: "37i9dQZF1EIhy9qfhxNEnX",
	190: "37i9dQZF1EIcID9rq1OAoH",
}

func convertSpotifyUserToDBUser(user *spotify.User) *db.User {
	imageURLs := make([]string, len(user.ImageURLs))
	for i, img := range user.ImageURLs {
		imageURLs[i] = img.URL
	}

	return &db.User{
		UserId:      user.Id,
		Email:       user.Email,
		DisplayName: user.DisplayName,
		Country:     user.Country,
		Followers:   user.Followers.Total,
		Product:     user.Product,
		ImageURLs:   imageURLs,
	}
}

// ConvertSpotifyTracksToDBTracks converts Spotify tracks to database tracks
func ConvertSpotifyTracksToDBTracks(tracks []*spotify.Track) []*db.Track {
	var dbTracks []*db.Track
	for _, track := range tracks {
		if track == nil || track.Id == "" {
			continue
		}

		artistIds := make([]string, len(track.Artists))
		for i, artist := range track.Artists {
			artistIds[i] = artist.Id
		}

		var albumId string
		if track.Album == nil {
			albumId = ""
		} else {
			albumId = track.Album.Id
		}

		audioFeatures := track.AudioFeatures
		var dbAudioFeatures *db.AudioFeatures
		if track.AudioFeatures == nil {
			dbAudioFeatures = &db.AudioFeatures{}
		} else {
			dbAudioFeatures = &db.AudioFeatures{
				Danceability:      audioFeatures.Danceability,
				Energy:            audioFeatures.Energy,
				Key:               audioFeatures.Key,
				Loudness:          audioFeatures.Loudness,
				Mode:              audioFeatures.Mode,
				Speechiness:       audioFeatures.Speechiness,
				Acousticness:      audioFeatures.Acousticness,
				Instrumentallness: audioFeatures.Instrumentallness,
				Liveness:          audioFeatures.Liveness,
				Valence:           audioFeatures.Valence,
				Tempo:             audioFeatures.Tempo,
				Duration:          audioFeatures.Duration,
				TimeSignature:     audioFeatures.TimeSignature,
			}
		}

		dbTrack := &db.Track{
			TrackId:          track.Id,
			Name:             track.Name,
			ArtistIds:        artistIds,
			AlbumId:          albumId,
			Popularity:       track.Popularity,
			DurationMS:       track.DurationMS,
			AvailableMarkets: track.AvailableMarkets,
			AudioFeatures:    dbAudioFeatures,
			TimeSignature:    dbAudioFeatures.TimeSignature,
		}
		dbTracks = append(dbTracks, dbTrack)
	}

	return dbTracks
}

func convertSpotifyPlaylistsToDBPlaylists(playlists []*spotify.Playlist) []*db.Playlist {
	var dbPlaylists []*db.Playlist
	for _, playlist := range playlists {
		if playlist == nil || playlist.Id == "" {
			continue
		}

		imageURLs := make([]string, len(playlist.Images))
		for i, img := range playlist.Images {
			imageURLs[i] = img.URL
		}

		dbPlaylist := &db.Playlist{
			PlaylistId:  playlist.Id,
			Name:        playlist.Name,
			Description: playlist.Description,
			OwnerId:     playlist.Owner.Id,
			Public:      playlist.Public,
			Followers:   playlist.Followers.Total,
			ImageURLs:   imageURLs,
		}
		dbPlaylists = append(dbPlaylists, dbPlaylist)
	}

	return dbPlaylists
}

func convertSpotifyArtistsToDBArtists(artists []*spotify.Artist) []*db.Artist {
	var dbArtists []*db.Artist
	for _, artist := range artists {
		if artist == nil || artist.Id == "" {
			continue
		}

		imageURLs := make([]string, len(artist.Images))
		for i, img := range artist.Images {
			imageURLs[i] = img.URL
		}

		dbArtist := &db.Artist{
			ArtistId:   artist.Id,
			Name:       artist.Name,
			Genres:     artist.Genres,
			Popularity: artist.Popularity,
			Followers:  artist.Followers.Total,
			ImageURLs:  imageURLs,
		}
		dbArtists = append(dbArtists, dbArtist)
	}

	return dbArtists
}

func convertSpotifyAlbumsToDBAlbums(albums []*spotify.Album) []*db.Album {
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
