package handlers

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"subtitle-hunter/config"
	"subtitle-hunter/internal/jellyfin"
	"subtitle-hunter/internal/opensubtitles"
	"subtitle-hunter/internal/subtitle"
	"subtitle-hunter/internal/translator"
)

type Handler struct {
	JellyfinClient      *jellyfin.Client
	OpenSubtitlesClient *opensubtitles.Client
	Translator          *translator.GoogleTranslator
	Parser              *subtitle.SRTParser
	Config              *config.Config
}

type MediaItemView struct {
	ID           string
	Name         string
	Type         string
	Path         string
	SeriesName   string
	SeasonName   string
	SeasonNumber int
	EpisodeNumber int
}

type SeriesGroup struct {
	Name    string
	Seasons map[int]*SeasonGroup
}

type SeasonGroup struct {
	Number   int
	Name     string
	Episodes []MediaItemView
}

type OrganizedMedia struct {
	Series map[string]*SeriesGroup
	Movies []MediaItemView
}

func NewHandler(jf *jellyfin.Client, os *opensubtitles.Client, cfg *config.Config) *Handler {
	return &Handler{
		JellyfinClient:      jf,
		OpenSubtitlesClient: os,
		Translator:          translator.NewGoogleTranslator(),
		Parser:              subtitle.NewSRTParser(),
		Config:              cfg,
	}
}

func (h *Handler) organizeMedia(items []jellyfin.MediaItem) *OrganizedMedia {
	organized := &OrganizedMedia{
		Series: make(map[string]*SeriesGroup),
		Movies: []MediaItemView{},
	}

	for _, item := range items {
		viewItem := MediaItemView{
			ID:            item.ID,
			Name:          item.Name,
			Type:          item.Type,
			Path:          item.Path,
			SeriesName:    item.SeriesName,
			SeasonName:    item.SeasonName,
			SeasonNumber:  item.ParentIndexNumber,
			EpisodeNumber: item.IndexNumber,
		}

		if item.Type == "Episode" {
			seriesName := item.SeriesName
			if seriesName == "" {
				seriesName = "Unknown Series"
			}

			if organized.Series[seriesName] == nil {
				organized.Series[seriesName] = &SeriesGroup{
					Name:    seriesName,
					Seasons: make(map[int]*SeasonGroup),
				}
			}

			seasonNum := item.ParentIndexNumber
			if seasonNum == 0 {
				seasonNum = 1 // Default to season 1 if not specified
			}

			if organized.Series[seriesName].Seasons[seasonNum] == nil {
				organized.Series[seriesName].Seasons[seasonNum] = &SeasonGroup{
					Number:   seasonNum,
					Name:     item.SeasonName,
					Episodes: []MediaItemView{},
				}
			}

			organized.Series[seriesName].Seasons[seasonNum].Episodes = append(
				organized.Series[seriesName].Seasons[seasonNum].Episodes, viewItem)
		} else {
			organized.Movies = append(organized.Movies, viewItem)
		}
	}

	// Sort episodes within each season by episode number
	for _, series := range organized.Series {
		for _, season := range series.Seasons {
			sort.Slice(season.Episodes, func(i, j int) bool {
				return season.Episodes[i].EpisodeNumber < season.Episodes[j].EpisodeNumber
			})
		}
	}

	// Sort movies by name
	sort.Slice(organized.Movies, func(i, j int) bool {
		return organized.Movies[i].Name < organized.Movies[j].Name
	})

	return organized
}

