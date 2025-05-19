INSERT INTO "track" (
        track_id,
        name,
        artist_ids,
        album_id,
        popularity,
        duration_ms,
        available_markets,
        audio_features,
        bpm
    )
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9) ON CONFLICT (track_id) DO
UPDATE
SET name = EXCLUDED.name,
    artist_ids = EXCLUDED.artist_ids,
    album_id = EXCLUDED.album_id,
    popularity = EXCLUDED.popularity,
    duration_ms = EXCLUDED.duration_ms,
    available_markets = EXCLUDED.available_markets,
    audio_features = EXCLUDED.audio_features,
    updated_at = NOW();