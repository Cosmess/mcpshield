package upstream

import (
	"fmt"
	"net/url"
	"strings"
	"sync"
	"time"
)

type Definition struct {
	ID              string
	Endpoint        string
	Enabled         bool
	Timeout         time.Duration
	ProtocolVersion string
}

type Registry struct {
	definitions map[string]Definition
	mu          sync.RWMutex
}

func NewRegistry(definitions []Definition) (*Registry, error) {
	entries := make(map[string]Definition, len(definitions))
	for _, definition := range definitions {
		if err := validate(definition); err != nil {
			return nil, err
		}
		if _, exists := entries[definition.ID]; exists {
			return nil, fmt.Errorf("upstream %q is duplicated", definition.ID)
		}
		entries[definition.ID] = definition
	}
	return &Registry{definitions: entries}, nil
}

func (registry *Registry) Resolve(id string) (Definition, bool) {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	definition, found := registry.definitions[id]
	if !found || !definition.Enabled {
		return Definition{}, false
	}
	return definition, true
}

func (registry *Registry) Definitions() []Definition {
	registry.mu.RLock()
	defer registry.mu.RUnlock()
	definitions := make([]Definition, 0, len(registry.definitions))
	for _, definition := range registry.definitions {
		if definition.Enabled {
			definitions = append(definitions, definition)
		}
	}
	return definitions
}

func validate(definition Definition) error {
	if strings.TrimSpace(definition.ID) == "" {
		return fmt.Errorf("upstream ID must not be empty")
	}
	parsed, err := url.Parse(definition.Endpoint)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return fmt.Errorf("upstream %q must have a valid HTTP endpoint", definition.ID)
	}
	if definition.Timeout <= 0 {
		return fmt.Errorf("upstream %q timeout must be positive", definition.ID)
	}
	if definition.ProtocolVersion == "" {
		return fmt.Errorf("upstream %q protocol version must not be empty", definition.ID)
	}
	return nil
}
