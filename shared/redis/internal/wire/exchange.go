package wire

import (
	"context"
	"fmt"
	"time"

	"github.com/emoss08/trenova/shared/redis/internal/pool"
	"github.com/emoss08/trenova/shared/redis/internal/resp"
)

var (
	multiArgs = []any{"multi"}
	execArgs  = []any{"exec"}
)

func (t *Transport) exchange(
	ctx context.Context,
	cn *pool.Conn,
	reqs []Request,
	tx bool,
	replies []Reply,
) (read int, err error) {
	if err = encode(cn, reqs, tx); err != nil {
		fail(replies, err)
		return 0, nil
	}

	if ctx.Done() != nil {
		stop := context.AfterFunc(ctx, cn.Interrupt)
		defer func() {
			if !stop() && err == nil {
				err = ctx.Err()
			}
		}()
	}

	if err = cn.Flush(deadline(ctx, t.cfg.WriteTimeout)); err != nil {
		return 0, contextError(ctx, err)
	}

	readDeadline := deadline(ctx, t.readTimeout(reqs))
	if tx {
		read, err = readTx(cn, readDeadline, replies)
	} else {
		read, err = readReplies(cn, readDeadline, replies)
	}
	return read, contextError(ctx, err)
}

func encode(cn *pool.Conn, reqs []Request, tx bool) error {
	cn.Reset()
	if tx {
		if err := cn.Append(multiArgs); err != nil {
			return err
		}
	}
	for idx := range reqs {
		if err := cn.Append(reqs[idx].Args); err != nil {
			return err
		}
	}
	if tx {
		return cn.Append(execArgs)
	}
	return nil
}

func readReplies(cn *pool.Conn, deadline time.Time, replies []Reply) (int, error) {
	for idx := range replies {
		v, err := cn.Read(deadline)
		if err != nil {
			return idx, err
		}
		replies[idx] = Reply{Value: v}
	}
	return len(replies), nil
}

func readTx(cn *pool.Conn, deadline time.Time, replies []Reply) (int, error) {
	read := 0
	multi, err := cn.Read(deadline)
	if err != nil {
		return read, err
	}
	read++
	for idx := range replies {
		v, readErr := cn.Read(deadline)
		if readErr != nil {
			return read, readErr
		}
		read++
		replies[idx] = Reply{}
		if v.Kind == resp.KindError {
			replies[idx].Err = resp.Error(v.Str)
		}
	}

	exec, err := cn.Read(deadline)
	if err != nil {
		return read, err
	}
	read++
	if multi.Kind == resp.KindError {
		fail(replies, resp.Error(multi.Str))
		return read, nil
	}

	switch exec.Kind {
	case resp.KindNil:
		fail(replies, resp.TxFailedErr)
	case resp.KindError:
		for idx := range replies {
			if replies[idx].Err == nil {
				replies[idx].Err = resp.Error(exec.Str)
			}
		}
	case resp.KindArray:
		if len(exec.Elems) != len(replies) {
			return read, fmt.Errorf(
				"%w: EXEC returned %d replies for %d commands",
				resp.ErrProtocol, len(exec.Elems), len(replies),
			)
		}
		for idx := range replies {
			replies[idx].Value = exec.Elems[idx]
		}
	case resp.KindStatus, resp.KindInt, resp.KindBulk:
		return read, fmt.Errorf("%w: unexpected EXEC reply kind %d", resp.ErrProtocol, exec.Kind)
	default:
		return read, fmt.Errorf("%w: unexpected EXEC reply kind %d", resp.ErrProtocol, exec.Kind)
	}
	return read, nil
}

func (t *Transport) readTimeout(reqs []Request) time.Duration {
	timeout := t.cfg.ReadTimeout
	for idx := range reqs {
		if !reqs[idx].Blocking {
			continue
		}
		if reqs[idx].Block == 0 {
			return 0
		}
		timeout = max(timeout, reqs[idx].Block+t.cfg.ReadTimeout)
	}
	return timeout
}

func deadline(ctx context.Context, timeout time.Duration) time.Time {
	var d time.Time
	if timeout > 0 {
		d = time.Now().Add(timeout)
	}
	if ctxDeadline, ok := ctx.Deadline(); ok && (d.IsZero() || ctxDeadline.Before(d)) {
		d = ctxDeadline
	}
	return d
}

func contextError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if ctxErr := ctx.Err(); ctxErr != nil {
		return ctxErr
	}
	if d, ok := ctx.Deadline(); ok && !time.Now().Before(d) {
		return context.DeadlineExceeded
	}
	return err
}

func fail(replies []Reply, err error) {
	for idx := range replies {
		replies[idx] = Reply{Err: err}
	}
}
