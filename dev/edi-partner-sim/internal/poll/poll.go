package poll

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"
)

const (
	firstInterval = 250 * time.Millisecond
	maxInterval   = 3 * time.Second
)

var ErrTimedOut = errors.New("timed out")

type permanent struct {
	err error
}

func (p *permanent) Error() string {
	return p.err.Error()
}

func (p *permanent) Unwrap() error {
	return p.err
}

func Stop(err error) error {
	return &permanent{err: err}
}

func Until(
	ctx context.Context,
	within time.Duration,
	check func(context.Context) (bool, error),
) error {
	policy := backoff.NewExponentialBackOff()
	policy.InitialInterval = firstInterval
	policy.MaxInterval = maxInterval
	var last, stopped error
	_, err := backoff.Retry(ctx, func() (struct{}, error) {
		done, checkErr := check(ctx)
		var stop *permanent
		switch {
		case errors.As(checkErr, &stop):
			stopped = stop.err
			return struct{}{}, backoff.Permanent(stop.err)
		case checkErr != nil:
			last = checkErr
			return struct{}{}, checkErr
		case !done:
			last = nil
			return struct{}{}, ErrTimedOut
		}
		return struct{}{}, nil
	}, backoff.WithBackOff(policy), backoff.WithMaxElapsedTime(within))
	switch {
	case err == nil:
		return nil
	case stopped != nil:
		return stopped
	case ctx.Err() != nil:
		return ctx.Err()
	case last != nil:
		return fmt.Errorf("%w after %s: %w", ErrTimedOut, within, last)
	}
	return fmt.Errorf("%w after %s", ErrTimedOut, within)
}
