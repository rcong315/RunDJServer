# RunDJ Crawler

The RunDJ Crawler is a background service that maintains fresh and updated data in the database by periodically fetching information from Spotify.

## Features

1. **Missing Data Recovery**: Refetches audio features for tracks with missing BPM or time signature (runs hourly)
2. **Entity Discovery**: Finds and processes artists/albums referenced by tracks but not in database (runs every 6 hours)
3. **Stale Data Refresh**: Updates artists and albums not refreshed in 30+ days (runs daily)
4. **Weekly Playlist Processing**: Deep processes a specific playlist and all its artists (runs weekly)

## Configuration

Set these environment variables:

```bash
# Required - same as main API
SPOTIFY_CLIENT_ID=your_client_id
SPOTIFY_CLIENT_SECRET=your_client_secret
TOKEN_URL=your_token_url
TOKEN_API_KEY=your_token_api_key
DB_HOST=localhost
DB_NAME=rundj
DB_PORT=5432
DB_USER=your_user
DB_PASSWORD=your_password

# Optional - for weekly playlist processing
CRAWLER_WEEKLY_PLAYLIST_ID=spotify_playlist_id
```

## Running the Crawler

```bash
# From project root
go run cmd/crawler/main.go

# Or build and run
go build -o crawler cmd/crawler/main.go
./crawler
```

## Architecture

The crawler uses:
- Worker pool pattern with 32 concurrent workers
- Batch processing for efficient API usage
- Deduplication tracking to avoid redundant processing
- Graceful shutdown on SIGINT/SIGTERM

## Job Types

### RefetchMissingDataJob
- Queries tracks where `bpm = 0` or `time_signature = 0`
- Fetches audio features from Spotify
- Updates track records with correct data

### DiscoverMissingEntitiesJob
- Finds tracks referencing non-existent artists/albums
- Queues processing jobs for missing entities
- Ensures referential integrity

### RefreshStaleDataJob
- Identifies artists/albums not updated in 30+ days
- Refetches latest data from Spotify
- Keeps popularity and follower counts current

### ProcessPlaylistJob
- Processes all tracks in configured playlist
- Deep mode: also processes all artists from tracks
- Updates artist top tracks and albums

### ProcessAlbumJob (Internal)
- Fetches complete album details from Spotify
- Saves album metadata (name, artists, genres, popularity, etc.)
- Processes all album tracks with audio features
- Creates album-track relationships
- Discovers and queues missing artists for processing
- Automatically triggered when missing albums are discovered

## Monitoring

The crawler logs:
- Job start/completion times
- Number of entities processed
- Any errors encountered
- Duration of each crawl stage

## Development

To add new crawl jobs:
1. Create job struct implementing the Execute interface
2. Add scheduling logic in crawler.go
3. Configure intervals as needed