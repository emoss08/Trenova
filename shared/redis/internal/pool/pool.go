package pool

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jackc/puddle/v2"
)

const maintenancePeriod = time.Minute

var (
	ErrClosed      = errors.New("redis: client is closed")
	ErrPoolTimeout = errors.New("redis: connection pool timeout")
)

type Options struct {
	Connect     func(ctx context.Context) (*Conn, error)
	Size        int
	MinIdle     int
	MaxIdleTime time.Duration
	MaxLifetime time.Duration
	Timeout     time.Duration
}

type Stats struct {
	Hits           int64
	Misses         int64
	Timeouts       int64
	Waits          int64
	WaitDurationNs int64
	TotalConns     int64
	IdleConns      int64
}

type Pool interface {
	Get(ctx context.Context) (*Conn, error)
	Put(cn *Conn)
	Remove(cn *Conn)
	Stats() Stats
	Close()
}

type puddlePool struct {
	p        *puddle.Pool[*Conn]
	opts     Options
	timeouts atomic.Int64
	done     chan struct{}
	stopOnce sync.Once
	wg       sync.WaitGroup
	mu       sync.Mutex
	active   map[*Conn]struct{}
}

func New(opts Options) Pool {
	p, err := puddle.NewPool(&puddle.Config[*Conn]{
		Constructor: opts.Connect,
		Destructor:  func(cn *Conn) { _ = cn.Close() },
		MaxSize:     int32(min(max(opts.Size, 1), math.MaxInt32)),
	})
	if err != nil {
		panic(fmt.Sprintf("redis: build the connection pool: %v", err))
	}

	pp := &puddlePool{
		p:      p,
		opts:   opts,
		done:   make(chan struct{}),
		active: make(map[*Conn]struct{}),
	}
	pp.wg.Add(1)
	go pp.maintain()

	return pp
}

func (p *puddlePool) Get(ctx context.Context) (*Conn, error) {
	select {
	case <-p.done:
		return nil, ErrClosed
	default:
	}

	acquireCtx, cancel := context.WithTimeout(ctx, p.opts.Timeout)
	defer cancel()

	for {
		res, err := p.p.Acquire(acquireCtx)
		if err != nil {
			return nil, p.acquireError(ctx, acquireCtx, err)
		}
		if p.expired(res) {
			res.Destroy()
			continue
		}
		cn := res.Value()
		cn.res = res
		p.mu.Lock()
		p.active[cn] = struct{}{}
		p.mu.Unlock()
		return cn, nil
	}
}

func (p *puddlePool) acquireError(ctx, acquireCtx context.Context, err error) error {
	switch {
	case errors.Is(err, puddle.ErrClosedPool):
		return ErrClosed
	case ctx.Err() != nil:
		return ctx.Err()
	case acquireCtx.Err() != nil:
		p.timeouts.Add(1)
		return ErrPoolTimeout
	default:
		return err
	}
}

func (p *puddlePool) Put(cn *Conn) {
	res := p.take(cn)
	if res != nil {
		res.Release()
	}
}

func (p *puddlePool) Remove(cn *Conn) {
	res := p.take(cn)
	if res != nil {
		res.Destroy()
	}
}

func (p *puddlePool) take(cn *Conn) *puddle.Resource[*Conn] {
	p.mu.Lock()
	delete(p.active, cn)
	p.mu.Unlock()
	res, _ := cn.res.(*puddle.Resource[*Conn])
	cn.res = nil
	return res
}

func (p *puddlePool) expired(res *puddle.Resource[*Conn]) bool {
	if p.opts.MaxLifetime > 0 && time.Since(res.CreationTime()) > p.opts.MaxLifetime {
		return true
	}
	return p.opts.MaxIdleTime > 0 && res.IdleDuration() > p.opts.MaxIdleTime
}

func (p *puddlePool) Stats() Stats {
	s := p.p.Stat()
	empty := s.EmptyAcquireCount()
	return Stats{
		Hits:           s.AcquireCount() - empty,
		Misses:         empty,
		Timeouts:       p.timeouts.Load(),
		Waits:          empty,
		WaitDurationNs: s.EmptyAcquireWaitTime().Nanoseconds(),
		TotalConns:     int64(s.TotalResources()),
		IdleConns:      int64(s.IdleResources()),
	}
}

func (p *puddlePool) Close() {
	p.stopOnce.Do(func() {
		close(p.done)
		p.wg.Wait()
		p.mu.Lock()
		for cn := range p.active {
			cn.Abort()
		}
		p.mu.Unlock()
		p.p.Close()
	})
}

func (p *puddlePool) maintain() {
	defer p.wg.Done()

	period := maintenancePeriod
	if p.opts.MaxIdleTime > 0 && p.opts.MaxIdleTime < period {
		period = p.opts.MaxIdleTime
	}
	ticker := time.NewTicker(period)
	defer ticker.Stop()

	p.topUp()
	for {
		select {
		case <-p.done:
			return
		case <-ticker.C:
			p.prune()
			p.topUp()
		}
	}
}

func (p *puddlePool) prune() {
	for _, res := range p.p.AcquireAllIdle() {
		if p.expired(res) {
			res.Destroy()
			continue
		}
		res.ReleaseUnused()
	}
}

func (p *puddlePool) topUp() {
	for {
		select {
		case <-p.done:
			return
		default:
		}
		s := p.p.Stat()
		if int(s.IdleResources()+s.ConstructingResources()) >= p.opts.MinIdle ||
			int(s.TotalResources()) >= p.opts.Size {
			return
		}
		if err := p.p.CreateResource(context.Background()); err != nil {
			return
		}
	}
}
