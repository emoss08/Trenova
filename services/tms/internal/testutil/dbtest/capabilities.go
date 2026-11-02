package dbtest

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// unstarted is a lifecycle that never runs its hooks.
type unstarted struct{}

func (unstarted) Append(fx.Hook) {}

// CapabilityProbe builds the probe through its constructor without starting it, so it reads
// the database's capabilities the first time a test asks.
func CapabilityProbe(conn *postgres.Connection) *postgres.CapabilityProbe {
	return postgres.NewCapabilityProbe(postgres.CapabilityProbeParams{
		Lifecycle: unstarted{},
		DB:        conn,
		Logger:    zap.NewNop(),
	})
}
