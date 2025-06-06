package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type DedicatedLaneFilterOptions struct {
	ExpandDetails bool `query:"expandDetails"`
}

type ListDedicatedLaneRequest struct {
	Filter        *ports.LimitOffsetQueryOptions
	FilterOptions DedicatedLaneFilterOptions `query:"filterOptions"`
}

type GetDedicatedLaneByIDRequest struct {
	ID            pulid.ID
	OrgID         pulid.ID
	BuID          pulid.ID
	UserID        pulid.ID
	FilterOptions DedicatedLaneFilterOptions `query:"filterOptions"`
}

type DedicatedLaneRepository interface {
	List(
		ctx context.Context,
		req *ListDedicatedLaneRequest,
	) (*ports.ListResult[*dedicatedlane.DedicatedLane], error)
	GetByID(
		ctx context.Context,
		req *GetDedicatedLaneByIDRequest,
	) (*dedicatedlane.DedicatedLane, error)
	Create(
		ctx context.Context,
		dl *dedicatedlane.DedicatedLane,
	) (*dedicatedlane.DedicatedLane, error)
	Update(
		ctx context.Context,
		dl *dedicatedlane.DedicatedLane,
	) (*dedicatedlane.DedicatedLane, error)
}
