package restx

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	defaultMaxAttempts    = 3
	defaultInitialBackoff = 250 * time.Millisecond
	defaultMaxBackoff     = 10 * time.Second
	maxRetryAfter         = 30 * time.Second
)

type RetryConfig struct {
	Enabled        bool
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func (r RetryConfig) normalized() RetryConfig {
	if !r.Enabled {
		return RetryConfig{MaxAttempts: 1}
	}
	if r.MaxAttempts <= 0 {
		r.MaxAttempts = defaultMaxAttempts
	}
	if r.InitialBackoff <= 0 {
		r.InitialBackoff = defaultInitialBackoff
	}
	if r.MaxBackoff <= 0 {
		r.MaxBackoff = defaultMaxBackoff
	}
	if r.MaxBackoff < r.InitialBackoff {
		r.MaxBackoff = r.InitialBackoff
	}
	return r
}

func (r RetryConfig) backoff(attempt int) time.Duration {
	delay := r.InitialBackoff
	for i := 1; i < attempt; i++ {
		if delay >= r.MaxBackoff/2 {
			return r.MaxBackoff
		}
		delay *= 2
	}
	return min(delay, r.MaxBackoff)
}

func (r RetryConfig) delay(
	attempt int,
	retryAfter time.Duration,
	hasRetryAfter bool,
) time.Duration {
	if hasRetryAfter {
		return min(max(retryAfter, 0), maxRetryAfter)
	}
	return r.backoff(attempt)
}

func ParseRetryAfter(value string) (time.Duration, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return 0, false
	}

	seconds, err := strconv.Atoi(value)
	if err == nil && seconds >= 0 {
		return time.Duration(seconds) * time.Second, true
	}

	t, err := http.ParseTime(value)
	if err != nil {
		return 0, false
	}
	until := time.Until(t)
	if until < 0 {
		return 0, true
	}
	return until, true
}

func retryableStatus(status int) bool {
	if status == http.StatusTooManyRequests {
		return true
	}
	return status >= http.StatusInternalServerError && status != http.StatusNotImplemented
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return ctx.Err()
	}
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}
