INSERT INTO "user_saved_album" (user_id, album_id, feedback)
VALUES ($1, $2, $3) ON CONFLICT (user_id, album_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    updated_at = NOW();