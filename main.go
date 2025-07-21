package main

import (
	"fmt"
	"log"
	"net/http"

	"subtitle-hunt/config"
	"subtitle-hunt/internal/handlers"
	"subtitle-hunt/internal/jellyfin"
	"subtitle-hunt/internal/opensubtitles"
)

func main() {
	cfg := config.Load()

	if cfg.JellyfinAPIKey == "" {
		log.Fatal("JELLYFIN_API_KEY environment variable is required")
	}

	if cfg.OpenSubtitlesKey == "" {
		log.Fatal("OPENSUBTITLES_API_KEY environment variable is required")
	}

	if cfg.JellyfinUserID == "" {
		log.Fatal("JELLYFIN_USER_ID environment variable is required")
	}

	jellyfinClient := jellyfin.NewClient(cfg.JellyfinURL, cfg.JellyfinAPIKey, cfg.JellyfinUserID)
	openSubtitlesClient := opensubtitles.NewClient(cfg.OpenSubtitlesKey)

	handler := handlers.NewHandler(jellyfinClient, openSubtitlesClient, cfg)

	http.HandleFunc("/", handler.IndexHandler)
	http.HandleFunc("/process/", handler.ProcessHandler)
	http.HandleFunc("/status", handler.StatusHandler)

	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("Starting subtitle-hunt server on %s", addr)
	log.Printf("Jellyfin URL: %s", cfg.JellyfinURL)
	log.Printf("Web interface: http://localhost%s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}