package scheduletest

import (
	"sync"
	"time"
)

type waiter struct {
	at time.Time
	ch chan time.Time
}

type Clock struct {
	mu      sync.Mutex
	now     time.Time
	waiters []waiter
}

func NewClock(start time.Time) *Clock {
	return &Clock{now: start}
}

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *Clock) Until(at time.Time) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	ch := make(chan time.Time, 1)
	if !at.After(c.now) {
		ch <- c.now
		return ch
	}
	c.waiters = append(c.waiters, waiter{at: at, ch: ch})
	return ch
}

func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
	kept := c.waiters[:0]
	for _, w := range c.waiters {
		if w.at.After(c.now) {
			kept = append(kept, w)
			continue
		}
		w.ch <- c.now
	}
	c.waiters = kept
}
