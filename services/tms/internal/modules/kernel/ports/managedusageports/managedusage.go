package managedusageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
)

const (
	CounterGroup = `group:"managed_hourly_counters"`
	LevelGroup   = `group:"managed_level_readers"`
)

type StateRepository interface {
	Get(ctx context.Context) (*managedusage.State, error)
	Save(ctx context.Context, state *managedusage.State) error
}

type HourCountRequest struct {
	Since int64
	Until int64
}

type HourCount struct {
	Hour     int64
	Quantity int64
}

type HourlyCounter interface {
	Meter() platformcatalog.MeterKey
	CountByHour(ctx context.Context, req *HourCountRequest) ([]HourCount, error)
}

type LevelReader interface {
	Meter() platformcatalog.MeterKey
	Level(ctx context.Context) (int64, error)
}

type Gate interface {
	Decide(
		ctx context.Context,
		meter platformcatalog.MeterKey,
		requested int64,
	) (managedusage.Decision, error)
	State(ctx context.Context) (*managedusage.State, error)
}
