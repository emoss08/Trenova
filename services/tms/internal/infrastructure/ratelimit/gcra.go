package ratelimit

import (
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

type gcraResult struct {
	allowed    bool
	newTAT     int64
	remaining  int
	retryAfter time.Duration
	resetAfter time.Duration
}

func emissionInterval(policy kernelports.RateLimitPolicy) int64 {
	interval := policy.Period.Microseconds() / int64(policy.Rate)
	if interval < 1 {
		return 1
	}
	return interval
}

func evaluateGCRA(nowMicros, storedTAT int64, req kernelports.RateLimitRequest) gcraResult {
	interval := emissionInterval(req.Policy)
	tolerance := interval * int64(req.Policy.Burst)
	cost := int64(max(req.Cost, 1))

	tat := max(storedTAT, nowMicros)
	newTAT := tat + interval*cost
	allowAt := newTAT - tolerance
	diff := nowMicros - allowAt

	if diff < 0 {
		return gcraResult{
			allowed:    false,
			newTAT:     tat,
			remaining:  0,
			retryAfter: time.Duration(-diff) * time.Microsecond,
			resetAfter: time.Duration(tat-nowMicros) * time.Microsecond,
		}
	}

	return gcraResult{
		allowed:    true,
		newTAT:     newTAT,
		remaining:  int(diff / interval),
		retryAfter: 0,
		resetAfter: time.Duration(newTAT-nowMicros) * time.Microsecond,
	}
}

func decisionFrom(
	req kernelports.RateLimitRequest,
	res gcraResult,
) kernelports.RateLimitDecision {
	return kernelports.RateLimitDecision{
		Allowed:    res.allowed,
		Limit:      req.Policy.Burst,
		Remaining:  res.remaining,
		RetryAfter: res.retryAfter,
		ResetAfter: res.resetAfter,
	}
}

func validateRequests(requests []kernelports.RateLimitRequest) error {
	for i := range requests {
		if requests[i].Key == "" {
			return ErrEmptyKey
		}
		if requests[i].Policy.IsZero() {
			return ErrInvalidPolicy
		}
	}
	return nil
}
