package spotify

func getAllTracks(token string) []string {
	var idsSet = make(map[string]struct{})

	// User's playlists
	var playlistIDs = getUsersPlaylists(token)
	for _, id := range playlistIDs {
		idsSet[id] = struct{}{}
	}

	// User's top artists
	var artistIDs = getUsersTopArtists(token)
	var albumIDs []string
	for _, id := range artistIDs {
		for _, id := range getArtistsTopTracks(token, id) {
			idsSet[id] = struct{}{}
		}
		albumIDs = append(albumIDs, getArtistsAlbums(token, id)...)
	}
	for _, id := range albumIDs {
		idsSet[id] = struct{}{}
	}

	// User's top tracks
	for _, id := range getUsersTopTracks(token) {
		idsSet[id] = struct{}{}
	}

	var ids []string
	for id := range idsSet {
		ids = append(ids, id)
	}

	return ids
}
