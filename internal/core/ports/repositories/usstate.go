package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/core/ports"
)

type UsStateRepository interface {
	List(ctx context.Context) (*ports.ListResult[*usstate.UsState], error)
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
}

type UsStateCacheRepository interface {
	Get(ctx context.Context) (*ports.ListResult[*usstate.UsState], error)
	GetByAbbreviation(ctx context.Context, abbreviation string) (*usstate.UsState, error)
	Set(ctx context.Context, states []*usstate.UsState) error
	Invalidate(ctx context.Context) error
}
