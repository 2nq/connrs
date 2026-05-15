package collector

import (
	"cmp"
	"slices"
	"strconv"
	"time"
)

func buildSnapshot(
	observed []ObservedConnection,
	previous map[ConnectionKey]TrafficTotals,
	last time.Time,
	now time.Time,
	isAdmin bool,
) (Snapshot, map[ConnectionKey]TrafficTotals) {
	seconds := now.Sub(last).Seconds()
	if last.IsZero() || seconds <= 0 {
		seconds = 0
	}

	next := make(map[ConnectionKey]TrafficTotals, len(observed))
	processes := make(map[string]*ProcessSnapshot, len(observed))

	for _, conn := range observed {
		next[conn.Key] = conn.Totals

		processName := normalizeProcessName(conn.PID, conn.ProcessName, conn.Key.Process)
		processKey := processName + "#" + itoa32(conn.PID)
		proc, ok := processes[processKey]
		if !ok {
			proc = &ProcessSnapshot{
				Name:        processName,
				PID:         conn.PID,
				Connections: make([]ConnectionSnapshot, 0, 1),
			}
			processes[processKey] = proc
		}

		sentBps, recvBps, known := diffTraffic(previous[conn.Key], conn.Totals, seconds)
		if conn.Totals.Available {
			proc.SentBps += sentBps
			proc.ReceivedBps += recvBps
			proc.BandwidthTracked++
		}

		proc.Connections = append(proc.Connections, ConnectionSnapshot{
			Key:            conn.Key,
			LocalIP:        conn.LocalIP,
			RemoteIP:       conn.RemoteIP,
			LocalPort:      conn.LocalPort,
			RemotePort:     conn.RemotePort,
			Protocol:       conn.Protocol,
			State:          conn.State,
			Totals:         conn.Totals,
			SentBps:        sentBps,
			ReceivedBps:    recvBps,
			BandwidthKnown: known,
		})
	}

	processList := make([]ProcessSnapshot, 0, len(processes))
	bandwidthTracked := 0
	for _, process := range processes {
		slices.SortStableFunc(process.Connections, func(left, right ConnectionSnapshot) int {
			leftTotal := left.SentBps + left.ReceivedBps
			rightTotal := right.SentBps + right.ReceivedBps
			if leftTotal != rightTotal {
				return cmp.Compare(rightTotal, leftTotal)
			}
			if left.Protocol != right.Protocol {
				return cmp.Compare(left.Protocol, right.Protocol)
			}
			if left.RemoteIP != right.RemoteIP {
				return cmp.Compare(left.RemoteIP, right.RemoteIP)
			}
			return cmp.Compare(left.RemotePort, right.RemotePort)
		})
		bandwidthTracked += process.BandwidthTracked
		processList = append(processList, *process)
	}

	slices.SortStableFunc(processList, func(left, right ProcessSnapshot) int {
		leftTotal := left.SentBps + left.ReceivedBps
		rightTotal := right.SentBps + right.ReceivedBps
		if leftTotal != rightTotal {
			return cmp.Compare(rightTotal, leftTotal)
		}
		if left.Name != right.Name {
			return cmp.Compare(left.Name, right.Name)
		}
		return cmp.Compare(left.PID, right.PID)
	})

	return Snapshot{
		CapturedAt:       now,
		IsAdmin:          isAdmin,
		Processes:        processList,
		TotalConnections: len(observed),
		BandwidthTracked: bandwidthTracked,
	}, next
}

func diffTraffic(previous TrafficTotals, current TrafficTotals, seconds float64) (uint64, uint64, bool) {
	if !current.Available {
		return 0, 0, false
	}
	if !previous.Available || seconds <= 0 {
		return 0, 0, true
	}
	if current.Sent < previous.Sent || current.Received < previous.Received {
		return 0, 0, true
	}

	return uint64(float64(current.Sent-previous.Sent) / seconds),
		uint64(float64(current.Received-previous.Received) / seconds),
		true
}

func itoa32(value int32) string {
	return strconv.FormatInt(int64(value), 10)
}
