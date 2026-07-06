package ui

import (
	"context"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	rdnsTimeout       = 2 * time.Second
	rdnsMaxConcurrent = 8
	rdnsMaxEntries    = 4096
)

// rdnsCache resolves remote IPs to hostnames in the background. Lookups are
// only triggered for addresses the UI actually renders (expanded rows), so a
// busy machine does not fan out hundreds of DNS queries.
type rdnsCache struct {
	mu      sync.Mutex
	names   map[string]string
	pending map[string]struct{}
	sem     chan struct{}
	resolve func(ctx context.Context, ip string) ([]string, error)
}

func newRDNSCache() *rdnsCache {
	return &rdnsCache{
		names:   map[string]string{},
		pending: map[string]struct{}{},
		sem:     make(chan struct{}, rdnsMaxConcurrent),
		resolve: net.DefaultResolver.LookupAddr,
	}
}

// lookup returns the cached hostname for ip ("" while unknown) and schedules
// an asynchronous reverse lookup the first time an address is seen. Results
// show up on a later render; the UI refreshes every second anyway.
func (c *rdnsCache) lookup(ip string) string {
	if c == nil {
		return ""
	}
	switch ip {
	case "", "*", "0.0.0.0", "::":
		return ""
	}

	c.mu.Lock()
	if name, ok := c.names[ip]; ok {
		c.mu.Unlock()
		return name
	}
	if _, inflight := c.pending[ip]; inflight {
		c.mu.Unlock()
		return ""
	}
	c.pending[ip] = struct{}{}
	c.mu.Unlock()

	go c.resolveAsync(ip)
	return ""
}

func (c *rdnsCache) resolveAsync(ip string) {
	c.sem <- struct{}{}
	defer func() { <-c.sem }()

	ctx, cancel := context.WithTimeout(context.Background(), rdnsTimeout)
	defer cancel()

	name := ""
	if names, err := c.resolve(ctx, ip); err == nil && len(names) > 0 {
		name = strings.TrimSuffix(names[0], ".")
	}

	c.mu.Lock()
	if len(c.names) >= rdnsMaxEntries {
		c.names = make(map[string]string, rdnsMaxEntries/4)
	}
	// Failures are cached as "" so unresolvable addresses are not retried on
	// every render.
	c.names[ip] = name
	delete(c.pending, ip)
	c.mu.Unlock()
}
