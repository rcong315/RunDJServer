INSERT INTO "artist_top_track"(artist_id, track_id, rank)
VALUES ($1, $2, $3) ON CONFLICT (artist_id, track_id) DO
UPDATE
SET rank = EXCLUDED.rank,
    updated_at = NOW();