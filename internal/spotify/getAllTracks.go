package spotify

import "log"

func GetAllTracks(token string) ([]Track, []Album, []Artist, []Playlist) {
	var idsSet = make(map[string]struct{})
	var trackCount = 0

	addTrack := func(id string) {
		if _, exists := idsSet[id]; !exists {
			idsSet[id] = struct{}{}
			trackCount++
			if trackCount%500 == 0 {
				log.Printf("Added %d tracks, latest: %s", trackCount, id)
			}
		}
	}

	// User's top tracks
	log.Print("Getting user's top tracks")
	for _, id := range getUsersTopTracks(token) {
		addTrack(id)
	}

	// User's saved tracks
	log.Printf("Getting user's saved tracks")
	for _, id := range getUsersSavedTracks(token) {
		addTrack(id)
	}

	// User's playlists
	log.Printf("Getting tracks from user's playlists")
	var playlistIds = getUsersPlaylists(token)
	for _, playlistId := range playlistIds {
		for _, id := range getPlaylistsTracks(token, playlistId) {
			addTrack(id)
		}
	}

	// User's top artists and followed artists
	log.Printf("Getting tracks from user's top and followed artists")
	var topArtistsIds = getUsersTopArtists(token)
	var followedArtistIds = getUsersFollowedArtists(token)
	artistsMap := make(map[string]struct{})
	for _, id := range topArtistsIds {
		artistsMap[id] = struct{}{}
	}
	for _, id := range followedArtistIds {
		artistsMap[id] = struct{}{}
	}
	var albumIds []string
	for _, artistId := range artistsMap {
		for _, traickId := range getArtistsTopTracks(token, artistId) {
			addTrack(traickId)
		}
		albumIds = append(albumIds, getArtistsAlbums(token, artistId)...)
	}
	for _, id := range albumIds {
		for _, id := range getAlbumsTracks(token, id) {
			idsSet[id] = struct{}{}
			log.Printf("Added track Id from album: %s", id)
		}
	}

	var ids []string
	for id := range idsSet {
		ids = append(ids, id)
	}

	return ids
}
