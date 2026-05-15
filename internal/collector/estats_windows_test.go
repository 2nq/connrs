//go:build windows

package collector

import "testing"

func TestParseIPv4MatchesWindowsInAddrLayout(t *testing.T) {
	value, ok := parseIPv4("127.0.0.1")
	if !ok {
		t.Fatalf("expected IPv4 parse to succeed")
	}

	const want uint32 = 0x0100007f
	if value != want {
		t.Fatalf("expected Windows in_addr layout 0x%08x, got 0x%08x", want, value)
	}
}

func TestMakeMIBTCPRowEncodesIPv4AndPortsForWindows(t *testing.T) {
	row, ok := makeMIBTCPRow(connectionStatStub(
		"192.168.1.10",
		51515,
		"1.1.1.1",
		443,
		"ESTABLISHED",
	))
	if !ok {
		t.Fatalf("expected row creation to succeed")
	}

	if row.LocalAddr != 0x0a01a8c0 {
		t.Fatalf("unexpected local addr encoding: 0x%08x", row.LocalAddr)
	}
	if row.RemoteAddr != 0x01010101 {
		t.Fatalf("unexpected remote addr encoding: 0x%08x", row.RemoteAddr)
	}
	if row.LocalPort != 0x00003bc9 {
		t.Fatalf("unexpected local port encoding: 0x%08x", row.LocalPort)
	}
	if row.RemotePort != 0x0000bb01 {
		t.Fatalf("unexpected remote port encoding: 0x%08x", row.RemotePort)
	}
}
