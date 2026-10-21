package imaging

import (
	"context"
	"errors"
	"fmt"
	"net"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config"
	"github.com/emoss08/trenova/internal/infrastructure/imaging/imagingpb"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var errNoEndpoints = status.Error(codes.Unavailable, "no imaging endpoint is available")

type endpoint struct {
	address string
	conn    *grpc.ClientConn
	rpc     imagingpb.ImagingServiceClient
	calls   sync.WaitGroup

	mu        sync.Mutex
	failures  int
	openUntil time.Time
	openFor   time.Duration
}

func (e *endpoint) available(now time.Time) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return !now.Before(e.openUntil)
}

type callOutcome struct {
	ctx        context.Context
	parent     context.Context
	err        error
	progressed bool
}

func (e *endpoint) record(
	outcome callOutcome,
	breaker *config.ImagingBreakerConfig,
	now time.Time,
) {
	e.mu.Lock()
	defer e.mu.Unlock()
	if !countsAgainstEndpoint(outcome) {
		if outcome.err == nil {
			e.failures = 0
			e.openFor = 0
			e.openUntil = time.Time{}
		}
		return
	}
	e.failures++
	if e.failures < breaker.GetConsecutiveFailures() && !draining(outcome.err) {
		return
	}
	if e.openFor == 0 {
		e.openFor = breaker.GetOpenFor()
	} else {
		e.openFor = min(e.openFor*2, breaker.GetMaxOpenFor())
	}
	e.openUntil = now.Add(e.openFor)
	e.failures = 0
}

func countsAgainstEndpoint(outcome callOutcome) bool {
	if outcome.err == nil {
		return false
	}
	if ctxErr := outcome.ctx.Err(); ctxErr != nil {
		return errors.Is(ctxErr, context.DeadlineExceeded) &&
			outcome.parent.Err() == nil &&
			!outcome.progressed
	}
	st, ok := status.FromError(outcome.err)
	if !ok {
		return false
	}
	code := st.Code()
	return code == codes.Unavailable || code == codes.DeadlineExceeded || code == codes.Unknown
}

func draining(err error) bool {
	return reasonOf(err) == "DRAINING"
}

type endpointSet struct {
	cfg    *config.ImagingConfig
	dial   func(target string) (*grpc.ClientConn, error)
	logger *zap.Logger

	lookup func(ctx context.Context) ([]string, error)
	static bool

	mu        sync.RWMutex
	endpoints []*endpoint
	next      atomic.Uint64

	resolveMu sync.Mutex
	retiring  sync.WaitGroup
	stop      chan struct{}
	done      chan struct{}
}

func newEndpointSet(
	cfg *config.ImagingConfig,
	logger *zap.Logger,
	dial func(target string) (*grpc.ClientConn, error),
) (*endpointSet, error) {
	set := &endpointSet{
		cfg:    cfg,
		dial:   dial,
		logger: logger,
		stop:   make(chan struct{}),
		done:   make(chan struct{}),
	}

	target := cfg.GetEndpoint()
	switch {
	case strings.HasPrefix(target, "dns:///"):
		host, port, err := net.SplitHostPort(strings.TrimPrefix(target, "dns:///"))
		if err != nil {
			return nil, fmt.Errorf("imaging.endpoint %q: %w", target, err)
		}
		set.lookup = func(ctx context.Context) ([]string, error) {
			addrs, lookupErr := net.DefaultResolver.LookupHost(ctx, host)
			if lookupErr != nil {
				return nil, lookupErr
			}
			for i, addr := range addrs {
				addrs[i] = net.JoinHostPort(addr, port)
			}
			return addrs, nil
		}
	default:
		set.static = true
		conn, err := dial(target)
		if err != nil {
			return nil, err
		}
		set.endpoints = []*endpoint{{
			address: target,
			conn:    conn,
			rpc:     imagingpb.NewImagingServiceClient(conn),
		}}
		close(set.done)
		return set, nil
	}

	go set.watch()
	return set, nil
}

