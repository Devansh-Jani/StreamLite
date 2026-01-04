# StreamLite

StreamLite is a lightweight video streaming platform optimized for low-resource hardware. It provides a YouTube-like interface for browsing and watching videos with features like playback speed control, likes, and comments.

## Features

### Backend (Go)
- **Video Discovery**: Recursively scans a user-specified directory for video files
- **PostgreSQL Database**: Stores video metadata, views, likes, and comments
- **REST API**: Provides endpoints for video listing, streaming, and interactions
- **Auto-Generated Metadata**: Creates titles from filenames if metadata is missing
- **Read-Only Video Files**: No modifications to original video files
- **Separate Configuration**: Dedicated `config` directory for logs and settings

### Frontend (React + Material-UI)
- **Video Grid**: Displays thumbnails, views, and timestamps
- **Video Player**: Full-featured player with:
  - Expand/Fullscreen mode
  - Speed adjustment (1x to 3x)
  - Like button
  - Comments section
- **Responsive Design**: Works on desktop and mobile devices

## Project Structure

```
StreamLite/
├── backend/
│   ├── main.go           # Go backend server
│   ├── schema.sql        # Database schema
│   ├── Dockerfile        # Backend Docker image
│   ├── go.mod            # Go dependencies
│   └── go.sum
├── frontend/
│   ├── src/
│   │   ├── pages/        # React pages
│   │   ├── components/   # React components
│   │   └── api.js        # API client
│   ├── Dockerfile        # Frontend Docker image
│   ├── nginx.conf        # Nginx configuration
│   └── package.json
├── config/               # Configuration and logs
├── videos/               # Video files directory (user-provided)
└── docker-compose.yml    # Full stack deployment
```

## Prerequisites

- Docker and Docker Compose
- Video files in a directory

## Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/Devansh-Jani/StreamLite.git
   cd StreamLite
   ```

2. **Add your video files**
   ```bash
   # Place your video files in the videos directory
   # Supported formats: .mp4, .avi, .mkv, .mov, .wmv, .flv, .webm, .m4v
   mkdir -p videos
   # Copy your videos here
   ```

3. **Start the application**
   ```bash
   docker-compose up -d
   ```

4. **Access the application**
   - Frontend: http://localhost:3000
   - Backend API: http://localhost:8080/api

## Development Setup

### Backend Development

```bash
cd backend
go mod download
export DATABASE_URL="postgres://streamlite:streamlite@localhost:5432/streamlite?sslmode=disable"
export VIDEO_DIR="../videos"
export CONFIG_DIR="../config"
go run main.go
```

### Frontend Development

```bash
cd frontend
npm install
npm start
```

## API Endpoints

### Videos
- `GET /api/videos` - List all videos
- `GET /api/videos/:id` - Get video details
- `GET /api/videos/:id/stream` - Stream video file
- `POST /api/videos/:id/view` - Increment view count
- `POST /api/videos/:id/like` - Toggle like (body: `{"action": "like" | "unlike"}`)

### Comments
- `GET /api/videos/:id/comments` - Get video comments
- `POST /api/videos/:id/comments` - Add comment (body: `{"author": "Name", "content": "Comment"}`)

## Configuration

### Environment Variables

**Backend:**
- `DATABASE_URL` - PostgreSQL connection string (default: `postgres://streamlite:streamlite@localhost:5432/streamlite?sslmode=disable`)
- `VIDEO_DIR` - Path to video directory (default: `./videos`)
- `CONFIG_DIR` - Path to config/logs directory (default: `./config`)
- `PORT` - Server port (default: `8080`)
- `ALLOWED_ORIGINS` - Comma-separated list of allowed CORS origins (default: `http://localhost:3000,http://localhost:80`)

**Frontend:**
- `REACT_APP_API_URL` - Backend API URL (default: `http://localhost:8080/api`)

### Docker Compose

The `docker-compose.yml` file orchestrates three services:
- **postgres**: PostgreSQL database
- **backend**: Go API server
- **frontend**: React app with Nginx

Customize the volumes in `docker-compose.yml` to point to your video directory:
```yaml
volumes:
  - /path/to/your/videos:/videos:ro
```

## Database Schema

### Videos Table
- `id` - Primary key
- `filename` - Original filename
- `filepath` - Full path to video file
- `title` - Display title
- `views` - View count
- `likes` - Like count
- `duration` - Video duration (seconds)
- `file_size` - File size (bytes)
- `created_at` - Record creation timestamp
- `modified_at` - File modification timestamp

### Comments Table
- `id` - Primary key
- `video_id` - Foreign key to videos
- `author` - Comment author name
- `content` - Comment text
- `created_at` - Comment timestamp

## Supported Video Formats

- MP4 (.mp4)
- AVI (.avi)
- MKV (.mkv)
- MOV (.mov)
- WMV (.wmv)
- FLV (.flv)
- WebM (.webm)
- M4V (.m4v)

## Deployment

### Production Deployment

1. Update `docker-compose.yml` with production settings
2. Set secure database credentials
3. Configure reverse proxy (e.g., Nginx) for SSL
4. Mount persistent volumes for database and config

### Lightweight Server Deployment

StreamLite is optimized for low-resource hardware:
- Minimal Go backend footprint
- Efficient video streaming with range requests
- Containerized deployment for easy management
- Read-only video access for safety

## License

MIT License - see LICENSE file for details

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
