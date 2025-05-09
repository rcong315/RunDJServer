INSERT INTO "user_followed_artist" (user_id, artist_id, feedback)
VALUES ($1, $2, $3) ON CONFLICT (user_id, artist_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    updated_at = NOW();