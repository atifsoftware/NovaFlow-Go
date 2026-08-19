package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AIService acts as a unified provider adapter for OpenAI and Google Gemini APIs,
// keeping NovaFlow completely dependency-free by using standard HTTP REST payloads.
type AIService struct {
	provider    string
	geminiKey   string
	openaiKey   string
	openaiModel string
	client      *http.Client
	geminiURL   string
	openaiURL   string
}

// NewAIService creates a new AIService instance populated from configurations.
func NewAIService(cfg *Config) *AIService {
	provider := cfg.Get("AI_PROVIDER", "gemini")
	return &AIService{
		provider:    strings.ToLower(provider),
		geminiKey:   cfg.Get("GEMINI_API_KEY", ""),
		openaiKey:   cfg.Get("OPENAI_API_KEY", ""),
		openaiModel: cfg.Get("OPENAI_MODEL", "gpt-3.5-turbo"),
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		geminiURL: cfg.Get("GEMINI_API_URL", "https://generativelanguage.googleapis.com"),
		openaiURL: cfg.Get("OPENAI_API_URL", "https://api.openai.com"),
	}
}

// GenerateText sends a prompt to the configured AI provider (Gemini/OpenAI) and returns the text completion.
func (ai *AIService) GenerateText(prompt string) (string, error) {
	if ai.provider == "openai" {
		return ai.generateOpenAI(prompt)
	}
	return ai.generateGemini(prompt)
}

// ModerateContent performs safety/content checks on the input text.
// Returns true if content is safe, or false if it contains hate speech/harassment/explicit/spam content.
func (ai *AIService) ModerateContent(text string) (bool, error) {
	if ai.provider == "openai" {
		return ai.moderateOpenAI(text)
	}

	// For Gemini, we pass a structured instruction prompt to analyze text safety
	prompt := fmt.Sprintf("Analyze the following text for hate speech, harassment, spam, or explicit content. Return exactly the word 'safe' if it contains none of those issues, or 'unsafe' if it does. Do not include any other characters or explanation.\n\nText: %s", text)
	res, err := ai.generateGemini(prompt)
	if err != nil {
		return false, err
	}
	clean := strings.ToLower(strings.TrimSpace(res))
	return strings.Contains(clean, "safe") && !strings.Contains(clean, "unsafe"), nil
}

// --- Gemini REST API Models & Logic ---

type geminiPart struct {
	Text string `json:"text"`
}

type geminiContent struct {
	Parts []geminiPart `json:"parts"`
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiResponse struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (ai *AIService) generateGemini(prompt string) (string, error) {
	if ai.geminiKey == "" {
		return "", errors.New("gemini: GEMINI_API_KEY is not configured in env")
	}

	url := fmt.Sprintf("%s/v1beta/models/gemini-1.5-flash:generateContent?key=%s", ai.geminiURL, ai.geminiKey)

	reqBody := geminiRequest{
		Contents: []geminiContent{
			{
				Parts: []geminiPart{
					{Text: prompt},
				},
			},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	resp, err := ai.client.Post(url, "application/json", bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("gemini: api request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var geminiResp geminiResponse
	if err := json.NewDecoder(resp.Body).Decode(&geminiResp); err != nil {
		return "", err
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", errors.New("gemini: returned an empty completion candidate")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

// --- OpenAI REST API Models & Logic ---

type openaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type openaiChatRequest struct {
	Model    string          `json:"model"`
	Messages []openaiMessage `json:"messages"`
}

type openaiChatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

func (ai *AIService) generateOpenAI(prompt string) (string, error) {
	if ai.openaiKey == "" {
		return "", errors.New("openai: OPENAI_API_KEY is not configured in env")
	}

	url := fmt.Sprintf("%s/v1/chat/completions", ai.openaiURL)

	reqBody := openaiChatRequest{
		Model: ai.openaiModel,
		Messages: []openaiMessage{
			{Role: "user", Content: prompt},
		},
	}

	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.openaiKey)

	resp, err := ai.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("openai: api request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var oaResp openaiChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&oaResp); err != nil {
		return "", err
	}

	if len(oaResp.Choices) == 0 {
		return "", errors.New("openai: returned empty choices")
	}

	return oaResp.Choices[0].Message.Content, nil
}

type openaiModRequest struct {
	Input string `json:"input"`
}

type openaiModResponse struct {
	Results []struct {
		Flagged bool `json:"flagged"`
	} `json:"results"`
}

func (ai *AIService) moderateOpenAI(text string) (bool, error) {
	if ai.openaiKey == "" {
		return false, errors.New("openai: OPENAI_API_KEY is not configured in env")
	}

	url := fmt.Sprintf("%s/v1/moderations", ai.openaiURL)

	reqBody := openaiModRequest{Input: text}
	jsonBytes, err := json.Marshal(reqBody)
	if err != nil {
		return false, err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return false, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+ai.openaiKey)

	resp, err := ai.client.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return false, fmt.Errorf("openai: moderation request failed with status %d: %s", resp.StatusCode, string(body))
	}

	var modResp openaiModResponse
	if err := json.NewDecoder(resp.Body).Decode(&modResp); err != nil {
		return false, err
	}

	if len(modResp.Results) == 0 {
		return false, errors.New("openai: moderation returned empty results")
	}

	return !modResp.Results[0].Flagged, nil
}
