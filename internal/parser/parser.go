package parser

import "github.com/divan1319/apidocgen/pkg/models"

type Parser interface {
	Parse(files []string) ([]models.Endpoint, error)
	Language() string
}
