package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
)

type DomainCache struct {
	mu      sync.RWMutex
	domains map[string]bool
	path    string
}

func NewDomainCache() (*DomainCache, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	cachePath := filepath.Join(configDir, "cache.json")

	cache := &DomainCache{
		domains: make(map[string]bool),
		path:    cachePath,
	}

	if err := cache.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}

	return cache, nil
}

func (c *DomainCache) load() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	data, err := os.ReadFile(c.path)
	if err != nil {
		return err
	}

	var domains []string
	if err := json.Unmarshal(data, &domains); err != nil {
		return err
	}

	for _, domain := range domains {
		c.domains[domain] = true
	}

	return nil
}

func (c *DomainCache) save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	domains := make([]string, 0, len(c.domains))
	for domain := range c.domains {
		domains = append(domains, domain)
	}

	data, err := json.Marshal(domains)
	if err != nil {
		return err
	}

	dir := filepath.Dir(c.path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	return os.WriteFile(c.path, data, 0644)
}

func (c *DomainCache) IsNew(domain string) bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return !c.domains[domain]
}

func (c *DomainCache) Add(domain string) error {
	c.mu.Lock()
	c.domains[domain] = true
	c.mu.Unlock()

	return c.save()
}
