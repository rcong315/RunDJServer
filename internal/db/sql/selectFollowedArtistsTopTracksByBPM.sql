SELECT t.track_id,
    t.bpm
FROM "user_followed_artist" ufa
    JOIN "artist_top_track" att ON ufa.artist_id = att.artist_id
    JOIN "track" t ON att.track_id = t.track_id
WHERE ufa.user_id = $1
    AND t.bpm BETWEEN $2 AND $3
    AND NOT EXISTS (
        SELECT 1
        FROM "user_track_interaction" uti
        WHERE uti.track_id = t.track_id
            AND uti.user_id = ufa.user_id
            AND uti.feedback < 0
    );
