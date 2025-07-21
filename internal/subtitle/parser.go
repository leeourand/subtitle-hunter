package subtitle

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type SubtitleEntry struct {
	Index     int
	StartTime string
	EndTime   string
	Text      string
}

type SRTParser struct{}

func NewSRTParser() *SRTParser {
	return &SRTParser{}
}

func (p *SRTParser) Parse(content []byte) ([]SubtitleEntry, error) {
	text := string(content)
	text = strings.ReplaceAll(text, "\r\n", "\n")
	
	blocks := strings.Split(text, "\n\n")
	var entries []SubtitleEntry
	
	timeRegex := regexp.MustCompile(`(\d{2}:\d{2}:\d{2},\d{3})\s*-->\s*(\d{2}:\d{2}:\d{2},\d{3})`)
	
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		
		lines := strings.Split(block, "\n")
		if len(lines) < 3 {
			continue
		}
		
		index, err := strconv.Atoi(strings.TrimSpace(lines[0]))
		if err != nil {
			continue
		}
		
		timeMatch := timeRegex.FindStringSubmatch(lines[1])
		if len(timeMatch) != 3 {
			continue
		}
		
		text := strings.Join(lines[2:], "\n")
		text = strings.TrimSpace(text)
		
		entries = append(entries, SubtitleEntry{
			Index:     index,
			StartTime: timeMatch[1],
			EndTime:   timeMatch[2],
			Text:      text,
		})
	}
	
	return entries, nil
}

func (p *SRTParser) Format(entries []SubtitleEntry) string {
	var result strings.Builder
	
	for i, entry := range entries {
		if i > 0 {
			result.WriteString("\n")
		}
		
		result.WriteString(fmt.Sprintf("%d\n", entry.Index))
		result.WriteString(fmt.Sprintf("%s --> %s\n", entry.StartTime, entry.EndTime))
		result.WriteString(entry.Text)
		result.WriteString("\n")
	}
	
	return result.String()
}

func (p *SRTParser) TranslateEntries(entries []SubtitleEntry, translator Translator) ([]SubtitleEntry, error) {
	var translated []SubtitleEntry
	
	for i, entry := range entries {
		if i%10 == 0 {
			log.Printf("Translating entry %d/%d...", i+1, len(entries))
		}
		
		translatedText, err := p.translateWithRetry(translator, entry.Text, 3)
		if err != nil {
			log.Printf("Warning: Failed to translate entry %d ('%s'): %v. Using original text.", entry.Index, entry.Text, err)
			translatedText = entry.Text // Fallback to original text
		}
		
		translatedEntry := SubtitleEntry{
			Index:     entry.Index,
			StartTime: entry.StartTime,
			EndTime:   entry.EndTime,
			Text:      translatedText,
		}
		
		translated = append(translated, translatedEntry)
	}
	
	return translated, nil
}

func (p *SRTParser) translateWithRetry(translator Translator, text string, maxRetries int) (string, error) {
	var lastErr error
	
	for attempt := 1; attempt <= maxRetries; attempt++ {
		if attempt > 1 {
			// Wait between retries (exponential backoff)
			waitTime := time.Duration(attempt-1) * time.Second
			log.Printf("Retrying translation attempt %d/%d after %v...", attempt, maxRetries, waitTime)
			time.Sleep(waitTime)
		}
		
		result, err := translator.TranslateToChineseTraditional(text)
		if err == nil {
			return result, nil
		}
		
		lastErr = err
		log.Printf("Translation attempt %d failed: %v", attempt, err)
	}
	
	return "", fmt.Errorf("translation failed after %d attempts: %w", maxRetries, lastErr)
}

type Translator interface {
	TranslateToChineseTraditional(text string) (string, error)
}