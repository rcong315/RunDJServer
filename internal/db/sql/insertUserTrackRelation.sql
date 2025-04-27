INSERT INTO "user_track_relation" (user_id, track_id, sources)
VALUES ($1, $2, $3) ON CONFLICT (user_id, track_id) DO
UPDATE
SET sources = (
        SELECT array_agg(DISTINCT element)
        FROM unnest(
                array_cat(user_track_relation.sources, EXCLUDED.sources)
            ) AS element
    );