package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type TrailerFilterOptions struct {
	IncludeEquipmentDetails bool   `form:"includeEquipmentDetails" json:"includeEquipmentDetails"`
	IncludeFleetDetails     bool   `form:"includeFleetDetails"     json:"includeFleetDetails"`
	Status                  string `form:"status"                  json:"status"`
}

type ListTrailerRequest struct {
	Filter        *pagination.QueryOptions
	FilterOptions TrailerFilterOptions `form:"filterOptions" json:"filterOptions"`
}

type GetTrailerByIDRequest struct {
	ID            pulid.ID             `form:"id"            json:"id"`
	OrgID         pulid.ID             `form:"orgId"         json:"orgId"`
	BuID          pulid.ID             `form:"buId"          json:"buId"`
	UserID        pulid.ID             `form:"userId"        json:"userId"`
	FilterOptions TrailerFilterOptions `form:"filterOptions" json:"filterOptions"`
}

type TrailerRepository interface {
	List(
		ctx context.Context,
		opts *ListTrailerRequest,
	) (*pagination.ListResult[*trailer.Trailer], error)
	GetByID(
		ctx context.Context,
		opts *GetTrailerByIDRequest,
	) (*trailer.Trailer, error)
	Create(ctx context.Context, t *trailer.Trailer) (*trailer.Trailer, error)
	Update(ctx context.Context, t *trailer.Trailer) (*trailer.Trailer, error)
}
