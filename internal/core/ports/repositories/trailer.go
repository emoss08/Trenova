package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/trailer"
	"github.com/emoss08/trenova/internal/core/ports"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type TrailerFilterOptions struct {
	IncludeEquipmentDetails bool   `query:"includeEquipmentDetails"`
	IncludeFleetDetails     bool   `query:"includeFleetDetails"`
	Status                  string `query:"status"`
}

type ListTrailerOptions struct {
	Filter        *ports.LimitOffsetQueryOptions
	FilterOptions TrailerFilterOptions `query:"filterOptions"`
}

type GetTrailerByIDOptions struct {
	ID            pulid.ID
	OrgID         pulid.ID
	BuID          pulid.ID
	UserID        pulid.ID
	FilterOptions TrailerFilterOptions `query:"filterOptions"`
}

type TrailerRepository interface {
	List(ctx context.Context, opts *ListTrailerOptions) (*ports.ListResult[*trailer.Trailer], error)
	GetByID(ctx context.Context, opts *GetTrailerByIDOptions) (*trailer.Trailer, error)
	Create(ctx context.Context, t *trailer.Trailer) (*trailer.Trailer, error)
	Update(ctx context.Context, t *trailer.Trailer) (*trailer.Trailer, error)
}
