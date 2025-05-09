INSERT INTO "user_playlist" (user_id, playlist_id, feedback)
VALUES ($1, $2, $3) ON CONFLICT (user_id, playlist_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    updated_at = NOW();