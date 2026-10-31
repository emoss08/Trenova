package sessionwatch

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var ErrSessionEnded = errors.New("session ended")

type Params struct {
	fx.In

	Lifecycle      fx.Lifecycle
	Liveness       sessionports.SessionLiveness
	RealtimeConfig *realtimeconfig.Config
	Logger         *zap.Logger
}

type watched struct {
	ended    chan struct{}
	watchers int
}

type Watch struct {
	liveness sessionports.SessionLiveness
	interval time.Duration
	l        *zap.Logger

	mu       sync.Mutex
	sessions map[pulid.ID]*watched
}

func New(p Params) *Watch {
	w := NewWatch(
		p.Liveness,
		p.RealtimeConfig.GetSessionCheckInterval(),
		p.Logger.Named("api.session-watch"),
	)

	ctx, cancel := context.WithCancel(context.Background())
	var wg sync.WaitGroup
	p.Lifecycle.Append(fx.Hook{
		OnStart: func(context.Context) error {
			wg.Go(func() { w.run(ctx) })
			return nil
		},
		OnStop: func(context.Context) error {
			cancel()
			wg.Wait()
			return nil
		},
	})

	return w
}

func NewWatch(
	liveness sessionports.SessionLiveness,
	interval time.Duration,
	logger *zap.Logger,
) *Watch {
	return &Watch{
		liveness: liveness,
		interval: interval,
		l:        logger,
		sessions: make(map[pulid.ID]*watched),
	}
}

func (w *Watch) Watch(sessionID pulid.ID) (<-chan struct{}, func()) {
	if sessionID.IsNil() {
		return nil, func() {}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	entry, ok := w.sessions[sessionID]
	if !ok {
		entry = &watched{ended: make(chan struct{})}
		w.sessions[sessionID] = entry
	}
	entry.watchers++

	var once sync.Once
	return entry.ended, func() {
		once.Do(func() { w.release(sessionID, entry) })
	}
}

func (w *Watch) Context(
	parent context.Context,
	sessionID pulid.ID,
) (context.Context, context.CancelFunc) {
	ended, stop := w.Watch(sessionID)
	ctx, cancel := context.WithCancelCause(parent)
	if ended != nil {
		go func() {
			select {
			case <-ended:
				cancel(ErrSessionEnded)
			case <-ctx.Done():
			}
		}()
	}

	return ctx, func() {
		cancel(context.Canceled)
		stop()
	}
}

func (w *Watch) release(sessionID pulid.ID, entry *watched) {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry.watchers--
	if entry.watchers == 0 && w.sessions[sessionID] == entry {
		delete(w.sessions, sessionID)
	}
}

func (w *Watch) run(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			w.Sweep(ctx)
		}
	}
}

func (w *Watch) Sweep(ctx context.Context) {
	sessionIDs := w.watchedIDs()
	if len(sessionIDs) == 0 {
		return
	}

	checkCtx, cancel := context.WithTimeout(ctx, w.interval)
	defer cancel()

	ended, err := w.liveness.Ended(checkCtx, sessionIDs)
	if err != nil {
		w.l.Warn(
			"could not check whether open streams' sessions have ended; checking again next sweep",
			zap.Int("sessions", len(sessionIDs)),
			zap.Error(err),
		)
		return
	}

	w.end(ended)
}

func (w *Watch) watchedIDs() []pulid.ID {
	w.mu.Lock()
	defer w.mu.Unlock()

	sessionIDs := make([]pulid.ID, 0, len(w.sessions))
	for sessionID := range w.sessions {
		sessionIDs = append(sessionIDs, sessionID)
	}
	return sessionIDs
}

func (w *Watch) end(sessionIDs []pulid.ID) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, sessionID := range sessionIDs {
		if entry, ok := w.sessions[sessionID]; ok {
			close(entry.ended)
			delete(w.sessions, sessionID)
		}
	}
}
