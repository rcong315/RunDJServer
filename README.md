# RunDJServer

This is the backend server for the RunDJ iOS app, providing APIs for music recommendation based on running pace (BPM matching) and integrating with Spotify for music data and playlist creation. The project includes both a **REST API server** and a **background crawler service** for continuous music data processing.

## 🎵 Features

### API Server
- **BPM-based music recommendations**: Find songs that match your running pace
- **Spotify integration**: Authenticate users and create playlists
- **User feedback system**: Learn from user preferences to improve recommendations
- **RESTful API**: Clean, documented endpoints for the mobile app

### Crawler Service
- **Continuous music data crawling**: Automatically discovers and processes Spotify music data
- **Priority-based job queue**: Prioritizes missing audio features over discovery
- **Rate limiting & fault tolerance**: Respects Spotify API limits with automatic retries
- **Prometheus metrics**: Built-in monitoring and observability
- **Scalable worker pool**: Configurable concurrent processing

## 🛠 Tech Stack

- **Go 1.24+**: Backend server with Gin web framework
- **PostgreSQL**: Database for storing music data and user preferences
- **Spotify Web API**: Music data source and playlist management
- **Prometheus**: Metrics and monitoring
- **Docker**: Containerization for easy deployment

## 🚀 Getting Started

### Prerequisites

- Go 1.24 or later
- PostgreSQL database
- Spotify Developer Account (for API credentials)
- Token service (for Spotify authentication)

### Environment Variables

Create a `.env` file in the root directory:

```env
# Database
DATABASE_URL=postgresql://username:password@localhost:5432/rundj

# Spotify API
SPOTIFY_CLIENT_ID=your_spotify_client_id
SPOTIFY_CLIENT_SECRET=your_spotify_client_secret

# Token Service (shared between server and crawler)
TOKEN_URL=your_token_service_url
TOKEN_API_KEY=your_token_api_key

# Server Configuration
PORT=8080
DEBUG=true

# Crawler Configuration (optional)
LOG_LEVEL=info
METRICS_PORT=9090
```

### Installation

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd RunDJServer
   ```

2. **Install dependencies**
   ```bash
   go mod download
   ```

3. **Set up database**
   ```bash
   # Run your database migrations here
   # The crawler requires additional columns:
   # ALTER TABLE artist ADD COLUMN last_crawled_at TIMESTAMP;
   # ALTER TABLE album ADD COLUMN tracks_fetched_at TIMESTAMP;
   ```

4. **Build services**
   ```bash
   # Build API server
   go build -o server ./cmd/server
   
   # Build crawler service
   go build -o crawler ./cmd/crawler
   ```

## 🏃‍♂️ Running the Services

### API Server

```bash
# Run with default settings
./server

# Or run directly with Go
go run cmd/server/main.go
```

The server will start on port 8080 (or PORT environment variable).

### Crawler Service

```bash
# Run with default settings
./crawler

# Run with custom configuration
./crawler -workers=16 -crawl-interval=10m -log-level=debug -metrics-port=9091

# Available options:
# -workers: Number of worker goroutines (default: 8)
# -crawl-interval: Interval between crawl cycles (default: 5m)
# -log-level: Log level - debug, info, warn, error (default: info)
# -metrics-port: Prometheus metrics port (default: 9090)
```

## 📡 API Endpoints

### Authentication
- `POST /api/v1/spotify/auth/token` - Get Spotify access token
- `POST /api/v1/spotify/auth/refresh` - Refresh Spotify token

### User Management
- `POST /api/v1/user/register` - Register a new user

### Music Recommendations
- `GET /api/v1/songs/bpm/:bpm` - Get songs matching a specific BPM

### Playlists
- `POST /api/v1/playlist/bpm/:bpm` - Create a playlist for a specific BPM

### Feedback
- `POST /api/v1/song/:songId/feedback` - Submit feedback for a song

## 🤖 Crawler Architecture

The crawler service runs independently and processes music data continuously:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│    Scheduler    │───▶│   Job Queue     │───▶│    Workers      │
│                 │    │                 │    │                 │
│ - Missing Audio │    │ Priority-based  │    │ - Spotify API   │
│ - Stale Data    │    │ - High: Audio   │    │ - Database      │
│ - Discovery     │    │ - Medium: Stale │    │ - Rate Limiting │
│                 │    │ - Low: Discovery│    │                 │
└─────────────────┘    └─────────────────┘    └─────────────────┘
```

### Job Types

1. **High Priority - Missing Audio Features** (every 5 minutes)
   - Finds tracks without audio features
   - Critical for app functionality

2. **Medium Priority - Stale Data Refresh** (every hour)
   - Refreshes tracks older than 30 days
   - Keeps data current

3. **Low Priority - Discovery** (every 6-12 hours)
   - Artist Discovery: Crawls artist albums
   - Album Discovery: Fetches album tracks
   - Expands music catalog

## 📊 Monitoring

### Prometheus Metrics

