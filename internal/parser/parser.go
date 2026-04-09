package parser

import "github.com/divan1319/apidocgen/pkg/models"

// Parser is the contract every language-specific parser must implement.
// Adding support for a new language = implementing this interface.
type Parser interface {
	// Parse receives a list of route files and returns all discovered endpoints.
	Parse(files []string) ([]models.Endpoint, error)

	// Language returns a human-readable identifier ("laravel", "dotnet", "nestjs"...)
	Language() string
}
