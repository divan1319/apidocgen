package ai

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/divan1319/apidocgen/pkg/models"
)

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
		return nil, fmt.Errorf("parsing JSON de la respuesta del modelo: %w\nRaw: %s", err, text)
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
