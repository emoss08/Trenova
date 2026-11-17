package pollwatch

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/sourcegraph/conc/panics"
	"go.uber.org/fx"
	"go.uber.org/zap"
	"golang.org/x/time/rate"
)

const (
	FailedPoll   = "Failed to poll for task."
	probeEvery   = 5 * time.Second
	probeTimeout = 5 * time.Second
	reportEvery  = time.Minute
)

type Probe func(context.Context) error

type Params struct {
	fx.In

	Logger *zap.Logger
	LC     fx.Lifecycle
}

type Watch struct {
	logger  *zap.Logger
	done    chan struct{}
	wg      sync.WaitGroup
	mu      sync.Mutex
	probe   Probe
	down    bool
	stopped bool
	since   time.Time
	failed  int
	report  *rate.Sometimes
}

func New(p Params) *Watch {
	w := newWatch(p.Logger)
	p.LC.Append(fx.Hook{OnStop: w.stop})
	return w
}

func newWatch(logger *zap.Logger) *Watch {
	return &Watch{
		logger: logger.Named("temporal-polling"),
		done:   make(chan struct{}),
	}
}

func (w *Watch) Bind(probe Probe) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.probe = probe
}

func (w *Watch) Failed(err error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}
	if w.down {
		w.failed++
		return
	}

	w.down = true
	w.since = time.Now()
	w.failed = 1
	w.report = &rate.Sometimes{Interval: reportEvery}
	w.report.Do(func() {
		w.logger.Warn(
			"Cannot reach the workflow engine; background jobs keep trying and resume on their own",
			zap.Error(err),
		)
	})

	w.wg.Add(1)
	go w.watch()
}

func (w *Watch) watch() {
	defer w.wg.Done()

	ticker := time.NewTicker(probeEvery)
	defer ticker.Stop()

	for {
		select {
		case <-w.done:
			return
		case <-ticker.C:
		}

		if w.reachable() {
			w.recovered()
			return
		}
		w.stillDown()
	}
}

func (w *Watch) reachable() bool {
	w.mu.Lock()
	probe := w.probe
	w.mu.Unlock()

	if probe == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), probeTimeout)
	defer cancel()

	var err error
	recovered := panics.Try(func() { err = probe(ctx) })
	if recovered != nil {
		err = fmt.Errorf("probe panicked: %w", recovered.AsError())
	}

	return err == nil
}

func (w *Watch) stillDown() {
	w.mu.Lock()
	defer w.mu.Unlock()

	away := time.Since(w.since).Round(time.Second)
	failed := w.failed
	w.report.Do(func() {
		w.logger.Warn(
			"The workflow engine is still unreachable; background jobs keep trying",
			zap.Duration("away", away),
			zap.Int("failedPolls", failed),
		)
	})
}

func (w *Watch) recovered() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.logger.Info(
		"The workflow engine is reachable again; background jobs resumed",
		zap.Duration("away", time.Since(w.since).Round(time.Second)),
		zap.Int("failedPolls", w.failed),
	)
	w.down = false
	w.failed = 0
	w.report = nil
}

func (w *Watch) stop(context.Context) error {
	w.mu.Lock()
	if !w.stopped {
		w.stopped = true
		close(w.done)
	}
	w.mu.Unlock()

	w.wg.Wait()
	return nil
}
