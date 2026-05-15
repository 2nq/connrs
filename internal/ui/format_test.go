package ui

import "testing"

func TestFormatRateUsesBinaryUnits(t *testing.T) {
	if got, want := formatRate(1_536), "1.5 KB/s"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}

func TestFormatEndpointFallsBackToWildcard(t *testing.T) {
	if got, want := formatEndpoint("", 0), "*:*"; got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}
}
