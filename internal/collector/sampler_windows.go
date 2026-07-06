//go:build windows

package collector

import (
	"context"
	"strings"
	"syscall"

	gopsnet "github.com/shirou/gopsutil/v3/net"
	"golang.org/x/sys/windows"
)

type windowsSampler struct {
	resolver  *processResolver
	bandwidth *tcpBandwidthSampler
}

func newWindowsSampler() *windowsSampler {
	return &windowsSampler{
		resolver:  newProcessResolver(),
		bandwidth: newTCPBandwidthSampler(),
	}
}

func (s *windowsSampler) Sample(ctx context.Context, isAdmin bool) ([]ObservedConnection, error) {
	connections, err := gopsnet.ConnectionsWithContext(ctx, "all")
	if err != nil {
		return nil, err
	}

	observed := make([]ObservedConnection, 0, len(connections))
	activeKeys := make(map[ConnectionKey]struct{}, len(connections))
	activePIDs := make(map[int32]struct{}, len(connections))
	for _, conn := range connections {
		processName := s.resolver.nameForPID(ctx, conn.Pid)
		key := makeConnectionKey(conn, processName)
		activeKeys[key] = struct{}{}
		activePIDs[conn.Pid] = struct{}{}

		totals := TrafficTotals{}
		if isAdmin {
			if sampled, ok := s.bandwidth.sample(conn, key); ok {
				totals = sampled
			}
		}

		observed = append(observed, ObservedConnection{
			Key:         key,
			ProcessName: processName,
			PID:         conn.Pid,
			LocalIP:     conn.Laddr.IP,
			RemoteIP:    conn.Raddr.IP,
			LocalPort:   conn.Laddr.Port,
			RemotePort:  conn.Raddr.Port,
			Protocol:    protocolLabel(conn),
			State:       stateLabel(conn),
			Totals:      totals,
		})
	}

	s.bandwidth.prune(activeKeys)
	s.resolver.prune(activePIDs)

	return observed, nil
}

func makeConnectionKey(conn gopsnet.ConnectionStat, processName string) ConnectionKey {
	return ConnectionKey{
		PID:        conn.Pid,
		Process:    processName,
		Protocol:   protocolLabel(conn),
		LocalIP:    conn.Laddr.IP,
		LocalPort:  conn.Laddr.Port,
		RemoteIP:   conn.Raddr.IP,
		RemotePort: conn.Raddr.Port,
	}
}

func protocolLabel(conn gopsnet.ConnectionStat) string {
	version := "4"
	if conn.Family == windows.AF_INET6 {
		version = "6"
	}

	switch conn.Type {
	case syscall.SOCK_STREAM:
		return "TCP" + version
	case syscall.SOCK_DGRAM:
		return "UDP" + version
	default:
		return "IP" + version
	}
}

func stateLabel(conn gopsnet.ConnectionStat) string {
	status := strings.TrimSpace(conn.Status)
	if status == "" || status == "NONE" {
		if conn.Type == syscall.SOCK_DGRAM {
			return "LISTEN"
		}
		return "UNKNOWN"
	}
	return status
}
