INSERT INTO "playlist_track" (playlist_id, track_id, feedback)
VALUES ($1, $2, $3) ON CONFLICT (playlist_id, track_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    updated_at = NOW();