package opensubtitles

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
)

type Client struct {
	APIKey string
	client *http.Client
	token  string
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type LoginResponse struct {
	Token string `json:"token"`
}

type Subtitle struct {
	ID         string `json:"id"`
	FileID     int    `json:"file_id"`
	Language   string `json:"language"`
	MovieHash  string `json:"moviehash"`
	MovieBytes int64  `json:"moviebytesize"`
	FileName   string `json:"filename"`
	URL        string `json:"url"`
}

type SearchResponse struct {
	Data []struct {
		ID         string `json:"id"`
		Type       string `json:"type"`
		Attributes struct {
			SubtitleID string `json:"subtitle_id"`
			Language   string `json:"language"`
			URL        string `json:"url"`
			Files      []struct {
				FileID   int    `json:"file_id"`
				FileName string `json:"file_name"`
			} `json:"files"`
			MovieHash string `json:"moviehash"`
			Release   string `json:"release"`
		} `json:"attributes"`
	} `json:"data"`
}

type DownloadResponse struct {
	Link string `json:"link"`
}

func NewClient(apiKey string) *Client {
	return &Client{
		APIKey: apiKey,
		client: &http.Client{},
	}
}

func (c *Client) SearchSubtitles(movieName string, imdbID string, language string) ([]Subtitle, error) {
	searchURL := "https://api.opensubtitles.com/api/v1/subtitles"
	
	params := url.Values{}
	if imdbID != "" {
		params.Add("imdb_id", imdbID)
	}
	if movieName != "" {
		params.Add("query", movieName)
	}
	params.Add("languages", language)
	
	req, err := http.NewRequest("GET", searchURL+"?"+params.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "subtitle-hunt v1.0")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to search subtitles: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	var searchResp SearchResponse
	if err := json.Unmarshal(body, &searchResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}
	
	var subtitles []Subtitle
	for _, item := range searchResp.Data {
		subtitle := Subtitle{
			ID:       item.Attributes.SubtitleID,
			Language: item.Attributes.Language,
			URL:      item.Attributes.URL,
		}
		
		if len(item.Attributes.Files) > 0 {
			subtitle.FileName = item.Attributes.Files[0].FileName
			subtitle.FileID = item.Attributes.Files[0].FileID
		}
		
		subtitles = append(subtitles, subtitle)
	}
	
	return subtitles, nil
}

func (c *Client) DownloadSubtitle(subtitle *Subtitle) ([]byte, error) {
	downloadURL := fmt.Sprintf("https://api.opensubtitles.com/api/v1/download")
	
	reqBody := map[string]interface{}{
		"file_id": subtitle.FileID,
		"sub_format": "srt",
	}
	
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}
	
	log.Printf("DEBUG: Download request body: %s", string(jsonBody))
	
	req, err := http.NewRequest("POST", downloadURL, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	
	req.Header.Set("Api-Key", c.APIKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "subtitle-hunt v1.0")
	
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download subtitle: %w", err)
	}
	defer resp.Body.Close()
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	fmt.Printf("DEBUG: Download response (status %d): %s\n", resp.StatusCode, string(body))
	
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("download API error (status %d): %s", resp.StatusCode, string(body))
	}
	
	var downloadResp DownloadResponse
	if err := json.Unmarshal(body, &downloadResp); err != nil {
		return nil, fmt.Errorf("failed to parse download response (body: %s): %w", string(body), err)
	}
	
	fileResp, err := c.client.Get(downloadResp.Link)
	if err != nil {
		return nil, fmt.Errorf("failed to download file: %w", err)
	}
	defer fileResp.Body.Close()
	
	return io.ReadAll(fileResp.Body)
}

func (c *Client) FindBestSubtitle(movieName string, language string) (*Subtitle, error) {
	subtitles, err := c.SearchSubtitles(movieName, "", language)
	if err != nil {
		return nil, err
	}
	
	if len(subtitles) == 0 {
		return nil, fmt.Errorf("no subtitles found")
	}
	
	log.Printf("DEBUG: Found %d subtitles, using first one with ID: %s, FileID: %d", len(subtitles), subtitles[0].ID, subtitles[0].FileID)
	return &subtitles[0], nil
}

func extractMovieName(videoPath string) string {
	fileName := filepath.Base(videoPath)
	name := strings.TrimSuffix(fileName, filepath.Ext(fileName))
	
	name = strings.ReplaceAll(name, ".", " ")
	name = strings.ReplaceAll(name, "_", " ")
	
	return strings.TrimSpace(name)
}