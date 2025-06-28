SELECT t.track_id,
    t.bpm
FROM "user_top_artist" uta
    JOIN "artist_album" aa ON uta.artist_id = aa.artist_id
    JOIN "album_track" atr ON aa.album_id = atr.album_id
    JOIN "album" a ON aa.album_id = a.album_id
    JOIN "track" t ON atr.track_id = t.track_id
WHERE uta.user_id = $1
    AND a.album_type = 'album'
    AND t.bpm BETWEEN $2 AND $3
    AND t.time_signature = 4
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = uta.user_id
            AND uti.feedback < 0
    );
