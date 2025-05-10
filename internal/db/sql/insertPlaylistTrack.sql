INSERT INTO "playlist_track" (playlist_id, track_id)
VALUES ($1, $2) ON CONFLICT (playlist_id, track_id) DO NOTHING;