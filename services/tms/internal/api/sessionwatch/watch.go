package sessionwatch

import (
	"context"
	"errors"
	"slices"
	"sync"
	"time"

	"github.com/emoss08/trenova/internal/infrastructure/config/realtimeconfig"
	"github.com/emoss08/trenova/internal/modules/identity/ports/sessionports"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

var (
	ErrSessionEnded   = errors.New("session ended")
	ErrSessionRotated = errors.New("session secret rotated")
)

func Ended(ctx context.Context) bool {
	cause := context.Cause(ctx)
	return errors.Is(cause, ErrSessionEnded) || errors.Is(cause, ErrSessionRotated)
}

type Params struct {
	fx.In

	Lifecycle      fx.Lifecycle
	Liveness       sessionports.SessionLiveness
	RealtimeConfig *realtimeconfig.Config
	Logger         *zap.Logger
}

type watched struct {
	done     chan struct{}
	cause    error
	watchers int
}

type Ending struct {
	entry *watched
}

func (e Ending) Done() <-chan struct{} {
	if e.entry == nil {
		return nil
	}
	return e.entry.done
}

func (e Ending) Cause() error {
	if e.entry == nil {
		return nil
	}
	return e.entry.cause
}

type Watch struct {
	liveness sessionports.SessionLiveness
	interval time.Duration
	l        *zap.Logger

	mu       sync.Mutex
	sessions map[pulid.ID]map[string]*watched
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
		sessions: make(map[pulid.ID]map[string]*watched),
	}
}

func (w *Watch) Watch(sessionID pulid.ID, secretHash string) (ending Ending, stop func()) {
	if sessionID.IsNil() {
		return Ending{}, func() {}
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	bySecret, ok := w.sessions[sessionID]
	if !ok {
		bySecret = make(map[string]*watched)
		w.sessions[sessionID] = bySecret
	}
	entry, ok := bySecret[secretHash]
	if !ok {
		entry = &watched{done: make(chan struct{})}
		bySecret[secretHash] = entry
	}
	entry.watchers++

	var once sync.Once
	return Ending{entry: entry}, func() {
		once.Do(func() { w.release(sessionID, secretHash, entry) })
	}
}

func (w *Watch) Context(
	parent context.Context,
	sessionID pulid.ID,
	secretHash string,
) (context.Context, context.CancelFunc) {
	ending, stop := w.Watch(sessionID, secretHash)
	ctx, cancel := context.WithCancelCause(parent)
	if done := ending.Done(); done != nil {
		go func() {
			select {
			case <-done:
				cancel(ending.Cause())
			case <-ctx.Done():
			}
		}()
	}

	return ctx, func() {
		cancel(context.Canceled)
		stop()
	}
}

func (w *Watch) SessionContext(
	parent context.Context,
	sessionID pulid.ID,
) (context.Context, context.CancelFunc) {
	return w.Context(parent, sessionID, "")
}

func (w *Watch) release(sessionID pulid.ID, secretHash string, entry *watched) {
	w.mu.Lock()
	defer w.mu.Unlock()

	entry.watchers--
	bySecret := w.sessions[sessionID]
	if entry.watchers > 0 || bySecret[secretHash] != entry {
		return
	}
	delete(bySecret, secretHash)
	if len(bySecret) == 0 {
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

	standings, err := w.liveness.Standings(checkCtx, sessionIDs)
	if err != nil {
		w.l.Warn(
			"could not check whether open streams' sessions have ended; checking again next sweep",
			zap.Int("sessions", len(sessionIDs)),
			zap.Error(err),
		)
		return
	}

	w.settle(standings)
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

func (w *Watch) settle(standings []sessionports.SessionStanding) {
	w.mu.Lock()
	defer w.mu.Unlock()

	for _, standing := range standings {
		bySecret, ok := w.sessions[standing.ID]
		if !ok {
			continue
		}
		for secretHash, entry := range bySecret {
			cause := endingCause(standing, secretHash)
			if cause == nil {
				continue
			}
			entry.cause = cause
			close(entry.done)
			delete(bySecret, secretHash)
		}
		if len(bySecret) == 0 {
			delete(w.sessions, standing.ID)
		}
	}
}

func endingCause(standing sessionports.SessionStanding, secretHash string) error {
	if standing.Ended {
		return ErrSessionEnded
	}
	if secretHash != "" && standing.SecretsKnown &&
		!slices.Contains(standing.SecretHashes, secretHash) {
		return ErrSessionRotated
	}
	return nil
}
