SELECT t.track_id,
    t.bpm
FROM "user_top_artist" uta
    JOIN "artist_top_track" att ON uta.artist_id = att.artist_id
    JOIN "track" t ON att.track_id = t.track_id
WHERE uta.user_id = $1
    AND t.bpm BETWEEN $2 AND $3
    AND t.time_signature = 4
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = uta.user_id
            AND uti.feedback < 0
    );
