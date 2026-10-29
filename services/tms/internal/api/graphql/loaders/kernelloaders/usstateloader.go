package kernelloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
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
	return loadercore.NewLoader(f.batchFunc())
}

func (f *UsStateByIDLoaderFactory) batchFunc() loadercore.BatchFetchFunc[*usstate.UsState] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*usstate.UsState, error) {
			return f.states.GetByIDs(ctx, ids)
		},
		"State not found",
	)
}

var usStateByIDKey = loadercore.NewKey(newUsStateByIDLoader)

func UsStateByID(ctx context.Context, key string) (*usstate.UsState, error) {
	return loadercore.Load(ctx, usStateByIDKey, key)
}

func (f *UsStateByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
