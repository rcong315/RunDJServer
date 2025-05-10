INSERT INTO "user_playlist" (user_id, playlist_id)
VALUES ($1, $2) ON CONFLICT (user_id, playlist_id) DO NOTHING;