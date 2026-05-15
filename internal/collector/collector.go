package collector

import (
	"context"
	"time"
)

type Sampler interface {
	Sample(ctx context.Context, isAdmin bool) ([]ObservedConnection, error)
}

type AdminChecker func() bool

type Collector struct {
	sampler   Sampler
	isAdmin   AdminChecker
	lastPoll  time.Time
	previous  map[ConnectionKey]TrafficTotals
	timeNow   func() time.Time
}

func New(sampler Sampler, isAdmin AdminChecker) *Collector {
	if isAdmin == nil {
		isAdmin = func() bool { return false }
	}
	return &Collector{
		sampler:  sampler,
		isAdmin:  isAdmin,
		previous: map[ConnectionKey]TrafficTotals{},
		timeNow:  time.Now,
	}
}

func (c *Collector) Poll(ctx context.Context) (Snapshot, error) {
	now := c.timeNow()
	isAdmin := c.isAdmin()

	observed, err := c.sampler.Sample(ctx, isAdmin)
	if err != nil {
		return Snapshot{}, err
	}

	snapshot, next := buildSnapshot(observed, c.previous, c.lastPoll, now, isAdmin)
	c.previous = next
	c.lastPoll = now
	return snapshot, nil
}
