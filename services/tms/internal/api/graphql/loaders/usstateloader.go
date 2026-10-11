package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/usstate"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/usstateports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type usStatesByIDsGetter interface {
	GetByIDs(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error)
}

type UsStateByIDLoaderFactoryParams struct {
	fx.In

	Repo usstateports.UsStateRepository
}

type UsStateByIDLoaderFactory struct {
	states usStatesByIDsGetter
}

func NewUsStateByIDLoaderFactory(p UsStateByIDLoaderFactoryParams) *UsStateByIDLoaderFactory {
	return &UsStateByIDLoaderFactory{
		states: p.Repo,
	}
}

func newUsStateByIDLoader(f *UsStateByIDLoaderFactory,
	_ pagination.TenantInfo,
) *dataloadgen.Loader[string, *usstate.UsState] {
	return newLoader(f.batchFunc())
}

func (f *UsStateByIDLoaderFactory) batchFunc() batchFetchFunc[*usstate.UsState] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
		return f.states.GetByIDs(ctx, ids)
	}, "State not found")
}
