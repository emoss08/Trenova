package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fleetcode"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListFleetCodesRequest struct {
	Filter                *pagination.QueryOptions `json:"filter"`
	IncludeManagerDetails bool                     `json:"includeManagerDetails"`
}

type GetFleetCodeByIDRequest struct {
	ID         pulid.ID               `json:"id"`
	TenantInfo *pagination.TenantInfo `json:"-"`
}

type FleetCodeRepository interface {
	List(
		ctx context.Context,
		req *ListFleetCodesRequest,
	) (*pagination.ListResult[*fleetcode.FleetCode], error)
	GetByID(
		ctx context.Context,
		req GetFleetCodeByIDRequest,
	) (*fleetcode.FleetCode, error)
	Create(
		ctx context.Context,
		entity *fleetcode.FleetCode,
	) (*fleetcode.FleetCode, error)
	Update(
		ctx context.Context,
		entity *fleetcode.FleetCode,
	) (*fleetcode.FleetCode, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*fleetcode.FleetCode], error)
}
