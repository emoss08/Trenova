package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type TrailerFilterOptions struct {
	IncludeEquipmentDetails bool   `query:"includeEquipmentDetails"`
	IncludeFleetDetails     bool   `query:"includeFleetDetails"`
	Status                  string `query:"status"`
}

type ListTrailerRequest struct {
	Filter        *pagination.QueryOptions
	FilterOptions TrailerFilterOptions `query:"filterOptions"`
}

type GetTrailerByIDRequest struct {
	ID            pulid.ID
	OrgID         pulid.ID
	BuID          pulid.ID
	UserID        pulid.ID
	FilterOptions TrailerFilterOptions `query:"filterOptions"`
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
