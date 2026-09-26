package ports

import (
	"context"
	"sync"
)

type afterCommitKey struct{}

type AfterCommitHooks struct {
	mu        sync.Mutex
	fns       []func(context.Context)
	done      bool
	committed context.Context
}

func WithAfterCommitHooks(ctx context.Context) (context.Context, *AfterCommitHooks) {
	hooks := &AfterCommitHooks{}
	return context.WithValue(ctx, afterCommitKey{}, hooks), hooks
}

func AfterCommit(ctx context.Context, fn func(context.Context)) {
	if fn == nil || IsReadOnly(ctx) {
		return
	}

	hooks, ok := ctx.Value(afterCommitKey{}).(*AfterCommitHooks)
	if !ok {
		fn(ctx)
		return
	}

	if runCtx, queued := hooks.add(fn); !queued {
		fn(runCtx)
	}
}

func (h *AfterCommitHooks) Run(ctx context.Context) {
	for {
		h.mu.Lock()
		if len(h.fns) == 0 {
			h.done = true
			h.committed = ctx
			h.mu.Unlock()
			return
		}
		fns := h.fns
		h.fns = nil
		h.mu.Unlock()

		for _, fn := range fns {
			fn(ctx)
		}
	}
}

func (h *AfterCommitHooks) add(fn func(context.Context)) (context.Context, bool) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if h.done {
		return h.committed, false
	}

	h.fns = append(h.fns, fn)
	return nil, true
}