func (h *Handler) IndexHandler(w http.ResponseWriter, r *http.Request) {
	items, err := h.JellyfinClient.GetMediaWithoutChineseSubtitles()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch media: %v", err), http.StatusInternalServerError)
		return
	}

	organized := h.organizeMedia(items)

	tmpl := `
<!DOCTYPE html>
<html>
<head>
    <title>Subtitle Hunter</title>
    <style>
        body { 
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Arial, sans-serif; 
            margin: 0; padding: 20px; background-color: #f5f5f5; 
        }
        .container { max-width: 1200px; margin: 0 auto; background: white; padding: 20px; border-radius: 8px; box-shadow: 0 2px 10px rgba(0,0,0,0.1); }
        h1 { color: #333; margin-bottom: 30px; text-align: center; }
        .search-box { 
            width: 100%; padding: 12px; margin-bottom: 20px; border: 1px solid #ddd; 
            border-radius: 6px; font-size: 16px; box-sizing: border-box;
        }
        .search-box:focus { outline: none; border-color: #4CAF50; }
        
        .series { margin-bottom: 30px; border: 1px solid #e1e1e1; border-radius: 6px; overflow: hidden; }
        .series-header { 
            background: #f8f9fa; padding: 15px; font-weight: bold; font-size: 18px; 
            border-bottom: 1px solid #e1e1e1; cursor: pointer; user-select: none;
            display: flex; justify-content: space-between; align-items: center;
        }
        .series-header:hover { background: #e9ecef; }
        .toggle { font-size: 14px; color: #666; }
        
        .season { border-top: 1px solid #f0f0f0; }
        .season-header { 
            background: #fafafa; padding: 12px 15px; font-weight: 600; 
            border-bottom: 1px solid #f0f0f0; font-size: 16px; color: #555;
        }
        
        .episodes { background: white; }
        .episode { 
            display: flex; justify-content: space-between; align-items: center; 
            padding: 12px 15px; border-bottom: 1px solid #f8f8f8; 
        }
        .episode:last-child { border-bottom: none; }
        .episode:hover { background: #f9f9f9; }
        
        .episode-info { flex: 1; }
        .episode-name { font-weight: 500; color: #333; margin-bottom: 4px; }
        .episode-details { font-size: 14px; color: #666; }
        
        .movies-section { margin-top: 30px; }
        .movies-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 15px; }
        .movie-card { 
            background: white; border: 1px solid #e1e1e1; border-radius: 6px; padding: 15px;
            display: flex; justify-content: space-between; align-items: center;
        }
        .movie-card:hover { box-shadow: 0 2px 8px rgba(0,0,0,0.1); }
        
        .button { 
            background-color: #4CAF50; color: white; padding: 8px 16px; 
            border: none; border-radius: 4px; cursor: pointer; font-size: 14px;
            transition: background-color 0.2s;
        }
        .button:hover { background-color: #45a049; }
        .button:disabled { opacity: 0.6; cursor: not-allowed; }
        
        .hidden { display: none; }
        .no-results { text-align: center; color: #666; padding: 40px; font-style: italic; }
    </style>
</head>
<body>
    <div class="container">
        <h1>Media Missing Traditional Chinese Subtitles</h1>
        
        <input type="text" class="search-box" placeholder="Search shows, movies, or episodes..." 
               oninput="filterContent(this.value)">

        <div id="content">
            {{range $seriesName, $series := .Series}}
            <div class="series" data-series="{{$seriesName}}">
                <div class="series-header" onclick="toggleSeries(this)">
                    <span>{{$seriesName}}</span>
                    <span class="toggle">▼</span>
                </div>
                <div class="series-content">
                    {{range $seasonNum, $season := $series.Seasons}}
                    <div class="season">
                        <div class="season-header">
                            Season {{$season.Number}}{{if $season.Name}} - {{$season.Name}}{{end}}
                        </div>
                        <div class="episodes">
                            {{range $season.Episodes}}
                            <div class="episode" data-episode="{{.Name}}">
                                <div class="episode-info">
                                    <div class="episode-name">{{.EpisodeNumber}}. {{.Name}}</div>
                                    <div class="episode-details">Episode {{.EpisodeNumber}}</div>
                                </div>
                                <button class="button" onclick="findSubtitle('{{.ID}}', this)">Find Subtitle</button>
                            </div>
                            {{end}}
                        </div>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
            
            {{if .Movies}}
            <div class="movies-section">
                <h2>Movies</h2>
                <div class="movies-grid">
                    {{range .Movies}}
                    <div class="movie-card" data-movie="{{.Name}}">
                        <div>
                            <div class="episode-name">{{.Name}}</div>
                            <div class="episode-details">Movie</div>
                        </div>
                        <button class="button" onclick="findSubtitle('{{.ID}}', this)">Find Subtitle</button>
                    </div>
                    {{end}}
                </div>
            </div>
            {{end}}
        </div>

        <div id="no-results" class="no-results hidden">
            No matching content found. Try a different search term.
        </div>
    </div>

    <script>
        // Toggle series expansion
        function toggleSeries(header) {
            const content = header.nextElementSibling;
            const toggle = header.querySelector('.toggle');
            
            if (content.style.display === 'none') {
                content.style.display = 'block';
                toggle.textContent = '▼';
            } else {
                content.style.display = 'none';
                toggle.textContent = '▶';
            }
        }

        // Search functionality
        function filterContent(searchTerm) {
            const term = searchTerm.toLowerCase();
            const series = document.querySelectorAll('.series');
            const movies = document.querySelectorAll('.movie-card');
            let hasResults = false;

            // Filter series and episodes
            series.forEach(seriesEl => {
                const seriesName = seriesEl.dataset.series.toLowerCase();
                const episodes = seriesEl.querySelectorAll('.episode');
                let hasMatchingEpisodes = false;

                episodes.forEach(episode => {
                    const episodeName = episode.dataset.episode.toLowerCase();
                    if (episodeName.includes(term) || seriesName.includes(term)) {
                        episode.style.display = 'flex';
                        hasMatchingEpisodes = true;
                        hasResults = true;
                    } else {
                        episode.style.display = 'none';
                    }
                });

                if (hasMatchingEpisodes || seriesName.includes(term)) {
                    seriesEl.style.display = 'block';
                    // Auto-expand if there's a match
                    if (term) {
                        const content = seriesEl.querySelector('.series-content');
                        const toggle = seriesEl.querySelector('.toggle');
                        content.style.display = 'block';
                        toggle.textContent = '▼';
                    }
                } else {
                    seriesEl.style.display = 'none';
                }
            });

            // Filter movies
            movies.forEach(movie => {
                const movieName = movie.dataset.movie.toLowerCase();
                if (movieName.includes(term)) {
                    movie.style.display = 'flex';
                    hasResults = true;
                } else {
                    movie.style.display = 'none';
                }
            });

            // Show/hide no results message
            document.getElementById('no-results').classList.toggle('hidden', hasResults || !term);
        }

        // Subtitle processing
        async function findSubtitle(itemId, button) {
            const originalText = button.textContent;
            button.textContent = 'Processing...';
            button.disabled = true;
            
            try {
                const response = await fetch('/process/' + itemId, {
                    method: 'POST'
                });
                
                const result = await response.text();
                
                if (response.ok) {
                    button.textContent = 'Success!';
                    button.style.backgroundColor = '#28a745';
                    setTimeout(() => {
                        location.reload();
                    }, 2000);
                } else {
                    button.textContent = 'Error: ' + result;
                    button.style.backgroundColor = '#dc3545';
                    button.disabled = false;
                }
            } catch (error) {
                button.textContent = 'Network Error';
                button.style.backgroundColor = '#dc3545';
                button.disabled = false;
            }
        }

        // Initialize - collapse all series by default
        document.addEventListener('DOMContentLoaded', () => {
            document.querySelectorAll('.series-content').forEach(content => {
                content.style.display = 'none';
            });
            document.querySelectorAll('.toggle').forEach(toggle => {
                toggle.textContent = '▶';
            });
        });
    </script>
</body>
</html>`

	t := template.Must(template.New("index").Parse(tmpl))
	if err := t.Execute(w, organized); err != nil {
		http.Error(w, "Template execution failed", http.StatusInternalServerError)
	}
}

