package main

import (
	"context"
	"fmt"
	"net/http"
	"sync"

	log "github.com/sirupsen/logrus"
)

type Printer struct {
	configLoader *ConfigLoader
	msg          string
	mu           sync.RWMutex
}

func NewPrinter(cl *ConfigLoader) (*Printer, error) {
	p := &Printer{configLoader: cl}
	err := p.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Printer) Reload(ctx context.Context) error {
	config, err := p.configLoader.Get()
	if err != nil {
		return err
	}

	p.mu.Lock()
	p.msg = config.Printer.Message
	p.mu.Unlock()

	return nil
}

func (p *Printer) Print() {
	p.mu.RLock()
	defer p.mu.RUnlock()

	fmt.Println(p.msg)
}

type Curler struct {
	configLoader *ConfigLoader
	url          string
	method       string
	mu           sync.RWMutex
}

func NewCurler(cl *ConfigLoader) (*Curler, error) {
	c := &Curler{configLoader: cl}
	err := c.Reload(context.Background())
	if err != nil {
		return nil, err
	}

	return c, nil
}

func (c *Curler) Reload(ctx context.Context) error {
	config, err := c.configLoader.Get()
	if err != nil {
		return err
	}

	c.mu.Lock()
	c.method = config.Curler.HTTPMethod
	c.url = config.Curler.URL
	c.mu.Unlock()

	return nil
}

func (c *Curler) Curl() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	log.Infof("Curl %q to %q...", c.method, c.url)
	r, err := http.NewRequest(c.method, c.url, nil)
	if err != nil {
		return fmt.Errorf("could not create request: %w", err)
	}

	resp, err := http.DefaultClient.Do(r)
	if err != nil {
		return fmt.Errorf("http request error: %w", err)
	}

	log.Infof("%d: %+v", resp.StatusCode, resp.Header)
	return nil
}
