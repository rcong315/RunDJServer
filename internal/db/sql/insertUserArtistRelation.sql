INSERT INTO "user_artist_relation" (user_id, artist_id, sources)
VALUES ($1, $2, $3) ON CONFLICT (user_id, artist_id) DO
UPDATE
SET sources = (
        SELECT array_agg(DISTINCT element)
        FROM unnest(
                array_cat(user_artist_relation.sources, EXCLUDED.sources)
            ) AS element
    );