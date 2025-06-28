SELECT t.track_id,
    t.bpm
FROM "user_saved_album" usa
    JOIN "album_track" atr ON usa.album_id = atr.album_id
    JOIN "track" t ON atr.track_id = t.track_id
WHERE usa.user_id = $1
    AND t.bpm BETWEEN $2 AND $3
    AND t.time_signature = 4
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = usa.user_id
            AND uti.feedback < 0
    );
