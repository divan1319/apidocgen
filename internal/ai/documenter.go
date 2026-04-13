package ai

import "github.com/divan1319/apidocgen/pkg/models"

// Documenter genera documentación enriquecida para un endpoint vía API de IA.
type Documenter interface {
	DocumentEndpoint(ep models.Endpoint) (*models.EndpointDoc, error)
}
