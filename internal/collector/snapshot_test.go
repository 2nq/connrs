package collector

import (
	"testing"
	"time"
)

func TestBuildSnapshotGroupsProcessesAndDiffsBandwidth(t *testing.T) {
	now := time.Date(2026, time.April, 15, 17, 0, 0, 0, time.UTC)
	last := now.Add(-time.Second)

	discordTLS := ConnectionKey{
		PID:        4242,
		Process:    "discord.exe",
		Protocol:   "TCP4",
		LocalIP:    "10.0.0.5",
		LocalPort:  55000,
		RemoteIP:   "162.159.128.233",
		RemotePort: 443,
	}
	discordVoice := ConnectionKey{
		PID:        4242,
		Process:    "discord.exe",
		Protocol:   "TCP4",
		LocalIP:    "10.0.0.5",
		LocalPort:  55001,
		RemoteIP:   "66.22.212.5",
		RemotePort: 443,
	}
	chromeUDP := ConnectionKey{
		PID:        9001,
		Process:    "chrome.exe",
		Protocol:   "UDP4",
		LocalIP:    "10.0.0.5",
		LocalPort:  56000,
		RemoteIP:   "",
		RemotePort: 0,
	}

	observed := []ObservedConnection{
		{
			Key:         discordTLS,
			ProcessName: "discord.exe",
			PID:         4242,
			LocalIP:     "10.0.0.5",
			RemoteIP:    "162.159.128.233",
			LocalPort:   55000,
			RemotePort:  443,
			Protocol:    "TCP4",
			State:       "ESTABLISHED",
			Totals: TrafficTotals{
				Sent:      3_000,
				Received:  9_000,
				Available: true,
			},
		},
		{
			Key:         discordVoice,
			ProcessName: "discord.exe",
			PID:         4242,
			LocalIP:     "10.0.0.5",
			RemoteIP:    "66.22.212.5",
			LocalPort:   55001,
			RemotePort:  443,
			Protocol:    "TCP4",
			State:       "ESTABLISHED",
			Totals: TrafficTotals{
				Sent:      1_000,
				Received:  2_000,
				Available: true,
			},
		},
		{
			Key:         chromeUDP,
			ProcessName: "chrome.exe",
			PID:         9001,
			LocalIP:     "10.0.0.5",
			RemoteIP:    "",
			LocalPort:   56000,
			RemotePort:  0,
			Protocol:    "UDP4",
			State:       "NONE",
			Totals: TrafficTotals{
				Sent:      500,
				Received:  500,
				Available: false,
			},
		},
	}

	previous := map[ConnectionKey]TrafficTotals{
		discordTLS: {
			Sent:      1_000,
			Received:  3_000,
			Available: true,
		},
		discordVoice: {
			Sent:      500,
			Received:  1_000,
			Available: true,
		},
	}

	snapshot, next := buildSnapshot(observed, previous, last, now, true)

	if got, want := len(snapshot.Processes), 2; got != want {
		t.Fatalf("expected %d processes, got %d", want, got)
	}

	top := snapshot.Processes[0]
	if top.Name != "discord.exe" || top.PID != 4242 {
		t.Fatalf("expected discord.exe/4242 to be first, got %s/%d", top.Name, top.PID)
	}
	if got, want := top.SentBps, uint64(2_500); got != want {
		t.Fatalf("expected process sent bps %d, got %d", want, got)
	}
	if got, want := top.ReceivedBps, uint64(7_000); got != want {
		t.Fatalf("expected process received bps %d, got %d", want, got)
	}
	if got, want := len(top.Connections), 2; got != want {
		t.Fatalf("expected %d discord connections, got %d", want, got)
	}
	if got, want := top.Connections[0].SentBps, uint64(2_000); got != want {
		t.Fatalf("expected top connection sent bps %d, got %d", want, got)
	}
	if got, want := top.Connections[0].ReceivedBps, uint64(6_000); got != want {
		t.Fatalf("expected top connection received bps %d, got %d", want, got)
	}
	if top.Connections[0].Key != discordTLS {
		t.Fatalf("expected discord TLS connection to sort first, got %+v", top.Connections[0].Key)
	}

	second := snapshot.Processes[1]
	if second.Name != "chrome.exe" || second.PID != 9001 {
		t.Fatalf("expected chrome.exe/9001 second, got %s/%d", second.Name, second.PID)
	}
	if second.Connections[0].BandwidthKnown {
		t.Fatalf("expected UDP connection to be marked as bandwidth-unknown")
	}
	if second.Connections[0].SentBps != 0 || second.Connections[0].ReceivedBps != 0 {
		t.Fatalf("expected new/unknown connection to have zero throughput, got tx=%d rx=%d",
			second.Connections[0].SentBps,
			second.Connections[0].ReceivedBps,
		)
	}

	if got, want := snapshot.TotalConnections, 3; got != want {
		t.Fatalf("expected total connections %d, got %d", want, got)
	}
	if got, want := snapshot.BandwidthTracked, 2; got != want {
		t.Fatalf("expected tracked connections %d, got %d", want, got)
	}

	if got, want := len(next), 3; got != want {
		t.Fatalf("expected %d saved counters, got %d", want, got)
	}
	if got, want := next[discordTLS].Received, uint64(9_000); got != want {
		t.Fatalf("expected saved receive counter %d, got %d", want, got)
	}
}

