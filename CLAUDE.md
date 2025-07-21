# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Application Overview

Subtitle Hunter is a Go web application that automates subtitle downloading and translation for Jellyfin media servers. It finds media missing Traditional Chinese subtitles, downloads them from OpenSubtitles, or translates English subtitles using Google Translate while preserving SRT formatting.

## Essential Commands

### Development
```bash
# Build the application
go build

# Run locally (requires .env file)
go run main.go

# Clean dependencies
go mod tidy

# Test Docker build
docker build -t subtitle-hunter .

# Run with Docker Compose (development)
docker-compose up -d

# Run with pre-built image (production)
docker-compose -f docker-compose.prod.yml up -d
```

### Environment Setup
```bash
# Copy environment template
cp .env.docker .env

# Edit with your API keys and configuration
nano .env
```

Required environment variables:
- `JELLYFIN_API_KEY` - Jellyfin API key (required)
- `JELLYFIN_USER_ID` - Jellyfin user ID (required)
- `OPENSUBTITLES_API_KEY` - OpenSubtitles API key (required)
- `JELLYFIN_URL` - Jellyfin server URL
- `ENABLE_DIRECT_SAVE` - Save subtitles to media directory (default: true)
- `JELLYFIN_PATH_PREFIX` - Jellyfin's media path prefix (e.g., "/data/media")
- `CONTAINER_PATH_PREFIX` - Container's media path prefix (e.g., "/media")

## Architecture

### Core Components

**Configuration (`config/`)**: Centralized configuration management with environment variable loading via godotenv. The `Config.MapJellyfinPathToContainer()` method handles path translation between Jellyfin's view and the container's mounted volumes.

**Web Layer (`internal/handlers/`)**: Single handler struct containing all HTTP endpoints and business logic. The `ProcessHandler` orchestrates the entire subtitle workflow - discovery, download, translation, and saving. Contains embedded HTML template with JavaScript for the web UI.

**External Service Clients (`internal/*/`)**: 
- `jellyfin/client.go` - Jellyfin API integration for media discovery and metadata refresh
- `opensubtitles/client.go` - OpenSubtitles API for subtitle search/download with retry logic
- `translator/google.go` - Google Translate integration with HTML tag cleaning and error handling

**Subtitle Processing (`internal/subtitle/`)**: SRT parser that maintains timing and formatting during translation, with retry logic and graceful fallback to original text on translation failures.

### Key Workflows

**Media Discovery**: Jellyfin client queries for movies/episodes → filters items missing Traditional Chinese subtitles → organizes by series/season for UI display.

**Subtitle Processing**: 
1. Try direct Traditional Chinese subtitle download from OpenSubtitles
2. If not found, download English subtitles → parse SRT → translate each entry → reformat → save
3. Dual save strategy: attempt media directory first (next to video files), fallback to downloads directory
4. Trigger Jellyfin metadata refresh

**Path Mapping**: Critical for Docker deployments - translates Jellyfin's internal paths (`/data/media/Movies/Movie.mkv`) to container paths (`/media/Movies/Movie.mkv`) for direct subtitle placement.

### Error Handling Patterns

The application uses extensive retry logic and graceful degradation:
- Translation failures fall back to original English text rather than failing completely
- Media directory write failures fall back to downloads directory
- OpenSubtitles API failures include retry with exponential backoff
- HTML response detection prevents JSON parsing errors from Google Translate

### UI Architecture

Single-page application with vanilla JavaScript. Media organized hierarchically (Series → Seasons → Episodes) with collapsible sections and real-time search filtering. AJAX-based subtitle processing with persistent error display and progress feedback.

## Key Implementation Details

- Uses Go 1.21+ with minimal dependencies (only godotenv)
- HTTP server uses standard library, no frameworks
- Multi-architecture Docker builds (amd64/arm64) via GitHub Actions
- Subtitle files saved with `.zh-Hant.srt` extension for Jellyfin compatibility
- Google Translate uses unofficial API (no key required) with HTML tag sanitization
- All external API interactions include proper error handling and logging
