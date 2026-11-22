package redis

import (
	"context"

	"github.com/emoss08/trenova/shared/redis/internal/instrument"
	"github.com/emoss08/trenova/shared/redis/internal/pool"
	"github.com/emoss08/trenova/shared/redis/internal/wire"
	"go.opentelemetry.io/otel"
)

type doFunc func(ctx context.Context, reqs []wire.Request, mode wire.Mode, apply wire.Apply) error

type Client struct {
	cmdable
	opt  *Options
	pool pool.Pool
	inst *instrument.Instrument
	t    *wire.Transport
}

func NewClient(opt *Options) *Client {
	o := *opt
	o.init()
	c := &Client{opt: &o, inst: newInstrument(&o)}

	dial := pool.DialOptions{
		Addr:        o.Addr,
		Username:    o.Username,
		Password:    o.Password,
		DB:          o.DB,
		DialTimeout: o.DialTimeout,
		TLSConfig:   o.TLSConfig,
	}
	if c.inst != nil {
		dial.Wrap = c.inst.Dial
	}
	c.pool = pool.New(pool.Options{
		Connect:     pool.Connector(&dial),
		Size:        o.PoolSize,
		MinIdle:     o.MinIdleConns,
		MaxIdleTime: o.ConnMaxIdleTime,
		MaxLifetime: o.ConnMaxLifetime,
		Timeout:     o.PoolTimeout,
	})
	if c.inst != nil {
		err := c.inst.ObservePool(instrument.PoolSource{
			Stats:   c.pool.Stats,
			Size:    o.PoolSize,
			MinIdle: o.MinIdleConns,
		})
		if err != nil {
			otel.Handle(err)
		}
	}
	c.t = wire.New(wire.Config{
		Pool:            c.pool,
		Instrument:      c.inst,
		ReadTimeout:     o.ReadTimeout,
		WriteTimeout:    o.WriteTimeout,
		MaxRetries:      o.MaxRetries,
		MinRetryBackoff: o.MinRetryBackoff,
		MaxRetryBackoff: o.MaxRetryBackoff,
	})
	c.cmdable = processor(c.t.Do)
	return c
}

func newInstrument(o *Options) *instrument.Instrument {
	if o.TracerProvider == nil && o.MeterProvider == nil {
		return nil
	}
	inst, err := instrument.New(instrument.Params{
		Addr:           o.Addr,
		TracerProvider: o.TracerProvider,
		MeterProvider:  o.MeterProvider,
	})
	if err != nil {
		otel.Handle(err)
		return nil
	}
	return inst
}

func (c *Client) Options() *Options { return c.opt }

func (c *Client) Close() error {
	c.pool.Close()
	if c.inst != nil {
		return c.inst.Close()
	}
	return nil
}

func (c *Client) Pipeline() *Pipeline {
	return newPipeline(c.t.Do, wire.Pipeline)
}

func (c *Client) TxPipeline() *Pipeline {
	return newPipeline(c.t.Do, wire.Tx)
}

func (c *Client) Pipelined(ctx context.Context, fn func(*Pipeline) error) ([]Cmder, error) {
	return c.Pipeline().run(ctx, fn)
}

func (c *Client) TxPipelined(ctx context.Context, fn func(*Pipeline) error) ([]Cmder, error) {
	return c.TxPipeline().run(ctx, fn)
}

func processor(do doFunc) cmdable {
	return func(ctx context.Context, cmd Cmder) error {
		return execute(ctx, do, []Cmder{cmd}, wire.Single)
	}
}

func execute(ctx context.Context, do doFunc, cmds []Cmder, mode wire.Mode) error {
	reqs := make([]wire.Request, len(cmds))
	for idx, cmd := range cmds {
		b := cmd.base()
		reqs[idx] = wire.Request{Args: b.args, Block: b.block, Blocking: b.blocking}
	}
	return do(ctx, reqs, mode, func(replies []wire.Reply) error {
		for idx, reply := range replies {
			if reply.Err != nil {
				cmds[idx].base().err = reply.Err
				continue
			}
			applyReply(cmds[idx], reply.Value)
		}
		for _, cmd := range cmds {
			if err := cmd.Err(); err != nil {
				return err
			}
		}
		return nil
	})
}
