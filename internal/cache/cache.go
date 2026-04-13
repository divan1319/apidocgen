package cache

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"

	"github.com/divan1319/apidocgen/pkg/models"
)

const currentVersion = 2

type Entry struct {
	Doc        models.EndpointDoc `json:"doc"`
	SourceHash string             `json:"source_hash"`
}

type Cache struct {
	Version int              `json:"version"`
	Entries map[string]Entry `json:"entries"`
	path    string
	scope   string // prefijo de clave en memoria (proveedor:modelo); no persiste en JSON
}

// Load lee el archivo de caché. keyScope debe ser estable por proveedor+modelo (p. ej. ai.Config.CacheScope).
// Si el archivo era de versión 1, las entradas se descartan para no mezclar resultados de otro modelo.
func Load(path, keyScope string) (*Cache, error) {
	c := &Cache{
		Version: currentVersion,
		Entries: make(map[string]Entry),
		path:    path,
		scope:   keyScope,
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

	if c.Version < currentVersion {
		c.Entries = make(map[string]Entry)
	}
	c.Version = currentVersion
	c.path = path
	c.scope = keyScope

	if c.Entries == nil {
		c.Entries = make(map[string]Entry)
	}
	return c, nil
}

func (c *Cache) entryKey(ep models.Endpoint) string {
	if c.scope == "" {
		return ep.Method + ":" + ep.URI
	}
	return c.scope + "|" + ep.Method + ":" + ep.URI
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

func sourceHash(ep models.Endpoint) string {
	h := sha256.Sum256([]byte(ep.RawSource))
	return fmt.Sprintf("%x", h[:8])
}

func (c *Cache) Get(ep models.Endpoint) (*models.EndpointDoc, bool) {
	entry, ok := c.Entries[c.entryKey(ep)]
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
	c.Entries[c.entryKey(ep)] = Entry{
		Doc:        doc,
		SourceHash: sourceHash(ep),
	}
}

func (c *Cache) Len() int {
	return len(c.Entries)
}
