//go:build windows

package collector

import (
	"syscall"

	gopsnet "github.com/shirou/gopsutil/v3/net"
	"golang.org/x/sys/windows"
)

func connectionStatStub(localIP string, localPort uint32, remoteIP string, remotePort uint32, status string) gopsnet.ConnectionStat {
	return gopsnet.ConnectionStat{
		Family: windows.AF_INET,
		Type:   syscall.SOCK_STREAM,
		Laddr: gopsnet.Addr{
			IP:   localIP,
			Port: localPort,
		},
		Raddr: gopsnet.Addr{
			IP:   remoteIP,
			Port: remotePort,
		},
		Status: status,
	}
}
