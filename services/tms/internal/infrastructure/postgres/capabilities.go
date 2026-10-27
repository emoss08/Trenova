package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pgvector"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const unavailableRecheckInterval = 5 * time.Minute

type CapabilityProbeParams struct {
	fx.In

	Lifecycle fx.Lifecycle
	DB        *Connection
	Logger    *zap.Logger
}

type CapabilityProbe struct {
	db     *Connection
	logger *zap.Logger
	now    func() time.Time

	mu       sync.RWMutex
	current  pgvector.VectorSupport
	probedAt time.Time
	probed   bool
}

func NewCapabilityProbe(p CapabilityProbeParams) *CapabilityProbe {
	probe := newCapabilityProbe(p.DB, p.Logger)

	p.Lifecycle.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			probe.startup(ctx)
			return nil
		},
	})

	return probe
}

func NewTestCapabilityProbe(db *Connection) *CapabilityProbe {
	return newCapabilityProbe(db, zap.NewNop())
}

func newCapabilityProbe(db *Connection, logger *zap.Logger) *CapabilityProbe {
	return &CapabilityProbe{
		db:     db,
		logger: logger.Named("postgres.capability-probe"),
		now:    time.Now,
	}
}

func (p *CapabilityProbe) startup(ctx context.Context) {
	vector, err := p.Refresh(ctx)
	if err != nil {
		p.logger.Warn(
			"database capability probe failed; retrying on first use",
			zap.Error(err),
		)
		return
	}

	fields := []zap.Field{
		zap.Bool("vector_search", vector.Ready()),
		zap.String("vector_state", string(vector.State)),
	}
	if vector.ExtensionVersion != "" {
		fields = append(fields, zap.String("pgvector_version", vector.ExtensionVersion))
	}

	p.logger.Info("database capabilities probed", fields...)
}

func (p *CapabilityProbe) VectorSupport(ctx context.Context) (pgvector.VectorSupport, error) {
	p.mu.RLock()
	current, fresh := p.current, p.freshLocked()
	p.mu.RUnlock()

	if fresh {
		return current, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if p.freshLocked() {
		return p.current, nil
	}

	return p.probeLocked(ctx)
}

func (p *CapabilityProbe) VectorReady(ctx context.Context) bool {
	vector, err := p.VectorSupport(ctx)
	if err != nil {
		p.logger.Warn(
			"database capability probe failed; treating vector search as absent",
			zap.Error(err),
		)
		return false
	}

	return vector.Ready()
}

func (p *CapabilityProbe) Refresh(ctx context.Context) (pgvector.VectorSupport, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.probeLocked(ctx)
}

const capabilityProbeReason = "probe the database server's installed extensions"

func (p *CapabilityProbe) probeLocked(ctx context.Context) (pgvector.VectorSupport, error) {
	if p.db.Keel(ctx) == nil {
		return pgvector.VectorSupport{}, ErrDatabaseConnectionNotInitialized
	}

	ctx = dbscope.WithSystem(ctx, capabilityProbeReason)
	vector, err := pgvector.ProbeVector(ctx, p.db.Keel(ctx))
	if err != nil {
		return pgvector.VectorSupport{}, err
	}

	p.current = vector
	p.probedAt = p.now()
	p.probed = true

	return vector, nil
}

func (p *CapabilityProbe) freshLocked() bool {
	if !p.probed {
		return false
	}

	if p.current.Ready() {
		return true
	}

	return p.now().Sub(p.probedAt) < unavailableRecheckInterval
}
