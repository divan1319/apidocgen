package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

const anthropicAPI = "https://api.anthropic.com/v1/messages"
const model = "claude-sonnet-4-6"

type Client struct {
	apiKey string
}

func New(apiKey string) *Client {
	return &Client{apiKey: apiKey}
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
func (c *Client) DocumentEndpoint(ep models.Endpoint) (*models.EndpointDoc, error) {
	prompt := buildPrompt(ep)

	reqBody, _ := json.Marshal(request{
		Model:     model,
		MaxTokens: 4096,
		System:    systemPrompt,
		Messages:  []message{{Role: "user", Content: prompt}},
	})

	req, _ := http.NewRequest("POST", anthropicAPI, bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", c.apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling Claude API: %w", err)
	}
	defer resp.Body.Close()

	var apiResp response
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("decoding response: %w", err)
	}

	if len(apiResp.Content) == 0 {
		return nil, fmt.Errorf("empty response from Claude")
	}

	return parseDocResponse(ep, apiResp.Content[0].Text)
}

const systemPrompt = `You are an API documentation expert. Analyze the provided Laravel endpoint information and generate structured documentation.

Always respond with valid JSON only, no markdown, no explanation. Use this exact structure:
{
  "summary": "short one-liner describing what this endpoint does",
  "description": "fuller explanation of the endpoint behavior, business logic, side effects",
  "parameters": [
    {
      "name": "field_name",
      "type": "string|integer|boolean|array|file",
      "required": true,
      "rules": "raw validation rules if available",
      "description": "what this parameter does"
    }
  ],
  "responses": [
    {
      "code": 200,
      "description": "what this response means",
      "body": "{\"key\": \"example value\"}"
    }
  ],
  "example": {
    "headers": {"Authorization": "Bearer {token}", "Accept": "application/json"},
    "body": "{\"field\": \"value\"}"
  }
}`

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
		sb.WriteString("\nController source code:\n```php\n")
		sb.WriteString(ep.RawSource)
		sb.WriteString("\n```\n")
	}

	sb.WriteString("\nGenerate complete API documentation for this endpoint.")
	return sb.String()
}

func parseDocResponse(ep models.Endpoint, text string) (*models.EndpointDoc, error) {
	// Strip any accidental markdown fences
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
