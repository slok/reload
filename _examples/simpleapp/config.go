package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type Config struct {
	Printer struct {
		Message string `json:"message"`
	} `json:"printer"`
	Curler struct {
		HTTPMethod string `json:"httpMethod"`
		URL        string `json:"url"`
	} `json:"curler"`
}

type ConfigLoader struct {
	path   string
	config Config
	mu     sync.RWMutex
}

// NewConfigLoader returns a new ConfigLoader.
func NewConfigLoader(path string) (*ConfigLoader, error) {
	c := &ConfigLoader{path: path}
	err := c.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *ConfigLoader) Reload(ctx context.Context) error {
	data, err := os.ReadFile(c.path)
	if err != nil {
		return fmt.Errorf("could no load config file %s: %w", c.path, err)
	}

	config := Config{}
	err = json.Unmarshal(data, &config)
	if err != nil {
		return fmt.Errorf("could not unmarshal JSON config: %w", err)
	}

	// Set config.
	c.mu.Lock()
	c.config = config
	c.mu.Unlock()

	return nil
}
func (c *ConfigLoader) Get() (*Config, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return &c.config, nil
}
