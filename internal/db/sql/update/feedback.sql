UPDATE user_track_interaction
SET feedback += $3
WHERE user_id = $1
    AND track_id = $2,
    updated_at = NOW();