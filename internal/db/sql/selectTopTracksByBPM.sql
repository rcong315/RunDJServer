SELECT t.track_id,
    t.bpm
FROM "user_top_track" utt
    JOIN "track" t ON utt.track_id = t.track_id
WHERE utt.user_id = $1
    AND t.bpm BETWEEN $2 AND $3
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = utt.user_id
            AND uti.feedback < 0
    );