SELECT track_id
FROM track
    JOIN user_track_relation USING (track_id)
WHERE user_id = $1
    AND(audio_features->>'tempo')::float BETWEEN $2 AND $3
    AND(sources && $4);