package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type ListFleetCodeOptions struct {
	Filter                *ports.LimitOffsetQueryOptions
	IncludeManagerDetails bool `query:"includeManagerDetails"`
}

type GetFleetCodeByIDOptions struct {
	ID                    pulid.ID
	OrgID                 pulid.ID
	BuID                  pulid.ID
	UserID                pulid.ID
	IncludeManagerDetails bool `query:"includeManagerDetails"`
}

type FleetCodeRepository interface {
	List(ctx context.Context, opts *ListFleetCodeOptions) (*ports.ListResult[*fleetcode.FleetCode], error)
	GetByID(ctx context.Context, opts GetFleetCodeByIDOptions) (*fleetcode.FleetCode, error)
	Create(ctx context.Context, fc *fleetcode.FleetCode) (*fleetcode.FleetCode, error)
	Update(ctx context.Context, fc *fleetcode.FleetCode) (*fleetcode.FleetCode, error)
}
