INSERT INTO "user_top_artist" (
        user_id,
        artist_id,
        feedback,
        rank
    )
VALUES ($1, $2, $3, $4) ON CONFLICT (user_id, artist_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    rank = EXCLUDED.rank,
    updated_at = NOW();