package ui

import (
	"context"
	"errors"
	"testing"
	"time"
)

func waitForCached(t *testing.T, cache *rdnsCache, ip string, want string) {
	t.Helper()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		cache.mu.Lock()
		name, ok := cache.names[ip]
		cache.mu.Unlock()
		if ok {
			if name != want {
				t.Fatalf("expected cached name %q for %s, got %q", want, ip, name)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("lookup for %s never completed", ip)
}

func TestRDNSCacheResolvesAsynchronouslyAndCaches(t *testing.T) {
	cache := newRDNSCache()
	cache.resolve = func(_ context.Context, ip string) ([]string, error) {
		return []string{"one.one.one.one."}, nil
	}

	if got := cache.lookup("1.1.1.1"); got != "" {
		t.Fatalf("expected first lookup to return empty while resolving, got %q", got)
	}
	waitForCached(t, cache, "1.1.1.1", "one.one.one.one")

	if got, want := cache.lookup("1.1.1.1"), "one.one.one.one"; got != want {
		t.Fatalf("expected cached hostname %q, got %q", want, got)
	}
}

func TestRDNSCacheCachesFailuresAsNegative(t *testing.T) {
	calls := 0
	cache := newRDNSCache()
	cache.resolve = func(_ context.Context, ip string) ([]string, error) {
		calls++
		return nil, errors.New("nxdomain")
	}

	cache.lookup("203.0.113.7")
	waitForCached(t, cache, "203.0.113.7", "")

	cache.lookup("203.0.113.7")
	time.Sleep(20 * time.Millisecond)
	if calls != 1 {
		t.Fatalf("expected failed lookup to be cached and not retried, resolver ran %d times", calls)
	}
}

func TestRDNSCacheSkipsUnresolvableAddressesAndNilReceiver(t *testing.T) {
	var nilCache *rdnsCache
	if got := nilCache.lookup("1.1.1.1"); got != "" {
		t.Fatalf("expected nil cache lookup to return empty, got %q", got)
	}

	cache := newRDNSCache()
	cache.resolve = func(_ context.Context, ip string) ([]string, error) {
		t.Fatalf("resolver must not run for %q", ip)
		return nil, nil
	}
	for _, ip := range []string{"", "*", "0.0.0.0", "::"} {
		if got := cache.lookup(ip); got != "" {
			t.Fatalf("expected no lookup for %q, got %q", ip, got)
		}
	}
	time.Sleep(20 * time.Millisecond)
}