func (h *Handler) ProcessHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	itemID := strings.TrimPrefix(r.URL.Path, "/process/")
	if itemID == "" {
		http.Error(w, "Item ID required", http.StatusBadRequest)
		return
	}

	log.Printf("Processing subtitle for item: %s", itemID)

	item, err := h.JellyfinClient.GetItem(itemID)
	if err != nil {
		log.Printf("Error getting item details for %s: %v", itemID, err)
		http.Error(w, fmt.Sprintf("Failed to get item details: %v", err), http.StatusInternalServerError)
		return
	}
	
	var videoPath string
	if len(item.MediaSources) > 0 {
		videoPath = item.MediaSources[0].Path
	} else {
		videoPath = item.Path
	}
	log.Printf("Got video path: %s", videoPath)

	searchQuery := h.JellyfinClient.GetSearchQuery(*item)
	log.Printf("Searching subtitles for: %s", searchQuery)

	var saveLocation string
	
	chineseSubtitle, err := h.OpenSubtitlesClient.FindBestSubtitle(searchQuery, "zh-TW")
	if err == nil && chineseSubtitle != nil {
		log.Printf("Found Chinese subtitle directly")
		location, err := h.downloadAndSaveSubtitle(chineseSubtitle, videoPath, "zh-Hant")
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to save Chinese subtitle: %v", err), http.StatusInternalServerError)
			return
		}
		saveLocation = location
	} else {
		log.Printf("Chinese subtitle not found, searching for English")
		englishSubtitle, err := h.OpenSubtitlesClient.FindBestSubtitle(searchQuery, "en")
		if err != nil {
			log.Printf("Error finding English subtitle: %v", err)
			http.Error(w, fmt.Sprintf("No subtitles found: %v", err), http.StatusNotFound)
			return
		}

		log.Printf("Found English subtitle, starting translation process...")
		location, err := h.translateAndSaveSubtitle(englishSubtitle, videoPath)
		if err != nil {
			log.Printf("Error in translation process: %v", err)
			http.Error(w, fmt.Sprintf("Failed to translate subtitle: %v", err), http.StatusInternalServerError)
			return
		}
		saveLocation = location
		log.Printf("Translation completed successfully")
	}

	log.Printf("Refreshing Jellyfin metadata")
	if err := h.JellyfinClient.RefreshMetadata(itemID); err != nil {
		log.Printf("Warning: Failed to refresh metadata: %v", err)
	}

	var successMessage string
	if saveLocation == "media" {
		successMessage = "Subtitle processed successfully and saved to media directory (Jellyfin will detect automatically)"
	} else {
		successMessage = fmt.Sprintf("Subtitle processed successfully and saved to downloads directory (%s)", h.Config.SubtitleDirectory)
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(successMessage))
}

