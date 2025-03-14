-- Description: SQL script to create the database schema

CREATE TABLE IF NOT EXISTS "user" (
    user_id TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    display_name TEXT NOT NULL,
    country TEXT,
    followers INTEGER DEFAULT 0,
    product TEXT,
    image_urls TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE TYPE audio_features_type AS (
    danceability FLOAT,
    energy FLOAT,
    key INTEGER,
    loudness FLOAT,
    mode INTEGER,
    speechiness FLOAT,
    acousticness FLOAT,
    instrumentallness FLOAT,
    liveness FLOAT,
    valence FLOAT,
    tempo FLOAT,
    duration_ms INTEGER,
    time_signature INTEGER
);

CREATE TABLE IF NOT EXISTS "track" (
    track_id TEXT PRIMARY KEY,
    user_ids TEXT[] NOT NULL,
    name TEXT NOT NULL,
    artist_ids TEXT[] NOT NULL,
    album_id TEXT NOT NULL,
    popularity INTEGER DEFAULT 0,
    duration_ms INTEGER NOT NULL,
    available_markets TEXT[] DEFAULT '{}',
    audio_features audio_features_type,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_track_user_ids ON "track" USING GIN (user_ids);
CREATE INDEX idx_track_artist_ids ON "track" USING GIN (artist_ids);
CREATE INDEX idx_track_album_id ON "track" ((audio_features).tempo);

CREATE OR REPLACE FUNCTION update_modified_timestamp()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_user_modtime
BEFORE UPDATE ON "user"
FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

CREATE TRIGGER update_track_modtime
BEFORE UPDATE ON "track"
FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();