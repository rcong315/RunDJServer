INSERT INTO "user_saved_album" (user_id, album_id)
VALUES ($1, $2) ON CONFLICT (user_id, album_id) DO NOTHING;