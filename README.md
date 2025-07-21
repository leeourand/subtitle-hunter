# Subtitle Hunter

A lightweight Go web application that automates subtitle downloading and translation for Jellyfin media servers. Perfect for running on low-powered NAS devices.

## Features

- **Smart Discovery**: Automatically finds media missing Traditional Chinese subtitles in your Jellyfin library
- **Dual Strategy**: Downloads Traditional Chinese subtitles directly or translates English subtitles
- **OpenSubtitles Integration**: Uses OpenSubtitles API v2 for subtitle search and download
- **Google Translate**: Translates English subtitles to Traditional Chinese while preserving SRT formatting
- **Jellyfin Integration**: Automatically triggers metadata refresh after subtitle installation
- **Modern Web UI**: Clean, organized interface with search functionality and series/season grouping

## Quick Start with Docker

### Option 1: Use Pre-built Image (Recommended)

```bash
# Download the production docker-compose file
wget https://raw.githubusercontent.com/your-username/subtitle-hunter/main/docker-compose.prod.yml

# Download environment template
wget https://raw.githubusercontent.com/your-username/subtitle-hunter/main/.env.docker -O .env

# Edit .env with your configuration
nano .env

# Update docker-compose.prod.yml with your media paths and user ID
nano docker-compose.prod.yml

# Start the application
docker-compose -f docker-compose.prod.yml up -d
```

### Option 2: Build from Source

