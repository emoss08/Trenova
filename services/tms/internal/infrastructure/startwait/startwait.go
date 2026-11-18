package startwait

import (
	"context"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"
	"go.uber.org/zap"
)

const (
	reportEvery     = time.Minute
	attemptTimeout  = 10 * time.Second
	initialInterval = 500 * time.Millisecond
	maxInterval     = 5 * time.Second
)

var processStart = time.Now()

type Params struct {
	Logger     *zap.Logger
	Dependency string
	Deadline   time.Time
	Check      func(context.Context) error
}

func Deadline(budget time.Duration) time.Time {
	return processStart.Add(budget)
}

func Wait(ctx context.Context, p Params) error {
	w := &waiting{
		params: p,
		logger: p.Logger.With(zap.String("dependency", p.Dependency)),
	}

	retryCtx, cancel := context.WithDeadline(ctx, p.Deadline)
	defer cancel()

	_, err := backoff.Retry(
		retryCtx,
		func() (struct{}, error) { return struct{}{}, w.attempt(ctx) },
		backoff.WithBackOff(newBackOff()),
		backoff.WithMaxElapsedTime(0),
		backoff.WithNotify(w.notify),
	)
	if err != nil {
		return w.gaveUp()
	}

	w.answered()
	return nil
}

type waiting struct {
	params     Params
	logger     *zap.Logger
	lastErr    error
	started    time.Time
	lastReport time.Time
	announced  bool
}

func (w *waiting) attempt(parent context.Context) error {
	ctx, cancel := context.WithTimeout(parent, attemptTimeout)
	defer cancel()

	err := w.params.Check(ctx)
	if err != nil {
		if w.lastErr == nil {
			w.started = time.Now()
		}
		w.lastErr = err
	}

	return err
}

func (w *waiting) notify(err error, _ time.Duration) {
	now := time.Now()
	if !w.announced {
		w.announced = true
		w.lastReport = now
		w.logger.Info(
			fmt.Sprintf("Waiting for %s to answer before starting", w.params.Dependency),
			zap.Error(err),
			zap.Duration("timeout", time.Until(w.params.Deadline).Round(time.Second)),
		)
		return
	}

	if now.Sub(w.lastReport) < reportEvery {
		return
	}
	w.lastReport = now
	w.logger.Info(
		fmt.Sprintf("Still waiting for %s to answer", w.params.Dependency),
		zap.Error(err),
		zap.Duration("waited", w.waited()),
	)
}

func (w *waiting) answered() {
	if !w.announced {
		return
	}
	w.logger.Info(
		fmt.Sprintf("%s answered; starting", w.params.Dependency),
		zap.Duration("waited", w.waited()),
	)
}

func (w *waiting) gaveUp() error {
	return fmt.Errorf(
		"%s did not answer within the start-up wait (app.startupWait, until %s; waited %s): %w",
		w.params.Dependency,
		w.params.Deadline.UTC().Format(time.RFC3339),
		w.waited(),
		w.lastErr,
	)
}

func (w *waiting) waited() time.Duration {
	if w.started.IsZero() {
		return 0
	}
	return time.Since(w.started).Round(100 * time.Millisecond)
}

func newBackOff() backoff.BackOff {
	b := backoff.NewExponentialBackOff()
	b.InitialInterval = initialInterval
	b.MaxInterval = maxInterval
	b.Multiplier = 2
	b.RandomizationFactor = 0.2
	return b
}
