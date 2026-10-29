package ratelimit

import (
	"context"
	"errors"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/securityconfig"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"go.uber.org/zap"
)

const storeErrorLogInterval = 30 * time.Second

type ResilientStoreOptions struct {
	Primary     kernelports.RateLimitStore
	Fallback    kernelports.RateLimitStore
	FailureMode string
	Timeout     time.Duration
	Logger      *zap.Logger
	OnFailure   func(mode string)
	Now         func() time.Time
}

type ResilientStore struct {
	primary     kernelports.RateLimitStore
	fallback    kernelports.RateLimitStore
	failureMode string
	timeout     time.Duration
	logger      *zap.Logger
	onFailure   func(mode string)
	now         func() time.Time
	lastLogged  atomic.Int64
}

func NewResilientStore(opts ResilientStoreOptions) *ResilientStore {
	logger := opts.Logger
	if logger == nil {
		logger = zap.NewNop()
	}
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	mode := opts.FailureMode
	if mode == "" {
		mode = securityconfig.RateLimitFailureModeLocal
	}
	return &ResilientStore{
		primary:     opts.Primary,
		fallback:    opts.Fallback,
		failureMode: mode,
		timeout:     opts.Timeout,
		logger:      logger.Named("ratelimit.store"),
		onFailure:   opts.OnFailure,
		now:         now,
	}
}

func (s *ResilientStore) Check(
	ctx context.Context,
	requests []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	checkCtx := ctx
	if s.timeout > 0 {
		var cancel context.CancelFunc
		checkCtx, cancel = context.WithTimeout(ctx, s.timeout)
		defer cancel()
	}

	decisions, err := s.primary.Check(checkCtx, requests)
	if err == nil {
		return decisions, nil
	}
	if errors.Is(err, ErrEmptyKey) || errors.Is(err, ErrInvalidPolicy) {
		return nil, err
	}
	if ctx.Err() != nil {
		return nil, err
	}

	s.reportFailure(err)

	switch s.failureMode {
	case securityconfig.RateLimitFailureModeAllow:
		return allowAll(requests), nil
	case securityconfig.RateLimitFailureModeDeny:
		return nil, ErrStoreDenied
	default:
		if s.fallback == nil {
			return allowAll(requests), nil
		}
		return s.fallback.Check(ctx, requests)
	}
}

func (s *ResilientStore) reportFailure(err error) {
	if s.onFailure != nil {
		s.onFailure(s.failureMode)
	}

	now := s.now().UnixNano()
	last := s.lastLogged.Load()
	if last != 0 && now-last < storeErrorLogInterval.Nanoseconds() {
		return
	}
	if s.lastLogged.CompareAndSwap(last, now) {
		s.logger.Warn(
			"rate limit store unavailable",
			zap.String("failureMode", s.failureMode),
			zap.Error(err),
		)
	}
}

func allowAll(requests []kernelports.RateLimitRequest) []kernelports.RateLimitDecision {
	decisions := make([]kernelports.RateLimitDecision, len(requests))
	for i := range requests {
		decisions[i] = kernelports.RateLimitDecision{
			Allowed:   true,
			Limit:     requests[i].Policy.Burst,
			Remaining: requests[i].Policy.Burst,
		}
	}
	return decisions
}
