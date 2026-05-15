package collector

import (
	"context"
	"time"
)

type ConnectionKey struct {
	PID        int32
	Process    string
	Protocol   string
	LocalIP    string
	LocalPort  uint32
	RemoteIP   string
	RemotePort uint32
}

type TrafficTotals struct {
	Sent      uint64
	Received  uint64
	Available bool
}

type ObservedConnection struct {
	Key         ConnectionKey
	ProcessName string
	PID         int32
	LocalIP     string
	RemoteIP    string
	LocalPort   uint32
	RemotePort  uint32
	Protocol    string
	State       string
	Totals      TrafficTotals
}

type ConnectionSnapshot struct {
	Key            ConnectionKey
	LocalIP        string
	RemoteIP       string
	LocalPort      uint32
	RemotePort     uint32
	Protocol       string
	State          string
	Totals         TrafficTotals
	SentBps        uint64
	ReceivedBps    uint64
	BandwidthKnown bool
}

type ProcessSnapshot struct {
	Name             string
	PID              int32
	Connections      []ConnectionSnapshot
	SentBps          uint64
	ReceivedBps      uint64
	BandwidthTracked int
}

type Snapshot struct {
	CapturedAt       time.Time
	IsAdmin          bool
	Processes        []ProcessSnapshot
	TotalConnections int
	BandwidthTracked int
}

type Poller interface {
	Poll(ctx context.Context) (Snapshot, error)
}
