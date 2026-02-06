package vcs

import (
	"fmt"
	"sync"
)

var (
	registry = make(map[string]Provider)
	mu       sync.RWMutex
)

// RegisterProvider registers a VCS provider implementation.
// Should be called during application initialization (e.g., in init() or main()).
func RegisterProvider(provider Provider) {
	mu.Lock()
	defer mu.Unlock()
	registry[provider.Name()] = provider
}

// GetProvider returns a registered provider by name.
// Returns an error if the provider is not registered.
func GetProvider(name string) (Provider, error) {
	mu.RLock()
	defer mu.RUnlock()

	provider, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("unknown VCS provider: %s", name)
	}
	return provider, nil
}

// ListProviders returns the names of all registered providers.
func ListProviders() []string {
	mu.RLock()
	defer mu.RUnlock()

	names := make([]string, 0, len(registry))
	for name := range registry {
		names = append(names, name)
	}
	return names
}
