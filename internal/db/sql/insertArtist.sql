INSERT INTO "artist" (
        artist_id,
        name,
        genres,
        popularity,
        followers,
        image_urls
    )
VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT (artist_id) DO
UPDATE
SET name = EXCLUDED.name,
    genres = EXCLUDED.genres,
    popularity = EXCLUDED.popularity,
    followers = EXCLUDED.followers,
    image_urls = EXCLUDED.image_urls,
    updated_at = NOW();