INSERT INTO user_track_interaction (user_id, track_id, feedback)
VALUES ($1, $2, $3) ON CONFLICT (user_id, track_id) DO
UPDATE
SET feedback = user_track_interaction.feedback + EXCLUDED.feedback,
    updated_at = NOW();