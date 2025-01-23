package repositories

import (
	"context"

	"github.com/trenova-app/transport/internal/core/domain/location"
	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/trenova-app/transport/pkg/types/pulid"
)

type ListLocationOptions struct {
	Filter          *ports.LimitOffsetQueryOptions
	IncludeCategory bool `query:"includeCategory"`
	IncludeState    bool `query:"includeState"`
}

type GetLocationByIDOptions struct {
	ID              pulid.ID
	OrgID           pulid.ID
	BuID            pulid.ID
	UserID          pulid.ID
	IncludeCategory bool `query:"includeCategory"`
	IncludeState    bool `query:"includeState"`
}

type LocationRepository interface {
	List(ctx context.Context, opts *ListLocationOptions) (*ports.ListResult[*location.Location], error)
	GetByID(ctx context.Context, opts GetLocationByIDOptions) (*location.Location, error)
	Create(ctx context.Context, l *location.Location) (*location.Location, error)
	Update(ctx context.Context, l *location.Location) (*location.Location, error)
}
