package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/divan1319/apidocgen/pkg/models"
)

const currentVersion = 1

type Entry struct {
	Doc        models.EndpointDoc `json:"doc"`
	SourceHash string             `json:"source_hash"`
}

type Cache struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
	path    string
}

func Load(path string) (*Cache, error) {
	c := &Cache{
		Version: currentVersion,
		Entries: make(map[string]Entry),
		path:    path,
	}

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return c, nil
	}
	if err != nil {
		return nil, fmt.Errorf("reading cache %s: %w", path, err)
	}

	if err := json.Unmarshal(data, c); err != nil {
		return nil, fmt.Errorf("parsing cache %s: %w", path, err)
	}
	c.path = path

	if c.Entries == nil {
		c.Entries = make(map[string]Entry)
	}
	return c, nil
}

func (c *Cache) Save() error {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling cache: %w", err)
	}

	tmp := c.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0644); err != nil {
		return fmt.Errorf("writing cache temp file: %w", err)
	}
	if err := os.Rename(tmp, c.path); err != nil {
		return fmt.Errorf("renaming cache file: %w", err)
	}
	return nil
}

func key(ep models.Endpoint) string {
	return ep.Method + ":" + ep.URI
}

func sourceHash(ep models.Endpoint) string {
	h := sha256.Sum256([]byte(ep.RawSource))
	return fmt.Sprintf("%x", h[:8])
}

func (c *Cache) Get(ep models.Endpoint) (*models.EndpointDoc, bool) {
	entry, ok := c.Entries[key(ep)]
	if !ok {
		return nil, false
	}
	if entry.SourceHash != sourceHash(ep) {
		return nil, false
	}
	doc := entry.Doc
	return &doc, true
}

func (c *Cache) Set(ep models.Endpoint, doc models.EndpointDoc) {
	c.Entries[key(ep)] = Entry{
		Doc:        doc,
		SourceHash: sourceHash(ep),
	}
}

func (c *Cache) Len() int {
	return len(c.Entries)
}
