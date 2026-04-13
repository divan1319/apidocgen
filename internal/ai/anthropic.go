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

const anthropicMessagesURL = "https://api.anthropic.com/v1/messages"

type anthropicClient struct {
	apiKey       string
	systemPrompt string
	model        string
}

func newAnthropicClient(cfg Config) (*anthropicClient, error) {
	prompt, err := loadPrompt(cfg.DocLang)
	if err != nil {
		return nil, err
	}
	return &anthropicClient{
		apiKey:       cfg.APIKey,
		systemPrompt: prompt,
		model:        cfg.ResolvedModel(),
	}, nil
}

type anthropicMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system"`
	Messages  []anthropicMessage `json:"messages"`
}

type anthropicResponse struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// DocumentEndpoint envía el endpoint a Anthropic Messages API.
func (c *anthropicClient) DocumentEndpoint(ep models.Endpoint) (*models.EndpointDoc, error) {
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

func (c *anthropicClient) attempt(ep models.Endpoint) (doc *models.EndpointDoc, err error, retry bool) {
	reqBody, _ := json.Marshal(anthropicRequest{
		Model:     c.model,
		MaxTokens: 4096,
		System:    c.systemPrompt,
		Messages:  []anthropicMessage{{Role: "user", Content: buildPrompt(ep)}},
	})

	req, _ := http.NewRequest("POST", anthropicMessagesURL, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("llamando a Anthropic API: %w", err), false
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
		s := strings.TrimSpace(string(bodyBytes))
		if len(s) > 512 {
			s = s[:512] + "..."
		}
		return nil, fmt.Errorf("Anthropic API error (%d): %s", resp.StatusCode, s), false
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(bodyBytes, &apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err), false
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Anthropic"), false
	}

	result, err := parseDocResponse(ep, apiResp.Content[0].Text)
	return result, err, false
}
