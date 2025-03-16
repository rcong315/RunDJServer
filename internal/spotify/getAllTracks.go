package spotify

import (
	"log"
)

func GetAllTracks(token string) ([]Track, []Album, []Artist, []Playlist) {
	// *** TODO: Save as you go ***

	var trackSet = make(map[string]Track)
	var trackCount = 0

	addTrack := func(track Track) {
		if _, exists := trackSet[track.Id]; !exists {
			trackSet[track.Id] = track
			trackCount++
			if trackCount%500 == 0 {
				log.Printf("Added %d tracks, latest: %s", trackCount, track.Id)
			}
		}
	}

	// User's top tracks
	log.Print("Getting user's top tracks")
	usersTopTracks, err := GetUsersTopTracks(token)
	if err != nil {
		log.Printf("Error getting user's top tracks: %v", err)
	}
	for _, track := range usersTopTracks {
		if err != nil {
			log.Printf("Error marshaling track: %v", err)
			continue
		}
		addTrack(track)
	}

	// // User's saved tracks
	// log.Printf("Getting user's saved tracks")
	// for _, id := range getUsersSavedTracks(token) {
	// 	addTrack(id)
	// }

	// // User's playlists
	// log.Printf("Getting tracks from user's playlists")
	// var playlistIds = getUsersPlaylists(token)
	// for _, playlistId := range playlistIds {
	// 	for _, id := range getPlaylistsTracks(token, playlistId) {
	// 		addTrack(id)
	// 	}
	// }

	// // User's top artists and followed artists
	// log.Printf("Getting tracks from user's top and followed artists")
	// var topArtistsIds = getUsersTopArtists(token)
	// var followedArtistIds = getUsersFollowedArtists(token)
	// artistsMap := make(map[string]struct{})
	// for _, id := range topArtistsIds {
	// 	artistsMap[id] = struct{}{}
	// }
	// for _, id := range followedArtistIds {
	// 	artistsMap[id] = struct{}{}
	// }
	// var albumIds []string
	// for _, artistId := range artistsMap {
	// 	for _, traickId := range getArtistsTopTracks(token, artistId) {
	// 		addTrack(traickId)
	// 	}
	// 	albumIds = append(albumIds, getArtistsAlbums(token, artistId)...)
	// }
	// for _, id := range albumIds {
	// 	for _, id := range getAlbumsTracks(token, id) {
	// 		idsSet[id] = struct{}{}
	// 		log.Printf("Added track Id from album: %s", id)
	// 	}
	// }

	// var ids []string
	// for id := range idsSet {
	// 	ids = append(ids, id)
	// }

	// return ids
	return MapTracksToArray(trackSet), MapAlbumsToArray(nil), MapArtistsToArray(nil), MapPlaylistsToArray(nil)
}
