INSERT INTO "album_track"(album_id, track_id)
VALUES ($1, $2) ON CONFLICT (album_id, track_id) DO NOTHING;