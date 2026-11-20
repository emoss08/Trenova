package managedusageports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedscope"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/managedusage"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	CounterGroup  = `group:"managed_hourly_counters"`
	LevelGroup    = `group:"managed_level_readers"`
	ObserverGroup = `group:"managed_state_observers"`
)

type StateRepository interface {
	Get(ctx context.Context, scope managedscope.Scope) (*managedusage.State, error)
	Save(ctx context.Context, scope managedscope.Scope, state *managedusage.State) error
}

type HourCountRequest struct {
	Scope managedscope.Scope
	Since int64
	Until int64
}

type HourCount struct {
	Hour     int64
	Quantity int64
}

type OrganizationsHourCountRequest struct {
	OrganizationIDs []pulid.ID
	Since           int64
	Until           int64
}

type OrganizationHourCount struct {
	OrganizationID pulid.ID
	Hour           int64
	Quantity       int64
}

type OrganizationLevel struct {
	OrganizationID pulid.ID
	Quantity       int64
}

type HourlyCounter interface {
	Meter() platformcatalog.MeterKey
	CountByHour(ctx context.Context, req *HourCountRequest) ([]HourCount, error)
	CountByHourForOrganizations(
		ctx context.Context,
		req *OrganizationsHourCountRequest,
	) ([]OrganizationHourCount, error)
}

type LevelReader interface {
	Meter() platformcatalog.MeterKey
	Level(ctx context.Context, scope managedscope.Scope) (int64, error)
	LevelsForOrganizations(
		ctx context.Context,
		organizationIDs []pulid.ID,
	) ([]OrganizationLevel, error)
}

type MeterReading struct {
	Meter    platformcatalog.MeterKey
	Hour     int64
	Quantity int64
}

type Writer interface {
	SetState(ctx context.Context, scope managedscope.Scope, state *managedusage.State) error
}

type OrganizationReadings interface {
	ReadingsForOrganizations(
		ctx context.Context,
		organizationIDs []pulid.ID,
		since int64,
	) (map[pulid.ID][]MeterReading, error)
}

type Gate interface {
	Decide(
		ctx context.Context,
		scope managedscope.Scope,
		meter platformcatalog.MeterKey,
		requested int64,
	) (managedusage.Decision, error)
	State(ctx context.Context, scope managedscope.Scope) (*managedusage.State, error)
}

type UsageReader interface {
	Usage(ctx context.Context, scope managedscope.Scope) (*managedusage.Usage, error)
}

type StateChange struct {
	Scope    managedscope.Scope
	Previous *managedusage.State
	Current  *managedusage.State
}

type StateObserver interface {
	StateChanged(ctx context.Context, change *StateChange) error
}
