SELECT t.track_id,
    t.bpm
FROM "user_saved_track" ust
    JOIN "track" t ON ust.track_id = t.track_id
WHERE ust.user_id = $1
    AND t.bpm BETWEEN $2 AND $3
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = ust.user_id
            AND uti.feedback < 0
    );