package ai

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/divan1319/apidocgen/pkg/models"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"
const model = "claude-sonnet-4-6"

//go:embed prompts.json
var promptsData []byte

type prompts struct {
	EN string `json:"en"`
	ES string `json:"es"`
}

func loadPrompt(lang string) (string, error) {
	var p prompts
	if err := json.Unmarshal(promptsData, &p); err != nil {
		return "", fmt.Errorf("loading prompts.json: %w", err)
	}
	switch strings.ToLower(lang) {
	case "es", "español", "spanish":
		return p.ES, nil
	default:
		return p.EN, nil
	}
}

type Client struct {
	apiKey       string
	systemPrompt string
}

func New(apiKey, lang string) (*Client, error) {
	prompt, err := loadPrompt(lang)
	if err != nil {
		return nil, err
	}
	return &Client{apiKey: apiKey, systemPrompt: prompt}, nil
}

type message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type request struct {
	Model     string    `json:"model"`
	MaxTokens int       `json:"max_tokens"`
	System    string    `json:"system"`
	Messages  []message `json:"messages"`
}

type response struct {
	Content []struct {
		Text string `json:"text"`
	} `json:"content"`
}

// DocumentEndpoint sends an endpoint to Claude and returns enriched documentation.
// Retries up to 4 times with exponential backoff on rate limit (429) or server (5xx) responses.
func (c *Client) DocumentEndpoint(ep models.Endpoint) (*models.EndpointDoc, error) {
	var lastErr error
	wait := 2 * time.Second

	for attempt := 0; attempt < 4; attempt++ {
		if attempt > 0 {
			fmt.Fprintf(os.Stderr, "    rate limited, retrying in %s...\n", wait)
			time.Sleep(wait)
			wait *= 2 // 2s → 4s → 8s
		}

		doc, err, retry := c.attempt(ep)
		if !retry {
			return doc, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("exceeded retries: %w", lastErr)
}

func (c *Client) attempt(ep models.Endpoint) (doc *models.EndpointDoc, err error, retry bool) {
	reqBody, _ := json.Marshal(request{
		Model:     model,
		MaxTokens: 4096,
		System:    c.systemPrompt,
		Messages:  []message{{Role: "user", Content: buildPrompt(ep)}},
	})

	req, _ := http.NewRequest("POST", anthropicAPI, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Claude API: %w", err), false
	}
	defer resp.Body.Close()

	// Rate limited or server error — worth retrying with backoff
	if resp.StatusCode == 429 {
		return nil, fmt.Errorf("rate limited (429)"), true
	}
	if resp.StatusCode >= 500 {
		return nil, fmt.Errorf("server error (%d)", resp.StatusCode), true
	}

	var apiResp response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err), false
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude"), false
	}

	result, err := parseDocResponse(ep, apiResp.Content[0].Text)
	return result, err, false
}

func buildPrompt(ep models.Endpoint) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("Method: %s\nURI: %s\n", ep.Method, ep.URI))

	if ep.Controller != "" {
		sb.WriteString(fmt.Sprintf("Controller: %s@%s\n", ep.Controller, ep.Action))
	}
	if len(ep.Middleware) > 0 {
		sb.WriteString(fmt.Sprintf("Middleware: %s\n", strings.Join(ep.Middleware, ", ")))
	}

	if len(ep.StaticMeta.RequestParams) > 0 {
		sb.WriteString("\nStatically extracted parameters:\n")
		for _, p := range ep.StaticMeta.RequestParams {
			sb.WriteString(fmt.Sprintf("  - %s (%s, required=%v): %s\n", p.Name, p.Type, p.Required, p.Rules))
		}
	}

	if ep.RawSource != "" {
		lang := ep.Language
		if lang == "" {
			lang = "php"
		}
		sb.WriteString(fmt.Sprintf("\nController source code:\n```%s\n", lang))
		sb.WriteString(ep.RawSource)
		sb.WriteString("\n```\n")
	}

	sb.WriteString("\nGenerate complete API documentation for this endpoint.")
	return sb.String()
}

func parseDocResponse(ep models.Endpoint, text string) (*models.EndpointDoc, error) {
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)

	var raw struct {
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Parameters  []struct {
			Name        string `json:"name"`
			Type        string `json:"type"`
			Required    bool   `json:"required"`
			Rules       string `json:"rules"`
			Description string `json:"description"`
		} `json:"parameters"`
		Responses []struct {
			Code        int    `json:"code"`
			Description string `json:"description"`
			Body        string `json:"body"`
		} `json:"responses"`
		Example struct {
			Headers map[string]string `json:"headers"`
			Body    string            `json:"body"`
		} `json:"example"`
	}

	if err := json.Unmarshal([]byte(text), &raw); err != nil {
		return nil, fmt.Errorf("parsing Claude JSON response: %w\nRaw: %s", err, text)
	}

	doc := &models.EndpointDoc{
		Endpoint:    ep,
		Summary:     raw.Summary,
		Description: raw.Description,
		Example: models.RequestExample{
			Headers: raw.Example.Headers,
			Body:    raw.Example.Body,
		},
	}

	for _, p := range raw.Parameters {
		doc.Parameters = append(doc.Parameters, models.ParamDoc{
			Param: models.Param{
				Name:     p.Name,
				Type:     p.Type,
				Required: p.Required,
				Rules:    p.Rules,
			},
			Description: p.Description,
		})
	}

	for _, r := range raw.Responses {
		doc.Responses = append(doc.Responses, models.ResponseDoc{
			Code:        r.Code,
			Description: r.Description,
			Body:        r.Body,
		})
	}

	return doc, nil
}
