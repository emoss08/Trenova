package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type ListLocationRequest struct {
	Filter          *pagination.QueryOptions `query:"filter"`
	IncludeCategory bool                     `query:"includeCategory"`
	IncludeState    bool                     `query:"includeState"`
	Status          string                   `query:"status"`
}

type GetLocationByIDRequest struct {
	ID              pulid.ID
	OrgID           pulid.ID
	BuID            pulid.ID
	UserID          pulid.ID
	IncludeCategory bool `query:"includeCategory"`
	IncludeState    bool `query:"includeState"`
}

type LocationRepository interface {
	List(
		ctx context.Context,
		req *ListLocationRequest,
	) (*pagination.ListResult[*location.Location], error)
	GetByID(ctx context.Context, req GetLocationByIDRequest) (*location.Location, error)
	Create(ctx context.Context, l *location.Location) (*location.Location, error)
	Update(ctx context.Context, l *location.Location) (*location.Location, error)
}
