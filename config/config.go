package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	JellyfinURL         string
	JellyfinAPIKey      string
	JellyfinUserID      string
	OpenSubtitlesKey    string
	Port                int
	SubtitleDirectory   string
	EnableDirectSave    bool
	JellyfinPathPrefix  string
	ContainerPathPrefix string
}

func Load() *Config {
	godotenv.Load()
	
	port := 8080
	if p := os.Getenv("PORT"); p != "" {
		if parsed, err := strconv.Atoi(p); err == nil {
			port = parsed
		}
	}

	return &Config{
		JellyfinURL:         getEnv("JELLYFIN_URL", "http://localhost:8096"),
		JellyfinAPIKey:      getEnv("JELLYFIN_API_KEY", ""),
		JellyfinUserID:      getEnv("JELLYFIN_USER_ID", ""),
		OpenSubtitlesKey:    getEnv("OPENSUBTITLES_API_KEY", ""),
		SubtitleDirectory:   getEnv("SUBTITLE_DIRECTORY", "./downloads"),
		EnableDirectSave:    getBoolEnv("ENABLE_DIRECT_SAVE", true),
		JellyfinPathPrefix:  getEnv("JELLYFIN_PATH_PREFIX", ""),
		ContainerPathPrefix: getEnv("CONTAINER_PATH_PREFIX", ""),
		Port:                port,
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
	}
	return defaultValue
}

func (c *Config) MapJellyfinPathToContainer(jellyfinPath string) string {
	// If no path mapping is configured, return the original path
	if c.JellyfinPathPrefix == "" || c.ContainerPathPrefix == "" {
		return jellyfinPath
	}
	
	// Replace the Jellyfin path prefix with the container path prefix
	if strings.HasPrefix(jellyfinPath, c.JellyfinPathPrefix) {
		return strings.Replace(jellyfinPath, c.JellyfinPathPrefix, c.ContainerPathPrefix, 1)
	}
	
	// If the path doesn't match the expected prefix, return as-is
	return jellyfinPath
}