func (h *Handler) downloadAndSaveSubtitle(subtitle *opensubtitles.Subtitle, videoPath, language string) (string, error) {
	content, err := h.OpenSubtitlesClient.DownloadSubtitle(subtitle)
	if err != nil {
		return "", fmt.Errorf("failed to download subtitle: %w", err)
	}

	subtitlePath, saveLocation := h.generateSubtitlePath(videoPath, language)
	if err := os.WriteFile(subtitlePath, content, 0644); err != nil {
		return "", fmt.Errorf("failed to write subtitle file: %w", err)
	}
	
	log.Printf("Subtitle saved to %s directory: %s", saveLocation, subtitlePath)
	return saveLocation, nil
}

func (h *Handler) translateAndSaveSubtitle(subtitle *opensubtitles.Subtitle, videoPath string) (string, error) {
	log.Printf("Downloading English subtitle...")
	content, err := h.OpenSubtitlesClient.DownloadSubtitle(subtitle)
	if err != nil {
		return "", fmt.Errorf("failed to download English subtitle: %w", err)
	}
	log.Printf("Downloaded %d bytes of subtitle content", len(content))

	log.Printf("Parsing SRT content...")
	entries, err := h.Parser.Parse(content)
	if err != nil {
		return "", fmt.Errorf("failed to parse subtitle: %w", err)
	}
	log.Printf("Parsed %d subtitle entries", len(entries))

	log.Printf("Starting translation of %d entries...", len(entries))
	translatedEntries, err := h.Parser.TranslateEntries(entries, h.Translator)
	if err != nil {
		return "", fmt.Errorf("failed to translate subtitle: %w", err)
	}
	log.Printf("Translation completed")

	log.Printf("Formatting translated content...")
	translatedContent := h.Parser.Format(translatedEntries)
	subtitlePath, saveLocation := h.generateSubtitlePath(videoPath, "zh-Hant")
	
	log.Printf("Saving translated subtitle to: %s", subtitlePath)
	if err := os.WriteFile(subtitlePath, []byte(translatedContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write subtitle file: %w", err)
	}
	
	log.Printf("Subtitle saved successfully to %s directory", saveLocation)
	return saveLocation, nil
}

func (h *Handler) generateSubtitlePath(videoPath, language string) (string, string) {
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	fileName := fmt.Sprintf("%s.%s.srt", base, language)
	
	// If direct save is enabled, try to save to the media directory first
	if h.Config.EnableDirectSave {
		// Map the Jellyfin path to container path
		containerPath := h.Config.MapJellyfinPathToContainer(videoPath)
		mediaDir := filepath.Dir(containerPath)
		mediaSubtitlePath := filepath.Join(mediaDir, fileName)
		
		// Check if we can write to the media directory
		if h.canWriteToDirectory(mediaDir) {
			log.Printf("Will save subtitle to media directory: %s", mediaSubtitlePath)
			return mediaSubtitlePath, "media"
		} else {
			log.Printf("Cannot write to media directory %s, falling back to downloads", mediaDir)
		}
	}
	
	// Fallback: save to downloads directory
	downloadsDir := h.Config.SubtitleDirectory
	if err := os.MkdirAll(downloadsDir, 0755); err != nil {
		log.Printf("Warning: Could not create downloads directory: %v", err)
	}
	
	fallbackPath := filepath.Join(downloadsDir, fileName)
	log.Printf("Will save subtitle to downloads directory: %s", fallbackPath)
	return fallbackPath, "downloads"
}

func (h *Handler) canWriteToDirectory(dirPath string) bool {
	// Check if directory exists
	if _, err := os.Stat(dirPath); os.IsNotExist(err) {
		// Try to create the directory
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			log.Printf("Cannot create directory %s: %v", dirPath, err)
			return false
		}
	}
	
	// Try to create a temporary file to test write permissions
	tempFile := filepath.Join(dirPath, ".subtitle-hunter-write-test")
	file, err := os.Create(tempFile)
	if err != nil {
		log.Printf("Cannot write to directory %s: %v", dirPath, err)
		return false
	}
	file.Close()
	os.Remove(tempFile)
	
	return true
}

func extractMovieName(videoPath string) string {
	fileName := filepath.Base(videoPath)
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))

	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")

	return strings.TrimSpace(name)
}

func (h *Handler) StatusHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]string{
		"status": "ok",
		"service": "subtitle-hunter",
	}
	
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}
