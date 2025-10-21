package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListFleetCodeRequest struct {
	Filter                *pagination.QueryOptions `query:"filter"`
	IncludeManagerDetails bool                     `query:"includeManagerDetails"`
}

type GetFleetCodeByIDRequest struct {
	ID                    pulid.ID
	OrgID                 pulid.ID
	BuID                  pulid.ID
	UserID                pulid.ID
	IncludeManagerDetails bool `query:"includeManagerDetails"`
}

type FleetCodeRepository interface {
	List(
		ctx context.Context,
		req *ListFleetCodeRequest,
	) (*pagination.ListResult[*fleetcode.FleetCode], error)
	GetByID(ctx context.Context, req GetFleetCodeByIDRequest) (*fleetcode.FleetCode, error)
	Create(ctx context.Context, l *fleetcode.FleetCode) (*fleetcode.FleetCode, error)
	Update(ctx context.Context, l *fleetcode.FleetCode) (*fleetcode.FleetCode, error)
}
