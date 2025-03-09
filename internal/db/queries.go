package db

// SQL queries for database operations
const (
	// User related queries
	InsertUserQuery = `
		INSERT INTO "user" (
			user_id, 
			email, 
			display_name, 
			country, 
			followers, 
			product, 
			explicit_filter_enabled, 
			image_urls
		) 
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (user_id) DO NOTHING
	`

	// Track related queries
	InsertTrackQuery = `
		INSERT INTO "track" (
			track_id, 
			user_ids, 
			name, 
			artist_ids,
			album_id,
			popularity,
			available_markets,
			audio_features
		) 
		VALUES (
			$1, ARRAY[$2], $3, $4, $5, $6, $7, $8
		)
		ON CONFLICT (track_id) DO UPDATE 
		SET 
			user_ids = track.user_ids || ARRAY[$2],
			name = $3,
			artist_ids = $4,
			album_id = $5,
			popularity = $6,
			available_markets = $7,
			audio_features = $8
		WHERE NOT (track.user_ids @> ARRAY[$2])
	`
	UpdateAudioFeaturesQuery = `
		UPDATE "track"
		SET audio_features = $2
		WHERE track_id = $1
	`

	// Album related queries
	InsertAlbumQuery = `
		INSERT INTO "album" (
			album_id,
			user_ids,
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
		VALUES (
			$1, ARRAY[$2], $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
		ON CONFLICT (album_id) DO UPDATE
		SET
			user_id = album.user_id || ARRAY[$2],
			title = $3,
			artist_ids = $4,
			genres = $5,
			popularity = $6,
			album_type = $7,
			total_tracks = $8,
			release_date = $9,
			available_markets = $10,
			image_urls = $11
		WHERE NOT (album.user_id @> ARRAY[$2])
	`

	// Artist related queries
	InsertArtistQuery = `
		INSERT INTO "artist" (
			artist_id,
			user_ids,
			name,
			genres,
			popularity,
			followers,
			image_urls
		) VALUES (
			$1, ARRAY[$2], $3, $4, $5, $6, $7
		)
		ON CONFLICT (artist_id) DO UPDATE
		SET
			user_ids = artist.user_ids || ARRAY[$2],
			name = $3,
			genres = $4,
			popularity = $5,
			followers = $6,
			image_urls = $7
		WHERE NOT (artist.user_ids @> ARRAY[$2])
	`

	// Playlist related queries
	InsertPlaylistQuery = `
		INSERT INTO "playlist" (
			playlist_id,
			owner_id,
			name,
			description
		) VALUES (
			$1, $2, $3, $4
		)
		ON CONFLICT (playlist_id) DO NOTHING
	`
)
