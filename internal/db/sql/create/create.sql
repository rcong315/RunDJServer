CREATE TABLE IF NOT EXISTS "user" (
    user_id VARCHAR(255) PRIMARY KEY,
    email VARCHAR(255) UNIQUE,
    display_name VARCHAR(255),
    country VARCHAR(2),
    followers INT,
    product TEXT,
    image_urls TEXT [] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS "track" (
    track_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    artist_ids TEXT [] DEFAULT '{}',
    album_id VARCHAR(255),
    popularity INT,
    duration_ms INT,
    available_markets TEXT [] DEFAULT '{}',
    audio_features JSONB,
    bpm FLOAT,
    time_signature INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS "playlist" (
    playlist_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id VARCHAR(255) NOT NULL,
    public BOOLEAN DEFAULT true,
    followers INT DEFAULT 0,
    image_urls TEXT [] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS "artist" (
    artist_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    genres TEXT [] DEFAULT '{}',
    popularity INT,
    followers INT,
    image_urls TEXT [] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS "album" (
    album_id VARCHAR(255) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    artist_ids TEXT [] DEFAULT '{}',
    genres TEXT [] DEFAULT '{}',
    popularity INT,
    album_type TEXT,
    total_tracks INT,
    release_date TEXT,
    available_markets TEXT [] DEFAULT '{}',
    image_urls TEXT [] DEFAULT '{}',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS "user_top_track" (
    user_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    rank INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (track_id) REFERENCES "track" (track_id)
);
CREATE TABLE IF NOT EXISTS "user_saved_track" (
    user_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (track_id) REFERENCES "track" (track_id)
);
CREATE TABLE IF NOT EXISTS "user_playlist" (
    user_id VARCHAR(255) NOT NULL,
    playlist_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, playlist_id),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (playlist_id) REFERENCES "playlist" (playlist_id)
);
CREATE TABLE IF NOT EXISTS "user_top_artist" (
    user_id VARCHAR(255) NOT NULL,
    artist_id VARCHAR(255) NOT NULL,
    rank INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (artist_id) REFERENCES "artist" (artist_id)
);
CREATE TABLE IF NOT EXISTS "user_followed_artist" (
    user_id VARCHAR(255) NOT NULL,
    artist_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, artist_id),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (artist_id) REFERENCES "artist" (artist_id)
);
CREATE TABLE IF NOT EXISTS "user_saved_album" (
    user_id VARCHAR(255) NOT NULL,
    album_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, album_id),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (album_id) REFERENCES "album" (album_id)
);
CREATE TABLE IF NOT EXISTS "user_track_interaction" (
    user_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    feedback INT,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, track_id, feedback),
    FOREIGN KEY (user_id) REFERENCES "user" (user_id),
    FOREIGN KEY (track_id) REFERENCES "track" (track_id)
);
CREATE TABLE IF NOT EXISTS "playlist_track" (
    playlist_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (playlist_id, track_id),
    FOREIGN KEY (playlist_id) REFERENCES "playlist" (playlist_id),
    FOREIGN KEY (track_id) REFERENCES "track" (track_id)
);
CREATE TABLE IF NOT EXISTS "artist_top_track" (
    artist_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    rank INT NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (artist_id, track_id),
    FOREIGN KEY (artist_id) REFERENCES "artist" (artist_id),
    FOREIGN KEY (track_id) REFERENCES "track" (track_id)
);
CREATE TABLE IF NOT EXISTS "artist_album" (
    artist_id VARCHAR(255) NOT NULL,
    album_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (artist_id, album_id),
    FOREIGN KEY (artist_id) REFERENCES "artist" (artist_id),
    FOREIGN KEY (album_id) REFERENCES "album" (album_id)
);
CREATE TABLE IF NOT EXISTS "album_track" (
    album_id VARCHAR(255) NOT NULL,
    track_id VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    PRIMARY KEY (album_id, track_id),
    FOREIGN KEY (album_id) REFERENCES "album" (album_id),
    FOREIGN KEY (track_id) REFERENCES "track" (track_id)
);

-- Recommended Indexes
CREATE INDEX IF NOT EXISTS idx_track_bpm ON "track" (bpm);
CREATE INDEX IF NOT EXISTS idx_track_time_signature ON "track" (time_signature);
CREATE INDEX IF NOT EXISTS idx_user_track_interaction_track_user ON "user_track_interaction" (track_id, user_id);
CREATE INDEX IF NOT EXISTS idx_album_type ON "album" (album_type);
CREATE INDEX IF NOT EXISTS idx_album_track_track_id ON "album_track" (track_id);
CREATE INDEX IF NOT EXISTS idx_artist_album_album_id ON "artist_album" (album_id);
CREATE INDEX IF NOT EXISTS idx_artist_top_track_track_id ON "artist_top_track" (track_id);
CREATE INDEX IF NOT EXISTS idx_playlist_track_track_id ON "playlist_track" (track_id);
