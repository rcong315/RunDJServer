INSERT INTO "track_playlist_relation" (track_id, playlist_id, sources)
VALUES ($1, $2, $3) ON CONFLICT (track_id, playlist_id) DO
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

    -- TODO: Rename to playlist_track_relation