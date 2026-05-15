//go:build windows

package collector

import (
	"encoding/binary"
	"errors"
	"math/bits"
	"net"
	"strconv"
	"strings"
	"syscall"
	"unsafe"

	gopsnet "github.com/shirou/gopsutil/v3/net"
	"golang.org/x/sys/windows"
)

const tcpConnectionEstatsData = 1

var (
	modIPHlpAPI                     = windows.NewLazySystemDLL("iphlpapi.dll")
	procGetPerTCPConnectionEStats   = modIPHlpAPI.NewProc("GetPerTcpConnectionEStats")
	procSetPerTCPConnectionEStats   = modIPHlpAPI.NewProc("SetPerTcpConnectionEStats")
	procGetPerTCP6ConnectionEStats  = modIPHlpAPI.NewProc("GetPerTcp6ConnectionEStats")
	procSetPerTCP6ConnectionEStats  = modIPHlpAPI.NewProc("SetPerTcp6ConnectionEStats")
	errBandwidthNotSupported        = errors.New("bandwidth sampling not supported")
)

type tcpBandwidthSampler struct {
	enabled map[ConnectionKey]struct{}
}

type mibTCPRow struct {
	State      uint32
	LocalAddr  uint32
	LocalPort  uint32
	RemoteAddr uint32
	RemotePort uint32
}

type mibTCP6Row struct {
	State         uint32
	LocalAddr     [16]byte
	LocalScopeID  uint32
	LocalPort     uint32
	RemoteAddr    [16]byte
	RemoteScopeID uint32
	RemotePort    uint32
}

type tcpEstatsDataRWv0 struct {
	EnableCollection byte
}

type tcpEstatsDataRODv0 struct {
	DataBytesOut      uint64
	DataSegsOut       uint64
	DataBytesIn       uint64
	DataSegsIn        uint64
	SegsOut           uint64
	SegsIn            uint64
	SoftErrors        uint32
	SoftErrorReason   uint32
	SndUna            uint32
	SndNxt            uint32
	SndMax            uint32
	_                 uint32
	ThruBytesAcked    uint64
	RcvNxt            uint32
	_                 uint32
	ThruBytesReceived uint64
}

func newTCPBandwidthSampler() *tcpBandwidthSampler {
	return &tcpBandwidthSampler{
		enabled: make(map[ConnectionKey]struct{}),
	}
}

func (s *tcpBandwidthSampler) sample(conn gopsnet.ConnectionStat, key ConnectionKey) (TrafficTotals, bool) {
	if conn.Type != syscall.SOCK_STREAM {
		return TrafficTotals{}, false
	}
	if !canTrackState(conn.Status) {
		return TrafficTotals{}, false
	}

	switch conn.Family {
	case windows.AF_INET:
		row, ok := makeMIBTCPRow(conn)
		if !ok {
			return TrafficTotals{}, false
		}
		if err := s.ensureEnabledIPv4(key, row); err != nil {
			return TrafficTotals{}, false
		}
		totals, err := readTCPv4(row)
		if err != nil {
			delete(s.enabled, key)
			return TrafficTotals{}, false
		}
		return totals, true
	case windows.AF_INET6:
		row, ok := makeMIBTCP6Row(conn)
		if !ok {
			return TrafficTotals{}, false
		}
		if err := s.ensureEnabledIPv6(key, row); err != nil {
			return TrafficTotals{}, false
		}
		totals, err := readTCPv6(row)
		if err != nil {
			delete(s.enabled, key)
			return TrafficTotals{}, false
		}
		return totals, true
	default:
		return TrafficTotals{}, false
	}
}

func (s *tcpBandwidthSampler) ensureEnabledIPv4(key ConnectionKey, row mibTCPRow) error {
	if _, ok := s.enabled[key]; ok {
		return nil
	}
	if err := enableTCPv4(row); err != nil {
		return err
	}
	s.enabled[key] = struct{}{}
	return nil
}

func (s *tcpBandwidthSampler) ensureEnabledIPv6(key ConnectionKey, row mibTCP6Row) error {
	if _, ok := s.enabled[key]; ok {
		return nil
	}
	if err := enableTCPv6(row); err != nil {
		return err
	}
	s.enabled[key] = struct{}{}
	return nil
}

func canTrackState(status string) bool {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "ESTABLISHED", "CLOSE_WAIT", "FIN_WAIT1", "FIN_WAIT2", "LAST_ACK", "TIME_WAIT", "CLOSING":
		return true
	default:
		return false
	}
}

func makeMIBTCPRow(conn gopsnet.ConnectionStat) (mibTCPRow, bool) {
	local, ok := parseIPv4(conn.Laddr.IP)
	if !ok {
		return mibTCPRow{}, false
	}
	remote, ok := parseIPv4(conn.Raddr.IP)
	if !ok {
		return mibTCPRow{}, false
	}
	state, ok := tcpStateCode(conn.Status)
	if !ok {
		return mibTCPRow{}, false
	}
	return mibTCPRow{
		State:      state,
		LocalAddr:  local,
		LocalPort:  htons(conn.Laddr.Port),
		RemoteAddr: remote,
		RemotePort: htons(conn.Raddr.Port),
	}, true
}

