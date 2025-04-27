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
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10) ON CONFLICT (album_id) DO
UPDATE
SET name = EXCLUDED.name,
    artist_ids = EXCLUDED.artist_ids,
    genres = EXCLUDED.genres,
    popularity = EXCLUDED.popularity,
    album_type = EXCLUDED.album_type,
    total_tracks = EXCLUDED.total_tracks,
    release_date = EXCLUDED.release_date,
    available_markets = EXCLUDED.available_markets,
    image_urls = EXCLUDED.image_urls;