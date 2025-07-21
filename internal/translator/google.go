package translator

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
)

type GoogleTranslator struct {
	client *http.Client
}

type TranslateResponse []interface{}

func NewGoogleTranslator() *GoogleTranslator {
	return &GoogleTranslator{
		client: &http.Client{},
	}
}

func (gt *GoogleTranslator) Translate(text, sourceLang, targetLang string) (string, error) {
	if text == "" {
		return "", nil
	}

	// Clean the text before translation
	cleanText := gt.cleanTextForTranslation(text)
	
	baseURL := "https://translate.googleapis.com/translate_a/single"
	params := url.Values{
		"client": {"gtx"},
		"sl":     {sourceLang},
		"tl":     {targetLang},
		"dt":     {"t"},
		"q":      {cleanText},
	}

	resp, err := gt.client.Get(baseURL + "?" + params.Encode())
	if err != nil {
		return "", fmt.Errorf("failed to call Google Translate API: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response: %w", err)
	}

	// Check if response is HTML (error page)
	if strings.HasPrefix(strings.TrimSpace(string(body)), "<") {
		return "", fmt.Errorf("Google Translate returned HTML error page, possibly rate limited or blocked")
	}

	var result TranslateResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse translation response (body: %s): %w", string(body), err)
	}

	if len(result) == 0 {
		return "", fmt.Errorf("empty translation response")
	}

	translations, ok := result[0].([]interface{})
	if !ok {
		return "", fmt.Errorf("unexpected response format")
	}

	var translatedText strings.Builder
	for _, translation := range translations {
		if translationArray, ok := translation.([]interface{}); ok && len(translationArray) > 0 {
			if translatedPart, ok := translationArray[0].(string); ok {
				translatedText.WriteString(translatedPart)
			}
		}
	}

	return translatedText.String(), nil
}

func (gt *GoogleTranslator) cleanTextForTranslation(text string) string {
	// Remove HTML tags but preserve the content
	htmlTagRegex := regexp.MustCompile(`<[^>]*>`)
	cleanText := htmlTagRegex.ReplaceAllString(text, "")
	
	// Clean up extra whitespace
	cleanText = strings.TrimSpace(cleanText)
	
	// Replace multiple spaces with single space
	spaceRegex := regexp.MustCompile(`\s+`)
	cleanText = spaceRegex.ReplaceAllString(cleanText, " ")
	
	return cleanText
}

func (gt *GoogleTranslator) TranslateToChineseTraditional(text string) (string, error) {
	return gt.Translate(text, "en", "zh-TW")
}