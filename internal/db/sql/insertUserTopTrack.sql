INSERT INTO "user_top_track" (
        user_id,
        track_id,
        rank
    )
VALUES ($1, $2, $3) ON CONFLICT (user_id, track_id) DO
UPDATE
SET rank = EXCLUDED.rank,
    updated_at = NOW();