func (s *endpointSet) watch() {
	defer close(s.done)
	ticker := time.NewTicker(s.cfg.GetResolveInterval())
	defer ticker.Stop()
	for {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := s.resolve(ctx); err != nil {
			s.logger.Warn("could not resolve the imaging endpoints", zap.Error(err))
		}
		cancel()
		select {
		case <-s.stop:
			return
		case <-ticker.C:
		}
	}
}

func (s *endpointSet) resolve(ctx context.Context) error {
	if s.static {
		return nil
	}
	s.resolveMu.Lock()
	defer s.resolveMu.Unlock()

	addrs, err := s.lookup(ctx)
	if err != nil {
		return err
	}
	wanted := make(map[string]struct{}, len(addrs))
	for _, addr := range addrs {
		wanted[addr] = struct{}{}
	}

	s.mu.RLock()
	current := slices.Clone(s.endpoints)
	s.mu.RUnlock()

	kept := make([]*endpoint, 0, len(wanted))
	have := make(map[string]struct{}, len(current))
	var retired []*endpoint
	for _, e := range current {
		if _, ok := wanted[e.address]; ok {
			kept = append(kept, e)
			have[e.address] = struct{}{}
			continue
		}
		retired = append(retired, e)
	}
	for address := range wanted {
		if _, ok := have[address]; ok {
			continue
		}
		conn, dialErr := s.dial("passthrough:///" + address)
		if dialErr != nil {
			s.logger.Warn("could not connect to an imaging endpoint",
				zap.String("address", address), zap.Error(dialErr))
			continue
		}
		kept = append(kept, &endpoint{
			address: address,
			conn:    conn,
			rpc:     imagingpb.NewImagingServiceClient(conn),
		})
	}
	slices.SortFunc(kept, func(a, b *endpoint) int { return strings.Compare(a.address, b.address) })

	s.mu.Lock()
	s.endpoints = kept
	s.mu.Unlock()

	for _, e := range retired {
		s.retire(e)
	}
	return nil
}

func (s *endpointSet) retire(e *endpoint) {
	s.retiring.Add(1)
	go func() {
		defer s.retiring.Done()
		finished := make(chan struct{})
		go func() {
			e.calls.Wait()
			close(finished)
		}()
		grace := time.NewTimer(s.cfg.Deadlines.Longest())
		defer grace.Stop()
		select {
		case <-finished:
		case <-s.stop:
		case <-grace.C:
			s.logger.Warn("closed a retired imaging endpoint with calls still in flight",
				zap.String("address", e.address))
		}
		_ = e.conn.Close()
	}()
}

func (s *endpointSet) pick(ctx context.Context, exclude map[string]struct{}) (*endpoint, error) {
	s.mu.RLock()
	empty := len(s.endpoints) == 0
	s.mu.RUnlock()
	if empty {
		if err := s.resolve(ctx); err != nil {
			return nil, fmt.Errorf("%w: %w", errNoEndpoints, err)
		}
	}

	s.mu.RLock()
	defer s.mu.RUnlock()
	n := len(s.endpoints)
	if n == 0 {
		return nil, errNoEndpoints
	}
	now := time.Now()
	start := int(s.next.Add(1) % uint64(n)) //nolint:gosec // the modulo keeps it below n
	var fallback *endpoint
	for i := range n {
		e := s.endpoints[(start+i)%n]
		if _, skip := exclude[e.address]; skip {
			if fallback == nil {
				fallback = e
			}
			continue
		}
		if e.available(now) {
			return claim(e), nil
		}
	}
	for i := range n {
		e := s.endpoints[(start+i)%n]
		if _, skip := exclude[e.address]; !skip {
			return claim(e), nil
		}
	}
	if fallback != nil {
		return claim(fallback), nil
	}
	return nil, errNoEndpoints
}

func claim(e *endpoint) *endpoint {
	e.calls.Add(1)
	return e
}

func (s *endpointSet) close() error {
	if !s.static {
		close(s.stop)
	}
	<-s.done
	s.retiring.Wait()
	s.mu.Lock()
	defer s.mu.Unlock()
	var errs []error
	for _, e := range s.endpoints {
		errs = append(errs, e.conn.Close())
	}
	s.endpoints = nil
	return errors.Join(errs...)
}
