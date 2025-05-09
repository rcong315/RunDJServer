INSERT INTO "user_top_track" (
        user_id,
        track_id,
        feedback,
        rank
    )
VALUES ($1, $2, $3, $3) ON CONFLICT (user_id, track_id) DO
UPDATE
SET feedback = EXCLUDED.feedback,
    rank = EXCLUDED.rank,
    updated_at = NOW();