The crawler exposes metrics on `/metrics` endpoint:

- `rundj_crawler_tracks_processed_total`: Total tracks processed
- `rundj_crawler_errors_total`: Total crawl errors
- `rundj_crawler_queue_depth`: Current job queue depth
- `rundj_crawler_api_calls_total`: Total Spotify API calls
- `rundj_crawler_api_errors_total`: Total API errors
- `rundj_crawler_job_duration_seconds`: Job processing duration

### Health Checks

Monitor the services by:
1. API server health endpoint
2. Crawler metrics endpoint health
3. Database connectivity
4. Queue depth and processing rates

## 🗄 Database Schema

The application uses PostgreSQL with tables for:

### Core Tables
- **Users**: User accounts and Spotify data
- **Tracks**: Music tracks with audio features and BPM data
- **Artists**: Artist information and crawl timestamps
- **Albums**: Album information and track fetch timestamps
- **Audio Features**: Detailed Spotify audio analysis

### Relationship Tables
- User preferences and feedback
- Artist-album-track relationships
- User saved/followed content

### Crawler-Specific Columns
```sql
-- Required for crawler functionality
ALTER TABLE artist ADD COLUMN last_crawled_at TIMESTAMP;
ALTER TABLE album ADD COLUMN tracks_fetched_at TIMESTAMP;
```

## 🐳 Docker Deployment

### API Server
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o server ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/server .
EXPOSE 8080
CMD ["./server"]
```

### Crawler Service
```dockerfile
FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o crawler ./cmd/crawler

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/crawler .
EXPOSE 9090
CMD ["./crawler"]
```

### Docker Compose
```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: rundj
      POSTGRES_USER: rundj
      POSTGRES_PASSWORD: ${POSTGRES_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
    ports:
      - "5432:5432"

  rundj-server:
    build: 
      context: .
      dockerfile: Dockerfile.server
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - SPOTIFY_CLIENT_ID=${SPOTIFY_CLIENT_ID}
      - SPOTIFY_CLIENT_SECRET=${SPOTIFY_CLIENT_SECRET}
      - TOKEN_URL=${TOKEN_URL}
      - TOKEN_API_KEY=${TOKEN_API_KEY}
    ports:
      - "8080:8080"
    depends_on:
      - postgres

  rundj-crawler:
    build:
      context: .
      dockerfile: Dockerfile.crawler
    environment:
      - DATABASE_URL=${DATABASE_URL}
      - TOKEN_URL=${TOKEN_URL}
      - TOKEN_API_KEY=${TOKEN_API_KEY}
      - LOG_LEVEL=info
    ports:
      - "9090:9090"
    depends_on:
      - postgres

volumes:
  postgres_data:
```

## 🔧 Development

### Project Structure
```
RunDJServer/
├── cmd/
│   ├── server/          # API server entry point
│   └── crawler/         # Crawler service entry point
├── internal/
│   ├── crawler/         # Crawler service logic
│   ├── db/              # Database operations
│   ├── service/         # API service handlers
│   └── spotify/         # Spotify API client
├── README.md            # This file
└── README_CRAWLER.md    # Detailed crawler documentation
```

### Adding Features

1. **New API endpoints**: Add handlers in `internal/service/`
2. **New crawler jobs**: Extend `internal/crawler/` components
3. **Database changes**: Update `internal/db/` functions
4. **Spotify integration**: Modify `internal/spotify/` client

### Testing

```bash
# Run all tests
go test ./...

# Run with race detection
go test -race ./...

# Test specific package
go test ./internal/crawler/...
```

## 🚀 Production Deployment

### Recommended Setup

1. **Load Balancer** → Multiple API server instances
2. **Single Crawler Instance** (or multiple with job coordination)
3. **PostgreSQL** with connection pooling
4. **Monitoring** with Prometheus + Grafana
5. **Logging** with structured JSON logs

### Scaling Considerations

- **API Server**: Stateless, can scale horizontally
- **Crawler**: Single instance recommended, or implement job coordination
- **Database**: Use connection pooling and read replicas if needed
- **Rate Limiting**: Monitor Spotify API usage across all services

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Ensure all tests pass (`go test ./...`)
6. Submit a pull request

### Code Style

- Follow Go conventions and `gofmt`
- Add comprehensive error handling
- Include unit tests for new features
- Document public functions and types
- Use structured logging with zap

## 📝 License

[Add your license information here]

## 🆘 Troubleshooting

### Common Issues

**API Server**
- Check database connectivity
- Verify Spotify API credentials
- Ensure token service is accessible

**Crawler**
- Monitor queue depth (shouldn't grow indefinitely)
- Check Spotify API rate limits
- Verify database schema includes crawler columns

**Both Services**
- Check environment variables
- Verify token service authentication
- Monitor logs for detailed error information

### Getting Help

1. Check the logs with debug level: `-log-level=debug`
2. Monitor Prometheus metrics for insights
3. Verify database connectivity and schema
4. Test Spotify API credentials independently
