package redis

import (
	"context"

	"github.com/emoss08/trenova/shared/redis/internal/wire"
)

type Pipeline struct {
	cmdable
	cmds []Cmder
	do   doFunc
	mode wire.Mode
}

func newPipeline(do doFunc, mode wire.Mode) *Pipeline {
	p := &Pipeline{do: do, mode: mode}
	p.cmdable = p.queue
	return p
}

func (p *Pipeline) queue(_ context.Context, cmd Cmder) error {
	p.cmds = append(p.cmds, cmd)
	return nil
}

func (p *Pipeline) Exec(ctx context.Context) ([]Cmder, error) {
	if len(p.cmds) == 0 {
		return nil, nil
	}
	cmds := p.cmds
	p.cmds = nil
	return cmds, execute(ctx, p.do, cmds, p.mode)
}

func (p *Pipeline) run(ctx context.Context, fn func(*Pipeline) error) ([]Cmder, error) {
	if err := fn(p); err != nil {
		return nil, err
	}
	return p.Exec(ctx)
}

type Conn struct {
	cmdable
	sticky *wire.Sticky
}

func (c *Client) Conn() *Conn {
	sticky := c.t.Sticky()
	return &Conn{cmdable: processor(sticky.Do), sticky: sticky}
}

func (c *Conn) Close() error {
	c.sticky.Release(ErrClosed)
	return nil
}

type Tx struct {
	cmdable
	sticky *wire.Sticky
}

func (t *Tx) TxPipelined(ctx context.Context, fn func(*Pipeline) error) ([]Cmder, error) {
	return newPipeline(t.sticky.Do, wire.Tx).run(ctx, fn)
}

func (c *Client) Watch(ctx context.Context, fn func(*Tx) error, keys ...string) error {
	sticky := c.t.Sticky()
	tx := &Tx{cmdable: processor(sticky.Do), sticky: sticky}
	defer func() {
		if sticky.Held() {
			_ = tx.status(context.WithoutCancel(ctx), "unwatch").Err()
		}
		sticky.Release(nil)
	}()

	if len(keys) > 0 {
		if err := tx.status(ctx, appendStrings([]any{"watch"}, keys)...).Err(); err != nil {
			return err
		}
	}
	return fn(tx)
}
