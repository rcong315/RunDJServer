package db

// SQL queries for database operations
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
			name = $2,
			artist_ids = $3,
			album_id = $4,
			popularity = $5,
			duration_ms = $6,
			available_markets = $7,
			audio_features = $8
	`

	UpdateAudioFeaturesQuery = `
		UPDATE "track"
		SET audio_features = $2
		WHERE track_id = $1
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

	InsertPlaylistQuery = `
		INSERT INTO "playlist" (
			playlist_id,
			owner_id,
			name,
			description,
			public,
			image_urls
		)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (playlist_id) DO UPDATE
		SET
			owner_id = $2,
			name = $3,
			description = $4,
			public = $5,
			image_urls = $6
	`

	InsertUserTrackRelationQuery = `
		INSERT INTO "user_track_relation" (
			user_id,
			track_id,
			sources
		)
		VALUES ($1, $2, $3)
		ON CONFLICT (user_id, track_id) DO UPDATE
		SET sources = array_cat(user_track_relation.sources, $3)
	`
)
