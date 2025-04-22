package db

// TODO: Migrate to sql files
const (
	InsertUserQuery = `
		INSERT INTO "user" (
			user_id, 
			email, 
			display_name, 
			country, 
			followers, 
			product, 
			image_urls
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (user_id) DO NOTHING
	`

	InsertTrackQuery = `
		INSERT INTO "track" (
			track_id, 
			name, 
			artist_ids,
			album_id,
			popularity,
			duration_ms,
			available_markets,
			audio_features
		) 
		VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (track_id) DO UPDATE 
		SET 
			name = EXCLUDED.name,
			artist_ids = EXCLUDED.artist_ids,
			album_id = EXCLUDED.album_id,
			popularity = EXCLUDED.popularity,
			duration_ms = EXCLUDED.duration_ms,
			available_markets = EXCLUDED.available_markets,
			audio_features = EXCLUDED.audio_features
	`

	InsertPlaylistQuery = `
		INSERT INTO "playlist" (
			playlist_id,
			name,
			description,
			owner_id,
			public,
			followers,
			image_urls
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (playlist_id) DO UPDATE
		SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			owner_id = EXCLUDED.owner_id,
			public = EXCLUDED.public,
			followers = EXCLUDED.followers,
			image_urls = EXCLUDED.image_urls
	`

	InsertArtistQuery = `
		INSERT INTO "artist" (
			artist_id,
			name,
			genres,
			popularity,
			followers,
			image_urls
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (artist_id) DO UPDATE
		SET
			name = $2,
			genres = $3,
			popularity = $4,
			followers = $5,
			image_urls = $6
	`

	InsertAlbumQuery = `
		INSERT INTO "album" (
			album_id,
			name,
			artist_ids,
			genres,
			popularity,
			album_type,
			total_tracks,
			release_date,
			available_markets,
			image_urls
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT (album_id) DO UPDATE
		SET
			name = $2,
			artist_ids = $3,
			genres = $4,
			popularity = $5,
			album_type = $6,
			total_tracks = $7,
			release_date = $8,
			available_markets = $9,
			image_urls = $10
	`

	InsertUserTrackRelationQuery = `
		INSERT INTO "user_track_relation" (user_id, track_id, sources)
		VALUES ($1, $2, $3) ON CONFLICT (user_id, track_id) DO
		UPDATE
		SET sources = array_cat(user_track_relation.sources, EXCLUDED.sources)
	`

	InsertUserPlaylistRelationQuery = `
		INSERT INTO "user_playlist_relation" (user_id, playlist_id, sources)
		VALUES ($1, $2, $3) ON CONFLICT (user_id, playlist_id) DO
		UPDATE
		SET sources = array_cat(user_playlist_relation.sources, EXCLUDED.sources)
	`

	InsertUserArtistRelationQuery = `
		INSERT INTO "user_artist_relation" (user_id, artist_id, sources)
		VALUES ($1, $2, $3) ON CONFLICT (user_id, artist_id) DO
		UPDATE
		SET sources = array_cat(user_artist_relation.sources, EXCLUDED.sources)
	`

	InsertUserAlbumRelationQuery = `
		INSERT INTO "user_album_relation" (user_id, album_id, sources)
		VALUES ($1, $2, $3) ON CONFLICT (user_id, album_id) DO
		UPDATE
		SET sources = array_cat(user_album_relation.sources, EXCLUDED.sources)
	`

	InsertTrackPlaylistRelationQuery = `
		INSERT INTO "track_playlist_relation" (track_id, playlist_id)
		VALUES ($1, $2) ON CONFLICT (track_id, playlist_id) DO NOTHING
	`
)
