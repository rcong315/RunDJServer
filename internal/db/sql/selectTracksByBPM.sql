SELECT track_id, (audio_features->>'tempo')::float AS bpm
FROM track
    JOIN user_track_relation USING (track_id)
WHERE user_id = $1
    AND(audio_features->>'tempo')::float BETWEEN $2 AND $3
    AND(sources && $4)
    AND(feedback >= 0);