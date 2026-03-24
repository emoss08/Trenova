package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetUsStateByIDRequest struct {
	StateID pulid.ID `json:"stateId"`
}

type UsStateRepository interface {
	GetByID(
		ctx context.Context,
		req GetUsStateByIDRequest,
	) (*usstate.UsState, error)
	SelectOptions(
		ctx context.Context,
		req *pagination.SelectQueryRequest,
	) (*pagination.ListResult[*usstate.UsState], error)
	GetByAbbreviation(
		ctx context.Context,
		abbreviation string,
	) (*usstate.UsState, error)
}

type UsStateCacheRepository interface {
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
	Set(ctx context.Context, states []*usstate.UsState) error
	Invalidate(ctx context.Context) error
}
