package datausageservice

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/datausageports"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/maypok86/otter/v2"
	"github.com/sourcegraph/conc"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	RefreshEvery = 5 * time.Minute
	KeepFor      = 30 * time.Minute
	FirstWait    = 2 * time.Second
)

type Params struct {
	fx.In

	Sources []datausageports.Source `group:"managed_data_sources"`
	Clock   timeutils.Clock
	Logger  *zap.Logger
	LC      fx.Lifecycle
}

type measurement struct {
	bytes int64
	at    int64
}

type Service struct {
	sources []datausageports.Source
	clock   timeutils.Clock
	l       *zap.Logger
	values  *otter.Cache[datausageports.Figure, measurement]

	ctx  context.Context
	stop context.CancelFunc

	mu          sync.Mutex
	attempted   bool
	attemptedAt int64
	running     chan struct{}
	measured    bool
}

var _ datausageports.Reader = (*Service)(nil)

func New(p Params) (*Service, error) {
	seen := make(map[datausageports.Figure]bool, len(p.Sources))
	for _, src := range p.Sources {
		figure := src.Figure()
		if !figure.IsValid() {
			return nil, fmt.Errorf("a data usage source measures an unknown figure %q", figure)
		}
		if seen[figure] {
			return nil, fmt.Errorf("two data usage sources measure %s", figure)
		}
		seen[figure] = true
	}
	values, err := otter.New(&otter.Options[datausageports.Figure, measurement]{
		ExpiryCalculator: otter.ExpiryWriting[datausageports.Figure, measurement](KeepFor),
		Clock:            secondsClock{clock: p.Clock},
	})
	if err != nil {
		return nil, fmt.Errorf("create the data usage cache: %w", err)
	}
	ctx, stop := context.WithCancel(context.Background())
	s := &Service{
		sources: p.Sources,
		clock:   p.Clock,
		l:       p.Logger.Named("managed-data-usage"),
		values:  values,
		ctx:     ctx,
		stop:    stop,
	}
	p.LC.Append(fx.Hook{OnStop: s.shutdown})
	return s, nil
}

func (s *Service) Usage(ctx context.Context) *datausageports.Usage {
	if wait := s.refreshIfDue(); wait != nil {
		timer := time.NewTimer(FirstWait)
		defer timer.Stop()
		select {
		case <-wait:
		case <-timer.C:
		case <-ctx.Done():
		}
	}
	return s.read()
}

func (s *Service) refreshIfDue() <-chan struct{} {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := s.clock()
	if s.running != nil ||
		(s.attempted && now-s.attemptedAt < int64(RefreshEvery/time.Second)) {
		return nil
	}
	s.attempted = true
	s.attemptedAt = now
	done := make(chan struct{})
	s.running = done
	go s.round(done)
	if s.measured {
		return nil
	}
	return done
}

func (s *Service) round(done chan struct{}) {
	var wg conc.WaitGroup
	for _, src := range s.sources {
		wg.Go(func() { s.measure(src) })
	}
	wg.Wait()
	s.mu.Lock()
	s.running = nil
	s.measured = true
	s.mu.Unlock()
	close(done)
}

type outcome struct {
	bytes int64
	err   error
}

func (s *Service) measure(src datausageports.Source) {
	figure := src.Figure()
	ctx, cancel := context.WithTimeout(s.ctx, src.Timeout())
	defer cancel()
	at := s.clock()
	result := make(chan outcome, 1)
	go func() {
		bytes, err := src.Measure(ctx)
		result <- outcome{bytes: bytes, err: err}
	}()
	var got outcome
	select {
	case got = <-result:
	case <-ctx.Done():
		got = outcome{err: ctx.Err()}
	}
	switch {
	case errors.Is(got.err, datausageports.ErrNotMeasured):
		s.l.Debug("this instance does not measure a data usage figure",
			zap.String("figure", string(figure)))
	case got.err != nil:
		s.l.Warn("cannot measure a data usage figure; the last one is kept for up to 30 minutes",
			zap.String("figure", string(figure)), zap.Error(got.err))
	case got.bytes < 0:
		s.l.Warn("a data usage source answered a negative size",
			zap.String("figure", string(figure)), zap.Int64("bytes", got.bytes))
	default:
		s.values.Set(figure, measurement{bytes: got.bytes, at: at})
	}
}

func (s *Service) read() *datausageports.Usage {
	out := &datausageports.Usage{}
	for figure, dst := range map[datausageports.Figure]**int64{
		datausageports.FigureDatabase:  &out.DatabaseBytes,
		datausageports.FigureDocuments: &out.DocumentsBytes,
		datausageports.FigureSearch:    &out.SearchBytes,
	} {
		m, ok := s.values.GetIfPresent(figure)
		if !ok {
			continue
		}
		bytes := m.bytes
		*dst = &bytes
		if out.MeasuredAt == 0 || m.at < out.MeasuredAt {
			out.MeasuredAt = m.at
		}
	}
	if out.MeasuredAt == 0 {
		out.MeasuredAt = s.clock()
	}
	return out
}

func (s *Service) shutdown(ctx context.Context) error {
	s.stop()
	s.mu.Lock()
	running := s.running
	s.mu.Unlock()
	if running == nil {
		return nil
	}
	select {
	case <-running:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

type secondsClock struct {
	clock timeutils.Clock
}

func (c secondsClock) NowNano() int64 {
	return c.clock() * int64(time.Second)
}

func (secondsClock) Tick(time.Duration) <-chan time.Time {
	return nil
}
