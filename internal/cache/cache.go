package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/divan1319/apidocgen/pkg/models"
)

const currentVersion = 1

// Entry wraps an EndpointDoc with a source hash so stale entries can be detected.
type Entry struct {
	Doc        models.EndpointDoc `json:"doc"`
	SourceHash string             `json:"source_hash"` // SHA-256 of RawSource
}

// Cache holds documented endpoints indexed by "METHOD:URI".
type Cache struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
	path    string
}

// Load reads the cache file from disk. If the file does not exist, an empty cache is returned.
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

// Save writes the cache to disk atomically.
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

// key returns the unique identifier for an endpoint.
func key(ep models.Endpoint) string {
	return ep.Method + ":" + ep.URI
}

// sourceHash computes a short hash of the raw source to detect changes.
func sourceHash(ep models.Endpoint) string {
	h := sha256.Sum256([]byte(ep.RawSource))
	return fmt.Sprintf("%x", h[:8])
}

// Get returns the cached EndpointDoc if it exists and the source has not changed.
// The second return value is true on a valid cache hit.
func (c *Cache) Get(ep models.Endpoint) (*models.EndpointDoc, bool) {
	entry, ok := c.Entries[key(ep)]
	if !ok {
		return nil, false
	}
	// If the source changed, treat as a miss so Claude re-documents it.
	if entry.SourceHash != sourceHash(ep) {
		return nil, false
	}
	doc := entry.Doc
	return &doc, true
}

// Set stores the documentation for an endpoint in the cache.
func (c *Cache) Set(ep models.Endpoint, doc models.EndpointDoc) {
	c.Entries[key(ep)] = Entry{
		Doc:        doc,
		SourceHash: sourceHash(ep),
	}
}

// Len returns the number of entries in the cache.
func (c *Cache) Len() int {
	return len(c.Entries)
}
