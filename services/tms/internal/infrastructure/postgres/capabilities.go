package postgres

import (
	"context"
	"sync"
	"time"

	"github.com/emoss08/trenova/pkg/dbdialect"
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
	current  dbdialect.Capabilities
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
	caps, err := p.Refresh(ctx)
	if err != nil {
		p.logger.Warn(
			"database capability probe failed; retrying on first use",
			zap.Error(err),
		)
		return
	}

	vector := caps.Vector()
	fields := []zap.Field{
		zap.String("driver", caps.Kind().String()),
		zap.Bool("vector_search", caps.Supports(dbdialect.CapVectorSearch)),
		zap.String("vector_state", string(vector.State)),
	}
	if vector.ExtensionVersion != "" {
		fields = append(fields, zap.String("pgvector_version", vector.ExtensionVersion))
	}

	p.logger.Info("database capabilities probed", fields...)
}

func (p *CapabilityProbe) Capabilities(ctx context.Context) (dbdialect.Capabilities, error) {
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

func (p *CapabilityProbe) Supports(ctx context.Context, capability dbdialect.Capability) bool {
	caps, err := p.Capabilities(ctx)
	if err != nil {
		p.logger.Warn(
			"database capability probe failed; treating capability as absent",
			zap.String("capability", string(capability)),
			zap.Error(err),
		)
		return false
	}

	return caps.Supports(capability)
}

func (p *CapabilityProbe) Refresh(ctx context.Context) (dbdialect.Capabilities, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	return p.probeLocked(ctx)
}

func (p *CapabilityProbe) probeLocked(ctx context.Context) (dbdialect.Capabilities, error) {
	db := p.db.DB()
	if db == nil {
		return dbdialect.Capabilities{}, ErrDatabaseConnectionNotInitialized
	}

	caps, err := dbdialect.ProbeCapabilities(ctx, db)
	if err != nil {
		return dbdialect.Capabilities{}, err
	}

	p.current = caps
	p.probedAt = p.now()
	p.probed = true

	return caps, nil
}

func (p *CapabilityProbe) freshLocked() bool {
	if !p.probed {
		return false
	}

	if p.current.Supports(dbdialect.CapVectorSearch) {
		return true
	}

	return p.now().Sub(p.probedAt) < unavailableRecheckInterval
}
