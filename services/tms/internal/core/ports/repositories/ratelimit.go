package repositories

import (
	"context"
	"time"
)

type RateLimitPolicy struct {
	Rate   int
	Period time.Duration
	Burst  int
}

func (p RateLimitPolicy) IsZero() bool {
	return p.Rate <= 0 || p.Burst <= 0 || p.Period <= 0
}

type RateLimitRequest struct {
	Key    string
	Policy RateLimitPolicy
	Cost   int
}

type RateLimitDecision struct {
	Allowed    bool
	Limit      int
	Remaining  int
	RetryAfter time.Duration
	ResetAfter time.Duration
}

type RateLimitStore interface {
	Check(ctx context.Context, requests []RateLimitRequest) ([]RateLimitDecision, error)
}
