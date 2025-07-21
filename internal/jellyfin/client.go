package jellyfin

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"path/filepath"
	"strings"
)

type Client struct {
	BaseURL string
	APIKey  string
	client  *http.Client
	UserID  string
}

type MediaItem struct {
	ID           string `json:"Id"`
	Name         string `json:"Name"`
	Type         string `json:"Type"`
	Path         string `json:"Path"`
	SeriesName   string `json:"SeriesName"`
	SeasonName   string `json:"SeasonName"`
	IndexNumber  int    `json:"IndexNumber"`
	ParentIndexNumber int `json:"ParentIndexNumber"`
	ProductionYear int   `json:"ProductionYear"`
	MediaSources []struct {
		Path string `json:"Path"`
	} `json:"MediaSources"`
	MediaStreams []struct {
		Type     string `json:"Type"`
		Language string `json:"Language"`
		Codec    string `json:"Codec"`
		IsExternal bool  `json:"IsExternal"`
	} `json:"MediaStreams"`
}

type ItemsResponse struct {
	Items []MediaItem `json:"Items"`
}

func NewClient(baseURL, apiKey, userID string) *Client {
	return &Client{
		BaseURL: strings.TrimSuffix(baseURL, "/"),
		APIKey:  apiKey,
		UserID:  userID,
		client:  &http.Client{},
	}
}

func (c *Client) GetMediaWithoutChineseSubtitles() ([]MediaItem, error) {
	url := fmt.Sprintf("%s/Users/%s/Items?Recursive=true&IncludeItemTypes=Movie,Episode&Fields=Path,MediaSources,MediaStreams", c.BaseURL, c.UserID)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("X-Emby-Token", c.APIKey)
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch items: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var itemsResp ItemsResponse
	if err := json.Unmarshal(body, &itemsResp); err != nil {
		return nil, fmt.Errorf("failed to parse response (response: %s): %w", string(body), err)
	}

	var filtered []MediaItem
	for _, item := range itemsResp.Items {
		if !c.hasChineseSubtitle(item) {
			filtered = append(filtered, item)
		}
	}

	return filtered, nil
}

func (c *Client) hasChineseSubtitle(item MediaItem) bool {
	for _, stream := range item.MediaStreams {
		if stream.Type == "Subtitle" && (stream.Language == "zh-TW" || stream.Language == "zh-Hant" || stream.Language == "chi") {
			return true
		}
	}

	if len(item.MediaSources) > 0 {
		videoPath := item.MediaSources[0].Path
		dir := filepath.Dir(videoPath)
		base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
		
		chineseSubtitles := []string{
			filepath.Join(dir, base+".zh-Hant.srt"),
			filepath.Join(dir, base+".zh-TW.srt"),
			filepath.Join(dir, base+".chi.srt"),
		}
		
		for _, subPath := range chineseSubtitles {
			if c.fileExists(subPath) {
				return true
			}
		}
	}

	return false
}

func (c *Client) fileExists(path string) bool {
	return false
}

func (c *Client) RefreshMetadata(itemID string) error {
	url := fmt.Sprintf("%s/Items/%s/Refresh?metadataRefreshMode=FullRefresh&replaceAllMetadata=false", c.BaseURL, itemID)
	
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("X-Emby-Token", c.APIKey)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to refresh metadata: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}


func (c *Client) GetItem(itemID string) (*MediaItem, error) {
	url := fmt.Sprintf("%s/Users/%s/Items/%s", c.BaseURL, c.UserID, itemID)
	
	log.Printf("DEBUG: Calling Jellyfin API: %s", url)
	
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("X-Emby-Token", c.APIKey)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch item: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var item MediaItem
	if err := json.Unmarshal(body, &item); err != nil {
		return nil, fmt.Errorf("failed to parse item (response: %s): %w", string(body), err)
	}

	return &item, nil
}

func (c *Client) GetVideoPath(itemID string) (string, error) {
	item, err := c.GetItem(itemID)
	if err != nil {
		return "", err
	}

	if len(item.MediaSources) > 0 {
		return item.MediaSources[0].Path, nil
	}

	return item.Path, nil
}

func (c *Client) GetSearchQuery(item MediaItem) string {
	if item.Type == "Episode" {
		// For TV episodes, use "SeriesName S##E##" format
		if item.SeriesName != "" {
			season := item.ParentIndexNumber
			episode := item.IndexNumber
			return fmt.Sprintf("%s S%02dE%02d", item.SeriesName, season, episode)
		}
	} else if item.Type == "Movie" {
		// For movies, use the name and year if available
		if item.ProductionYear > 0 {
			return fmt.Sprintf("%s %d", item.Name, item.ProductionYear)
		}
	}
	
	// Fallback to just the name
	return item.Name
}