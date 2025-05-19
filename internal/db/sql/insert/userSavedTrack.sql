INSERT INTO "user_saved_track" (user_id, track_id)
VALUES ($1, $2) ON CONFLICT (user_id, track_id) DO NOTHING;