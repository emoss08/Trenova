package wire

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/cenkalti/backoff/v5"
	"github.com/emoss08/trenova/shared/redis/internal/instrument"
	"github.com/emoss08/trenova/shared/redis/internal/pool"
	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

type Mode uint8

const (
	Single Mode = iota
	Pipeline
	Tx
)

var errConnLost = errors.New("redis: connection lost")

type Request struct {
	Args     []any
	Block    time.Duration
	Blocking bool
}

type Reply struct {
	Value resp.Value
	Err   error
}

type Apply func(replies []Reply) error

type Config struct {
	Pool            pool.Pool
	Instrument      *instrument.Instrument
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
	MaxRetries      int
	MinRetryBackoff time.Duration
	MaxRetryBackoff time.Duration
}

type Transport struct {
	cfg Config
}

func New(cfg Config) *Transport {
	return &Transport{cfg: cfg}
}

func (t *Transport) Do(ctx context.Context, reqs []Request, mode Mode, apply Apply) error {
	return t.observe(ctx, reqs, mode, apply, t.run)
}

func (t *Transport) observe(
	ctx context.Context,
	reqs []Request,
	mode Mode,
	apply Apply,
	run func(context.Context, []Request, Mode) []Reply,
) error {
	inst := t.cfg.Instrument
	if inst == nil {
		return apply(run(ctx, reqs, mode))
	}

	var end func(error)
	if mode == Single {
		ctx, end = inst.Command(ctx, reqs[0].Args)
	} else {
		ctx, end = inst.Pipeline(ctx, argsOf(reqs, mode))
	}
	err := apply(run(ctx, reqs, mode))
	end(err)
	return err
}

func argsOf(reqs []Request, mode Mode) [][]any {
	out := make([][]any, 0, len(reqs)+2)
	if mode == Tx {
		out = append(out, multiArgs)
	}
	for idx := range reqs {
		out = append(out, reqs[idx].Args)
	}
	if mode == Tx {
		out = append(out, execArgs)
	}
	return out
}

func (t *Transport) run(ctx context.Context, reqs []Request, mode Mode) []Reply {
	replies := make([]Reply, len(reqs))
	retryTimeout := !blocking(reqs)
	var bo *backoff.ExponentialBackOff
	for attempt := 0; ; attempt++ {
		read, err := t.attempt(ctx, reqs, mode, replies)
		if err == nil {
			if mode != Single || replies[0].Value.Kind != resp.KindError {
				return replies
			}
			err = resp.Error(replies[0].Value.Str)
			if attempt >= t.cfg.MaxRetries || !shouldRetry(err, retryTimeout) {
				return replies
			}
		} else if attempt >= t.cfg.MaxRetries || !shouldRetry(err, retryTimeout) ||
			(read > 0 && !isServerError(err)) {
			fail(replies, err)
			return replies
		}

		if bo == nil {
			bo = &backoff.ExponentialBackOff{
				InitialInterval:     t.cfg.MinRetryBackoff,
				RandomizationFactor: backoff.DefaultRandomizationFactor,
				Multiplier:          2,
				MaxInterval:         t.cfg.MaxRetryBackoff,
			}
		}
		if sleepErr := sleep(ctx, bo.NextBackOff()); sleepErr != nil {
			fail(replies, sleepErr)
			return replies
		}
	}
}

func (t *Transport) attempt(
	ctx context.Context,
	reqs []Request,
	mode Mode,
	replies []Reply,
) (int, error) {
	cn, err := t.cfg.Pool.Get(ctx)
	if err != nil {
		return 0, err
	}
	read, err := t.exchange(ctx, cn, reqs, mode == Tx, replies)
	if err != nil {
		t.cfg.Pool.Remove(cn)
		return read, err
	}
	t.cfg.Pool.Put(cn)
	return read, nil
}

type Sticky struct {
	t    *Transport
	cn   *pool.Conn
	lost error
}

func (t *Transport) Sticky() *Sticky {
	return &Sticky{t: t}
}

func (s *Sticky) Do(ctx context.Context, reqs []Request, mode Mode, apply Apply) error {
	return s.t.observe(ctx, reqs, mode, apply, s.run)
}

func (s *Sticky) run(ctx context.Context, reqs []Request, mode Mode) []Reply {
	replies := make([]Reply, len(reqs))
	if s.lost != nil {
		fail(replies, s.lost)
		return replies
	}
	if s.cn == nil {
		cn, err := s.t.cfg.Pool.Get(ctx)
		if err != nil {
			fail(replies, err)
			return replies
		}
		s.cn = cn
	}

	if _, err := s.t.exchange(ctx, s.cn, reqs, mode == Tx, replies); err != nil {
		s.t.cfg.Pool.Remove(s.cn)
		s.cn = nil
		s.lost = fmt.Errorf("%w: %w", errConnLost, err)
		fail(replies, err)
	}
	return replies
}

func (s *Sticky) Held() bool {
	return s.cn != nil
}

func (s *Sticky) Release(err error) {
	if s.cn != nil {
		s.t.cfg.Pool.Put(s.cn)
		s.cn = nil
	}
	if err != nil {
		s.lost = err
	}
}

func blocking(reqs []Request) bool {
	for idx := range reqs {
		if reqs[idx].Blocking {
			return true
		}
	}
	return false
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