### 1. Prerequisites
- Docker and Docker Compose installed
- Jellyfin server running and accessible
- OpenSubtitles API account ([register here](https://www.opensubtitles.com/api))

### 2. Setup

```bash
# Clone the project
git clone <repository-url>
cd subtitle-hunter

# Copy environment template
cp .env.docker .env

# Edit .env with your configuration
nano .env
```

### 3. Configure Environment Variables

Edit `.env` with your settings:

```bash
# Jellyfin Configuration
JELLYFIN_URL=http://your-jellyfin-server:8096
JELLYFIN_API_KEY=your_jellyfin_api_key
JELLYFIN_USER_ID=your_jellyfin_user_id

# OpenSubtitles Configuration  
OPENSUBTITLES_API_KEY=your_opensubtitles_api_key
```

### 4. Run with Docker Compose

```bash
# Start the application
docker-compose up -d

# Check logs
docker-compose logs -f

# Stop the application
docker-compose down
```

### 5. Access the Application

Open `http://your-nas-ip:8080` in your browser.

## Getting API Keys

### Jellyfin API Key
1. Go to Jellyfin Dashboard → API Keys
2. Click "+" to create new API key
3. Copy the generated key

### Jellyfin User ID
1. Go to Jellyfin Dashboard → Users
2. Click on your user
3. Look at the URL - the user ID is the long string at the end
4. Or use: `curl -H "X-Emby-Token: YOUR_API_KEY" http://jellyfin:8096/Users`

### OpenSubtitles API Key
1. Register at [OpenSubtitles.com](https://www.opensubtitles.com/api)
2. Get your API key from the developer section

## Manual Installation

### 1. Build and Run

```bash
# Install dependencies
go mod tidy

# Build
go build

# Run
./subtitle-hunter
```

### 2. Environment Variables

```bash
export JELLYFIN_URL="http://localhost:8096"
export JELLYFIN_API_KEY="your_api_key"
export JELLYFIN_USER_ID="your_user_id"
export OPENSUBTITLES_API_KEY="your_opensubtitles_key"
export SUBTITLE_DIRECTORY="./downloads"
export PORT="8080"
```

## Configuration Options

| Variable | Description | Default |
|----------|-------------|---------|
| `JELLYFIN_URL` | Jellyfin server URL | `http://localhost:8096` |
| `JELLYFIN_API_KEY` | Jellyfin API key | Required |
| `JELLYFIN_USER_ID` | Jellyfin user ID | Required |
| `OPENSUBTITLES_API_KEY` | OpenSubtitles API key | Required |
| `SUBTITLE_DIRECTORY` | Download directory | `./downloads` |
| `ENABLE_DIRECT_SAVE` | Save subtitles to media directory | `true` |
| `JELLYFIN_PATH_PREFIX` | Jellyfin's media path prefix | `/data/media` |
| `CONTAINER_PATH_PREFIX` | Container's media path prefix | `/media` |
| `PORT` | Server port | `8080` |

## How It Works

1. **Discovery**: Scans Jellyfin library for movies/episodes without Traditional Chinese subtitles
2. **Search**: Looks for Traditional Chinese subtitles on OpenSubtitles
3. **Fallback**: If not found, downloads English subtitles and translates them using Google Translate
4. **Save**: Stores subtitles with proper naming convention (`filename.zh-Hant.srt`)
5. **Refresh**: Triggers Jellyfin metadata refresh to recognize new subtitles

## Features

### Web Interface
- **Series Grouping**: Episodes organized by series and season
- **Search Functionality**: Real-time search across all content
- **Collapsible Sections**: Keep interface organized
- **Progress Tracking**: Visual feedback for processing status

### Subtitle Processing
- **Intelligent Search**: Uses proper series/episode names instead of filenames
- **HTML Tag Cleaning**: Removes formatting tags before translation
- **Retry Logic**: Handles temporary API failures gracefully
- **Fallback Handling**: Uses original text if translation fails

## Direct Media Directory Saving

The application can save subtitles directly to your media directories alongside video files, eliminating the need for manual file placement.

### Path Mapping Configuration

Configure path mapping to translate Jellyfin paths to container paths:

```yaml
environment:
  - ENABLE_DIRECT_SAVE=true
  - JELLYFIN_PATH_PREFIX=/data/media    # How Jellyfin sees your media
  - CONTAINER_PATH_PREFIX=/media        # How container sees your media
volumes:
  - /your/media/path:/media             # Mount your actual media directory
user: "1000:1000"                       # Run as user with media permissions
```

### Example Scenarios

**Synology NAS:**
```yaml
environment:
  - JELLYFIN_PATH_PREFIX=/data/media
  - CONTAINER_PATH_PREFIX=/media
volumes:
  - /volume1/media:/media
user: "1026:100"  # Synology media user
```

**QNAP NAS:**
```yaml
environment:
  - JELLYFIN_PATH_PREFIX=/share/media
  - CONTAINER_PATH_PREFIX=/media
volumes:
  - /share/CACHEDEV1_DATA/media:/media
user: "1000:1000"
```

### Fallback Behavior

- **Primary**: Saves to media directory next to video files
- **Fallback**: Saves to downloads directory if media directory isn't writable
- **Status**: UI shows where subtitles were saved

## Docker Volumes

The docker-compose setup includes:
- `./downloads:/app/downloads` - Downloaded subtitles (accessible on host)
- `/your/media:/media` - Your media directory for direct subtitle placement

## Health Checks

The container includes health checks that verify the application is responding correctly.

## Troubleshooting

### Common Issues

1. **Permission Errors**: Ensure the downloads directory is writable
2. **API Rate Limits**: Google Translate may rate limit - the app includes retry logic
3. **Network Issues**: Ensure the container can reach Jellyfin and OpenSubtitles APIs

### Logs

```bash
# View container logs
docker-compose logs subtitle-hunter

# Follow logs in real-time
docker-compose logs -f subtitle-hunter
```

## Container Images

Pre-built Docker images are automatically built and published via GitHub Actions:

- **Latest stable**: `ghcr.io/your-username/subtitle-hunter:latest`
- **Specific versions**: `ghcr.io/your-username/subtitle-hunter:v1.0.0`
- **Development**: `ghcr.io/your-username/subtitle-hunter:main-<commit-sha>`

### Supported Architectures

- `linux/amd64` (Intel/AMD 64-bit)
- `linux/arm64` (ARM 64-bit, including Apple Silicon and ARM-based NAS)

### CI/CD Pipeline

The GitHub Actions workflow automatically:
- Builds multi-architecture Docker images
- Publishes to GitHub Container Registry (GHCR)
- Creates artifact attestations for security
- Caches layers for faster builds
- Tags releases appropriately

## License

MIT License - See LICENSE file for details
