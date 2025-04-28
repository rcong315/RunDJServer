INSERT INTO "user_album_relation" (user_id, album_id, sources)
VALUES ($1, $2, $3) ON CONFLICT (user_id, album_id) DO
UPDATE
SET sources = (
        SELECT array_agg(DISTINCT element)
        FROM unnest(
                array_cat(user_album_relation.sources, EXCLUDED.sources)
            ) AS element
    );