func makeMIBTCP6Row(conn gopsnet.ConnectionStat) (mibTCP6Row, bool) {
	localAddr, localScopeID, ok := parseIPv6(conn.Laddr.IP)
	if !ok {
		return mibTCP6Row{}, false
	}
	remoteAddr, remoteScopeID, ok := parseIPv6(conn.Raddr.IP)
	if !ok {
		return mibTCP6Row{}, false
	}
	state, ok := tcpStateCode(conn.Status)
	if !ok {
		return mibTCP6Row{}, false
	}
	return mibTCP6Row{
		State:         state,
		LocalAddr:     localAddr,
		LocalScopeID:  htonl(localScopeID),
		LocalPort:     htons(conn.Laddr.Port),
		RemoteAddr:    remoteAddr,
		RemoteScopeID: htonl(remoteScopeID),
		RemotePort:    htons(conn.Raddr.Port),
	}, true
}

func enableTCPv4(row mibTCPRow) error {
	cfg := tcpEstatsDataRWv0{EnableCollection: 1}
	ret, _, _ := procSetPerTCPConnectionEStats.Call(
		uintptr(unsafe.Pointer(&row)),
		uintptr(tcpConnectionEstatsData),
		uintptr(unsafe.Pointer(&cfg)),
		0,
		unsafe.Sizeof(cfg),
		0,
	)
	if ret != 0 {
		return windows.Errno(ret)
	}
	return nil
}

func enableTCPv6(row mibTCP6Row) error {
	cfg := tcpEstatsDataRWv0{EnableCollection: 1}
	ret, _, _ := procSetPerTCP6ConnectionEStats.Call(
		uintptr(unsafe.Pointer(&row)),
		uintptr(tcpConnectionEstatsData),
		uintptr(unsafe.Pointer(&cfg)),
		0,
		unsafe.Sizeof(cfg),
		0,
	)
	if ret != 0 {
		return windows.Errno(ret)
	}
	return nil
}

func readTCPv4(row mibTCPRow) (TrafficTotals, error) {
	stats := tcpEstatsDataRODv0{}
	ret, _, _ := procGetPerTCPConnectionEStats.Call(
		uintptr(unsafe.Pointer(&row)),
		uintptr(tcpConnectionEstatsData),
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&stats)),
		0,
		unsafe.Sizeof(stats),
	)
	if ret != 0 {
		return TrafficTotals{}, windows.Errno(ret)
	}
	return TrafficTotals{
		Sent:      stats.DataBytesOut,
		Received:  stats.DataBytesIn,
		Available: true,
	}, nil
}

func readTCPv6(row mibTCP6Row) (TrafficTotals, error) {
	stats := tcpEstatsDataRODv0{}
	ret, _, _ := procGetPerTCP6ConnectionEStats.Call(
		uintptr(unsafe.Pointer(&row)),
		uintptr(tcpConnectionEstatsData),
		0,
		0,
		0,
		0,
		0,
		0,
		uintptr(unsafe.Pointer(&stats)),
		0,
		unsafe.Sizeof(stats),
	)
	if ret != 0 {
		return TrafficTotals{}, windows.Errno(ret)
	}
	return TrafficTotals{
		Sent:      stats.DataBytesOut,
		Received:  stats.DataBytesIn,
		Available: true,
	}, nil
}

func parseIPv4(raw string) (uint32, bool) {
	if strings.TrimSpace(raw) == "" {
		return 0, true
	}

	ip := net.ParseIP(raw)
	if ip == nil {
		return 0, false
	}
	ipv4 := ip.To4()
	if ipv4 == nil {
		return 0, false
	}
	return binary.LittleEndian.Uint32(ipv4), true
}

func parseIPv6(raw string) ([16]byte, uint32, bool) {
	var addr [16]byte
	if strings.TrimSpace(raw) == "" {
		return addr, 0, true
	}

	host, zone := splitZone(raw)
	ip := net.ParseIP(host)
	if ip == nil {
		return addr, 0, false
	}
	ip = ip.To16()
	if ip == nil {
		return addr, 0, false
	}
	copy(addr[:], ip)
	return addr, zone, true
}

func splitZone(raw string) (string, uint32) {
	host, zone, found := strings.Cut(raw, "%")
	if !found {
		return raw, 0
	}
	value, err := strconv.ParseUint(zone, 10, 32)
	if err != nil {
		return host, 0
	}
	return host, uint32(value)
}

func tcpStateCode(status string) (uint32, bool) {
	switch strings.ToUpper(strings.TrimSpace(status)) {
	case "CLOSED":
		return 1, true
	case "LISTEN":
		return 2, true
	case "SYN_SENT":
		return 3, true
	case "SYN_RECV", "SYN_RECEIVED":
		return 4, true
	case "ESTABLISHED":
		return 5, true
	case "FIN_WAIT1":
		return 6, true
	case "FIN_WAIT2":
		return 7, true
	case "CLOSE_WAIT":
		return 8, true
	case "CLOSING":
		return 9, true
	case "LAST_ACK":
		return 10, true
	case "TIME_WAIT":
		return 11, true
	case "DELETE_TCB":
		return 12, true
	default:
		return 0, false
	}
}

func htons(port uint32) uint32 {
	return uint32(bits.ReverseBytes16(uint16(port)))
}

func htonl(value uint32) uint32 {
	return bits.ReverseBytes32(value)
}
