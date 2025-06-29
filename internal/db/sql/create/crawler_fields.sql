-- Add crawler-specific timestamp fields for tracking crawl status

-- Add last_crawled_at to artist table for tracking when artist albums were last crawled
ALTER TABLE "artist" ADD COLUMN IF NOT EXISTS last_crawled_at TIMESTAMP;

-- Add tracks_fetched_at to album table for tracking when album tracks were last fetched
ALTER TABLE "album" ADD COLUMN IF NOT EXISTS tracks_fetched_at TIMESTAMP;

-- Create indexes for efficient querying of stale data
CREATE INDEX IF NOT EXISTS idx_artist_last_crawled_at ON "artist" (last_crawled_at);
CREATE INDEX IF NOT EXISTS idx_album_tracks_fetched_at ON "album" (tracks_fetched_at);

-- Create index for audio features queries
CREATE INDEX IF NOT EXISTS idx_track_audio_features_null ON "track" (track_id) WHERE audio_features IS NULL;
CREATE INDEX IF NOT EXISTS idx_track_updated_at ON "track" (updated_at);