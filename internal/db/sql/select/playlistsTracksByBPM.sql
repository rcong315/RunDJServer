SELECT t.track_id,
    t.bpm
FROM "user_playlist" up
    JOIN "playlist_track" pt ON up.playlist_id = pt.playlist_id
    JOIN "track" t ON pt.track_id = t.track_id
WHERE up.playlist_id = $1
    AND t.bpm BETWEEN $2 AND $3
    AND t.time_signature = 4
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = up.user_id
            AND uti.feedback < 0
    );
