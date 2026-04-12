package parser

import (
	"fmt"
	"sync"

	"github.com/divan1319/apidocgen/pkg/models"
)

type Parser interface {
	Language() string
	ResolveIncludes(files []string) ([]string, error)
	ParseSections(files []string) ([]models.RouteSection, error)
}

type Factory func(projectRoot string) Parser

var (
	mu        sync.RWMutex
	factories = make(map[string]Factory)
)

func Register(name string, f Factory) {
	mu.Lock()
	defer mu.Unlock()
	if _, dup := factories[name]; dup {
		panic(fmt.Sprintf("parser: Register called twice for %q", name))
	}
	factories[name] = f
}

func Get(name, projectRoot string) (Parser, error) {
	mu.RLock()
	f, ok := factories[name]
	mu.RUnlock()
	if !ok {
		return nil, fmt.Errorf("unknown language %q (registered: %v)", name, Names())
	}
	return f(projectRoot), nil
}

func Names() []string {
	mu.RLock()
	defer mu.RUnlock()
	names := make([]string, 0, len(factories))
	for n := range factories {
		names = append(names, n)
	}
	return names
}
