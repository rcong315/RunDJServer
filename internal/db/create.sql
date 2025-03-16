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

CREATE TABLE IF NOT EXISTS "track" (
    track_id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    artist_ids TEXT[] NOT NULL,
    album_id TEXT NOT NULL,
    popularity INTEGER DEFAULT 0,
    duration_ms INTEGER NOT NULL,
    available_markets TEXT[] DEFAULT '{}',
    audio_features JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- CREATE TABLE IF NOT EXISTS "album" (
--     album_id TEXT PRIMARY KEY,
--     name TEXT NOT NULL,
--     artist_ids TEXT[] NOT NULL,
--     genres TEXT[] DEFAULT '{}',
--     popularity INTEGER DEFAULT 0,
--     album_type TEXT NOT NULL,
--     total_tracks INTEGER DEFAULT 0,
--     release_date DATE NOT NULL,
--     available_markets TEXT[] DEFAULT '{}',
--     image_urls TEXT[] DEFAULT '{}',
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );

-- CREATE TABLE IF NOT EXISTS "artist" (
--     artist_id TEXT PRIMARY KEY,
--     name TEXT NOT NULL,
--     genres TEXT[] DEFAULT '{}',
--     popularity INTEGER DEFAULT 0,
--     followers INTEGER DEFAULT 0,
--     image_urls TEXT[] DEFAULT '{}',
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );

-- CREATE TABLE IF NOT EXISTS "playlist" (
--     playlist_id TEXT PRIMARY KEY,
--     name TEXT NOT NULL,
--     description TEXT,
--     owner_id TEXT NOT NULL,
--     public BOOLEAN DEFAULT FALSE,
--     followers INTEGER DEFAULT 0,
--     image_urls TEXT[] DEFAULT '{}',
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
-- );

CREATE TABLE IF NOT EXISTS "user_track_relation" (
    user_id TEXT REFERENCES user(user_id) ON DELETE CASCADE,
    track_id TEXT REFERENCES track(track_id) ON DELETE CASCADE,
    sources TEXT[] NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id),
    FOREIGN KEY (user_id) REFERENCES "user"(user_id) ON DELETE CASCADE,
    FOREIGN KEY (track_id) REFERENCES "track"(track_id) ON DELETE CASCADE
);

-- CREATE TABLE IF NOT EXISTS "user_album_relation" (
--     user_id TEXT REFERENCES user(user_id) ON DELETE CASCADE,
--     album_id TEXT REFERENCES album(album_id) ON DELETE CASCADE,
--     sources TEXT[] NOT NULL,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     PRIMARY KEY (user_id, album_id),
--     FOREIGN KEY (user_id) REFERENCES "user"(user_id) ON DELETE CASCADE,
--     FOREIGN KEY (album_id) REFERENCES "album"(album_id) ON DELETE CASCADE
-- );

-- CREATE TABLE IF NOT EXISTS "user_artist_relation" (
--     user_id TEXT REFERENCES user(user_id) ON DELETE CASCADE,
--     artist_id TEXT REFERENCES artist(artist_id) ON DELETE CASCADE,
--     sources TEXT[] NOT NULL,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     PRIMARY KEY (user_id, artist_id),
--     FOREIGN KEY (user_id) REFERENCES "user"(user_id) ON DELETE CASCADE,
--     FOREIGN KEY (artist_id) REFERENCES "artist"(artist_id) ON DELETE CASCADE
-- );

-- CREATE TABLE IF NOT EXISTS "user_playlist_relation" (
--     user_id TEXT REFERENCES user(user_id) ON DELETE CASCADE,
--     playlist_id TEXT REFERENCES playlist(playlist_id) ON DELETE CASCADE,
--     sources TEXT[] NOT NULL,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     PRIMARY KEY (user_id, playlist_id),
--     FOREIGN KEY (user_id) REFERENCES "user"(user_id) ON DELETE CASCADE,
--     FOREIGN KEY (playlist_id) REFERENCES "playlist"(playlist_id) ON DELETE CASCADE
-- );

-- CREATE TABLE IF NOT EXISTS "track_album_relation" (
--     track_id TEXT REFERENCES track(track_id) ON DELETE CASCADE,
--     album_id TEXT REFERENCES album(album_id) ON DELETE CASCADE,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     PRIMARY KEY (track_id, album_id),
--     FOREIGN KEY (track_id) REFERENCES "track"(track_id) ON DELETE CASCADE,
--     FOREIGN KEY (album_id) REFERENCES "album"(album_id) ON DELETE CASCADE
-- );

-- CREATE TABLE IF NOT EXISTS "track_artist_relation" (
--     track_id TEXT REFERENCES track(track_id) ON DELETE CASCADE,
--     artist_id TEXT REFERENCES artist(artist_id) ON DELETE CASCADE,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     PRIMARY KEY (track_id, artist_id),
--     FOREIGN KEY (track_id) REFERENCES "track"(track_id) ON DELETE CASCADE,
--     FOREIGN KEY (artist_id) REFERENCES "artist"(artist_id) ON DELETE CASCADE
-- );

-- CREATE TABLE IF NOT EXISTS "track_playlist_relation" (
--     track_id TEXT REFERENCES track(track_id) ON DELETE CASCADE,
--     playlist_id TEXT REFERENCES playlist(playlist_id) ON DELETE CASCADE,
--     created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
--     PRIMARY KEY (track_id, playlist_id),
--     FOREIGN KEY (track_id) REFERENCES "track"(track_id) ON DELETE CASCADE,
--     FOREIGN KEY (playlist_id) REFERENCES "playlist"(playlist_id) ON DELETE CASCADE
-- );

-- TODO: INDEXES

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

-- CREATE TRIGGER update_album_modtime
-- BEFORE UPDATE ON "album"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_artist_modtime
-- BEFORE UPDATE ON "artist"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_playlist_modtime
-- BEFORE UPDATE ON "playlist"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();  

CREATE TRIGGER update_user_track_relation_modtime
BEFORE UPDATE ON "user_track_relation"
FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_user_album_relation_modtime
-- BEFORE UPDATE ON "user_album_relation"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_user_artist_relation_modtime
-- BEFORE UPDATE ON "user_artist_relation"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_user_playlist_relation_modtime
-- BEFORE UPDATE ON "user_playlist_relation"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_track_album_relation_modtime
-- BEFORE UPDATE ON "track_album_relation"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_track_artist_relation_modtime
-- BEFORE UPDATE ON "track_artist_relation"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();

-- CREATE TRIGGER update_track_playlist_relation_modtime
-- BEFORE UPDATE ON "track_playlist_relation"
-- FOR EACH ROW EXECUTE FUNCTION update_modified_timestamp();
