INSERT INTO "artist_album"(artist_id, album_id)
VALUES ($1, $2) ON CONFLICT (artist_id, album_id) DO NOTHING;