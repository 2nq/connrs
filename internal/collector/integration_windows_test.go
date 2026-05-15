//go:build windows

package collector

import (
	"context"
	"testing"
	"time"
)

func TestDefaultCollectorPollsOnce(t *testing.T) {
	poller, err := NewDefault()
	if err != nil {
		t.Fatalf("expected default collector, got error: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	snapshot, err := poller.Poll(ctx)
	if err != nil {
		t.Fatalf("expected poll to succeed, got error: %v", err)
	}
	if snapshot.CapturedAt.IsZero() {
		t.Fatalf("expected snapshot capture time to be populated")
	}
}
