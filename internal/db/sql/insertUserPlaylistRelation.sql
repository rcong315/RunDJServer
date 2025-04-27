INSERT INTO "user_playlist_relation" (user_id, playlist_id, sources)
VALUES ($1, $2, $3) ON CONFLICT (user_id, playlist_id) DO
UPDATE
SET sources = (
        SELECT array_agg(DISTINCT element)
        FROM unnest(
                array_cat(user_playlist_relation.sources, EXCLUDED.sources)
            ) AS element
    );