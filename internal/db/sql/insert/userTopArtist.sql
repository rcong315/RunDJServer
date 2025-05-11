INSERT INTO "user_top_artist" (
        user_id,
        artist_id,
        rank
    )
VALUES ($1, $2, $3) ON CONFLICT (user_id, artist_id) DO
UPDATE
SET rank = EXCLUDED.rank,
    updated_at = NOW();