func TestBuildSnapshotClampsCounterResetsToZero(t *testing.T) {
	now := time.Date(2026, time.April, 15, 17, 0, 1, 0, time.UTC)
	last := now.Add(-time.Second)

	key := ConnectionKey{
		PID:        7,
		Process:    "edge.exe",
		Protocol:   "TCP4",
		LocalIP:    "10.0.0.5",
		LocalPort:  57000,
		RemoteIP:   "20.42.65.89",
		RemotePort: 443,
	}

	observed := []ObservedConnection{
		{
			Key:         key,
			ProcessName: "edge.exe",
			PID:         7,
			LocalIP:     "10.0.0.5",
			RemoteIP:    "20.42.65.89",
			LocalPort:   57000,
			RemotePort:  443,
			Protocol:    "TCP4",
			State:       "ESTABLISHED",
			Totals: TrafficTotals{
				Sent:      50,
				Received:  75,
				Available: true,
			},
		},
	}

	previous := map[ConnectionKey]TrafficTotals{
		key: {
			Sent:      100,
			Received:  200,
			Available: true,
		},
	}

	snapshot, _ := buildSnapshot(observed, previous, last, now, false)

	conn := snapshot.Processes[0].Connections[0]
	if conn.SentBps != 0 || conn.ReceivedBps != 0 {
		t.Fatalf("expected reset counters to clamp to zero, got tx=%d rx=%d", conn.SentBps, conn.ReceivedBps)
	}
}

func TestBuildSnapshotNormalizesBlankProcessNames(t *testing.T) {
	now := time.Date(2026, time.April, 15, 17, 0, 2, 0, time.UTC)

	observed := []ObservedConnection{
		{
			Key: ConnectionKey{
				PID:        5150,
				Process:    "",
				Protocol:   "TCP4",
				LocalIP:    "127.0.0.1",
				LocalPort:  8080,
				RemoteIP:   "127.0.0.1",
				RemotePort: 62000,
			},
			ProcessName: "",
			PID:         5150,
			LocalIP:     "127.0.0.1",
			RemoteIP:    "127.0.0.1",
			LocalPort:   8080,
			RemotePort:  62000,
			Protocol:    "TCP4",
			State:       "ESTABLISHED",
		},
	}

	snapshot, _ := buildSnapshot(observed, nil, time.Time{}, now, false)

	if got, want := snapshot.Processes[0].Name, "PID 5150"; got != want {
		t.Fatalf("expected fallback process label %q, got %q", want, got)
	}
}
