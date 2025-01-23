package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
)

type ListUsStateResult struct {
	States []*usstate.UsState
	Total  int
}

type UsStateRepository interface {
	List(ctx context.Context) (*ListUsStateResult, error)
}

type UsStateCacheRepository interface {
	Get(ctx context.Context) (*ListUsStateResult, error)
	Set(ctx context.Context, states []*usstate.UsState) error
	Invalidate(ctx context.Context) error
}
