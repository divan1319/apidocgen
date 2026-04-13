package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/divan1319/apidocgen/pkg/models"
)

type openAICompatClient struct {
	apiKey       string
	systemPrompt string
	model        string
	endpoint     string
}

func chatCompletionsURL(base string) string {
	b := strings.TrimRight(strings.TrimSpace(base), "/")
	if strings.HasSuffix(b, "/chat/completions") {
		return b
	}
	return b + "/chat/completions"
}

func newOpenAICompatClient(cfg Config) (*openAICompatClient, error) {
	prompt, err := loadPrompt(cfg.DocLang)
	if err != nil {
		return nil, err
	}
	base := cfg.ResolvedBaseURL()
	if base == "" {
		return nil, fmt.Errorf("base URL vacía para proveedor %s", cfg.Provider)
	}
	return &openAICompatClient{
		apiKey:       cfg.APIKey,
		systemPrompt: prompt,
		model:        cfg.ResolvedModel(),
		endpoint:     chatCompletionsURL(base),
	}, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model     string        `json:"model"`
	MaxTokens int           `json:"max_tokens"`
	Messages  []chatMessage `json:"messages"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// DocumentEndpoint usa la API estilo OpenAI (chat completions).
func (c *openAICompatClient) DocumentEndpoint(ep models.Endpoint) (*models.EndpointDoc, error) {
	var lastErr error
	wait := 2 * time.Second

	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			fmt.Fprintf(os.Stderr, "    rate limited, retrying in %s...\n", wait)
			time.Sleep(wait)
			wait *= 2
		}

		doc, err, retry := c.attempt(ep)
		if !retry {
			return doc, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("exceeded retries: %w", lastErr)
}

func (c *openAICompatClient) attempt(ep models.Endpoint) (doc *models.EndpointDoc, err error, retry bool) {
	reqBody, _ := json.Marshal(chatRequest{
		Model:     c.model,
		MaxTokens: 4096,
		Messages: []chatMessage{
			{Role: "system", Content: c.systemPrompt},
			{Role: "user", Content: buildPrompt(ep)},
		},
	})

	req, _ := http.NewRequest("POST", c.endpoint, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llamando a API de chat: %w", err), false
	}
	defer resp.Body.Close()

	bodyBytes, _ := io.ReadAll(resp.Body)

	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429)"), true
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error (%d)", resp.StatusCode), true
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (%d): %s", resp.StatusCode, truncateBody(bodyBytes, 512)), false
	}

	var apiResp chatResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err), false
	}

	if len(apiResp.Choices) == 0 || apiResp.Choices[0].Message.Content == "" {
		return nil, fmt.Errorf("empty response from chat API"), false
	}

	result, err := parseDocResponse(ep, apiResp.Choices[0].Message.Content)
	return result, err, false
}

func truncateBody(b []byte, max int) string {
	s := strings.TrimSpace(string(b))
	if len(s) <= max {
		return s
	}
	return s[:max] + "..."
}
