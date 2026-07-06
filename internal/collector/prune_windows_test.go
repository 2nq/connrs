//go:build windows

package collector

import "testing"

func TestBandwidthSamplerPruneDropsDepartedConnections(t *testing.T) {
	stale := ConnectionKey{PID: 1, LocalIP: "10.0.0.5", LocalPort: 50000, RemoteIP: "1.1.1.1", RemotePort: 443}
	alive := ConnectionKey{PID: 2, LocalIP: "10.0.0.5", LocalPort: 50001, RemoteIP: "1.0.0.1", RemotePort: 443}

	sampler := newTCPBandwidthSampler()
	sampler.enabled[stale] = struct{}{}
	sampler.enabled[alive] = struct{}{}

	sampler.prune(map[ConnectionKey]struct{}{alive: {}})

	if _, ok := sampler.enabled[stale]; ok {
		t.Fatalf("expected departed connection to be forgotten so a reused tuple re-enables estats")
	}
	if _, ok := sampler.enabled[alive]; !ok {
		t.Fatalf("expected live connection to stay enabled")
	}
}

func TestProcessResolverPruneDropsDepartedPIDsButKeepsKernelSeeds(t *testing.T) {
	resolver := newProcessResolver()
	resolver.names[1234] = "old.exe"
	resolver.names[5678] = "live.exe"

	resolver.prune(map[int32]struct{}{5678: {}})

	if _, ok := resolver.names[1234]; ok {
		t.Fatalf("expected stale PID cache entry to be pruned (PIDs get reused)")
	}
	if got := resolver.names[5678]; got != "live.exe" {
		t.Fatalf("expected live PID cache entry to survive, got %q", got)
	}
	if got := resolver.names[0]; got != "System Idle Process" {
		t.Fatalf("expected kernel seed for PID 0 to survive, got %q", got)
	}
	if got := resolver.names[4]; got != "System" {
		t.Fatalf("expected kernel seed for PID 4 to survive, got %q", got)
	}
}
