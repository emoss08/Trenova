package sim

import (
	"sync"
	"time"
)

type ClockSnapshot struct {
	Now          time.Time
	Paused       bool
	Speed        float64
	LastWallTime time.Time
	LastSimTime  time.Time
}

type SimClock struct {
	mu       sync.RWMutex
	nowFn    func() time.Time
	simNow   time.Time
	lastWall time.Time
	lastSim  time.Time
	paused   bool
	speed    float64
}

func NewSimClock(start time.Time) *SimClock {
	return newSimClockWithNowFn(start, func() time.Time {
		return time.Now().UTC()
	})
}

func newSimClockWithNowFn(start time.Time, nowFn func() time.Time) *SimClock {
	initial := start.UTC()
	wallNow := nowFn().UTC()
	return &SimClock{
		nowFn:    nowFn,
		simNow:   initial,
		lastWall: wallNow,
		lastSim:  initial,
		paused:   false,
		speed:    1,
	}
}

func (c *SimClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.advanceLocked(c.nowFn().UTC())
}

func (c *SimClock) SetTime(value time.Time) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.simNow = value.UTC()
	c.lastSim = c.simNow
	c.lastWall = c.nowFn().UTC()
	return c.simNow
}

func (c *SimClock) SetPaused(paused bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.advanceLocked(c.nowFn().UTC())
	c.paused = paused
}

func (c *SimClock) SetSpeed(speed float64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	_ = c.advanceLocked(c.nowFn().UTC())
	c.speed = clampFloat64(speed, 0.1, 20)
}

func (c *SimClock) Step(duration time.Duration) time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	if duration <= 0 {
		return c.advanceLocked(c.nowFn().UTC())
	}
	now := c.advanceLocked(c.nowFn().UTC())
	c.simNow = now.Add(duration)
	c.lastSim = c.simNow
	c.lastWall = c.nowFn().UTC()
	return c.simNow
}

func (c *SimClock) Snapshot() ClockSnapshot {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := c.advanceLocked(c.nowFn().UTC())
	return ClockSnapshot{
		Now:          now,
		Paused:       c.paused,
		Speed:        c.speed,
		LastWallTime: c.lastWall,
		LastSimTime:  c.lastSim,
	}
}

func (c *SimClock) advanceLocked(wallNow time.Time) time.Time {
	if wallNow.Before(c.lastWall) {
		wallNow = c.lastWall
	}
	if c.paused {
		c.lastWall = wallNow
		c.lastSim = c.simNow
		return c.simNow
	}
	elapsed := wallNow.Sub(c.lastWall)
	if elapsed < 0 {
		elapsed = 0
	}
	advance := time.Duration(float64(elapsed) * c.speed)
	c.simNow = c.simNow.Add(advance)
	c.lastWall = wallNow
	c.lastSim = c.simNow
	return c.simNow
}
