package equipmentloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/equipment/app/tractorservice"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/tractor"
	"github.com/emoss08/trenova/internal/modules/equipment/ports/tractorports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type tractorsByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req tractorports.GetTractorsByIDsRequest,
	) ([]*tractor.Tractor, error)
}

type TractorByIDLoaderFactoryParams struct {
	fx.In

	TractorService *tractorservice.Service
}

type TractorByIDLoaderFactory struct {
	tractors tractorsByIDsGetter
}

func NewTractorByIDLoaderFactory(
	p TractorByIDLoaderFactoryParams,
) *TractorByIDLoaderFactory {
	return &TractorByIDLoaderFactory{
		tractors: p.TractorService,
	}
}

func newTractorByIDLoader(f *TractorByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tractor.Tractor] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *TractorByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*tractor.Tractor] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*tractor.Tractor, error) {
			return f.tractors.GetByIDs(ctx, tractorports.GetTractorsByIDsRequest{
				TenantInfo: tenantInfo,
				TractorIDs: ids,
			})
		},
		"Tractor not found within your organization",
	)
}

var tractorByIDKey = loadercore.NewKey(newTractorByIDLoader)

func TractorByID(ctx context.Context, key string) (*tractor.Tractor, error) {
	return loadercore.Load(ctx, tractorByIDKey, key)
}

func (f *TractorByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
