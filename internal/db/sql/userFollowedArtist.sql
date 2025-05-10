INSERT INTO "user_followed_artist" (user_id, artist_id)
VALUES ($1, $2) ON CONFLICT (user_id, artist_id) DO NOTHING;