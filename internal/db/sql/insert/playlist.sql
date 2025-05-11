INSERT INTO "playlist" (
        playlist_id,
        name,
        description,
        owner_id,
        public,
        followers,
        image_urls
    )
VALUES ($1, $2, $3, $4, $5, $6, $7) ON CONFLICT (playlist_id) DO
UPDATE
SET name = EXCLUDED.name,
    description = EXCLUDED.description,
    owner_id = EXCLUDED.owner_id,
    public = EXCLUDED.public,
    followers = EXCLUDED.followers,
    image_urls = EXCLUDED.image_urls,
    updated_at = NOW();