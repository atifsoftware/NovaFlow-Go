package core

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newTestConfig(vals map[string]string) *Config {
	return &Config{values: vals}
}

func TestAIServiceConfig(t *testing.T) {
	cfg := newTestConfig(map[string]string{
		"AI_PROVIDER":     "OpenAI",
		"OPENAI_API_KEY":  "test-openai-key",
		"OPENAI_MODEL":    "gpt-4",
		"GEMINI_API_KEY":  "test-gemini-key",
		"GEMINI_API_URL":  "https://custom-gemini.com",
		"OPENAI_API_URL":  "https://custom-openai.com",
	})

	ai := NewAIService(cfg)
	if ai.provider != "openai" {
		t.Errorf("expected provider 'openai', got %q", ai.provider)
	}
	if ai.openaiKey != "test-openai-key" {
		t.Errorf("expected openaiKey 'test-openai-key', got %q", ai.openaiKey)
	}
	if ai.openaiModel != "gpt-4" {
		t.Errorf("expected openaiModel 'gpt-4', got %q", ai.openaiModel)
	}
	if ai.geminiKey != "test-gemini-key" {
		t.Errorf("expected geminiKey 'test-gemini-key', got %q", ai.geminiKey)
	}
	if ai.geminiURL != "https://custom-gemini.com" {
		t.Errorf("expected geminiURL 'https://custom-gemini.com', got %q", ai.geminiURL)
	}
	if ai.openaiURL != "https://custom-openai.com" {
		t.Errorf("expected openaiURL 'https://custom-openai.com', got %q", ai.openaiURL)
	}
}

func TestAIServiceGeminiMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("key") != "mock-gemini-key" {
			t.Errorf("expected API key query param 'mock-gemini-key', got %q", r.URL.Query().Get("key"))
		}

		var reqBody geminiRequest
		if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
			t.Fatal("failed to decode gemini request body")
		}

		if len(reqBody.Contents) == 0 {
			t.Fatal("gemini: contents is empty")
		}
		promptText := reqBody.Contents[0].Parts[0].Text
		if promptText != "Hello AI" && !strings.Contains(promptText, "Analyze the following text") {
			t.Errorf("unexpected request prompt: %q", promptText)
		}

		// Mock response
		resp := geminiResponse{}
		resp.Candidates = []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		}{
			{
				Content: struct {
					Parts []struct {
						Text string `json:"text"`
					} `json:"parts"`
				}{
					Parts: []struct {
						Text string `json:"text"`
					}{
						{Text: "safe"},
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	cfg := newTestConfig(map[string]string{
		"AI_PROVIDER":    "gemini",
		"GEMINI_API_KEY": "mock-gemini-key",
		"GEMINI_API_URL": server.URL,
	})

	ai := NewAIService(cfg)
	ai.client = server.Client()

	text, err := ai.GenerateText("Hello AI")
	if err != nil {
		t.Fatal(err)
	}
	if text != "safe" {
		t.Errorf("expected response 'safe', got %q", text)
	}

	isSafe, err := ai.ModerateContent("some comment")
	if err != nil {
		t.Fatal(err)
	}
	if !isSafe {
		t.Error("expected content to be moderated as safe")
	}
}

func TestAIServiceOpenAIMock(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer mock-openai-key" {
			t.Errorf("expected Bearer token 'mock-openai-key', got %q", r.Header.Get("Authorization"))
		}

		if r.URL.Path == "/v1/chat/completions" {
			var reqBody openaiChatRequest
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Fatal("failed to decode openai request body")
			}

			if len(reqBody.Messages) == 0 || reqBody.Messages[0].Content != "Hello OpenAI" {
				t.Errorf("unexpected request messages: %v", reqBody)
			}

			// Mock Chat Completions response
			resp := openaiChatResponse{}
			resp.Choices = []struct {
				Message struct {
					Content string `json:"content"`
				} `json:"message"`
			}{
				{
					Message: struct {
						Content string `json:"content"`
					}{
						Content: "OpenAI response",
					},
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		if r.URL.Path == "/v1/moderations" {
			var reqBody openaiModRequest
			if err := json.NewDecoder(r.Body).Decode(&reqBody); err != nil {
				t.Fatal("failed to decode moderation request body")
			}

			// Mock Moderation response (flagged = false)
			resp := openaiModResponse{}
			resp.Results = []struct {
				Flagged bool `json:"flagged"`
			}{
				{Flagged: false},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(resp)
			return
		}

		t.Errorf("unexpected request path: %q", r.URL.Path)
	}))
	defer server.Close()

	cfg := newTestConfig(map[string]string{
		"AI_PROVIDER":    "openai",
		"OPENAI_API_KEY": "mock-openai-key",
		"OPENAI_API_URL": server.URL,
	})

	ai := NewAIService(cfg)
	ai.client = server.Client()

	text, err := ai.GenerateText("Hello OpenAI")
	if err != nil {
		t.Fatal(err)
	}
	if text != "OpenAI response" {
		t.Errorf("expected response 'OpenAI response', got %q", text)
	}

	isSafe, err := ai.ModerateContent("some comment")
	if err != nil {
		t.Fatal(err)
	}
	if !isSafe {
		t.Error("expected content to be moderated as safe")
	}
}
