INSERT INTO "track_playlist_relation" (track_id, playlist_id)
VALUES ($1, $2) ON CONFLICT (track_id, playlist_id)
UPDATE
SET sources = (
        SELECT array_agg(DISTINCT element)
        FROM unnest(
                array_cat(
                    track_playlist_relation.sources,
                    EXCLUDED.sources
                )
            ) AS element
    );