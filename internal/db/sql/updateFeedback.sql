UPDATE user_track_relation
SET feedback = $3
WHERE user_id = $1
    AND track_id = $2