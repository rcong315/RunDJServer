INSERT INTO "user_saved_track" (user_id, track_id, feedback)
VALUES ($1, $2, $3) ON CONFLICT (user_id, track_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    updated_at = NOW();