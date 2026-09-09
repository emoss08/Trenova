package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tractor"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/tractorservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type tractorsByIDsGetter interface {
	GetByIDs(
		ctx context.Context,
		req repositories.GetTractorsByIDsRequest,
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

func (f *TractorByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *tractor.Tractor] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *TractorByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*tractor.Tractor] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*tractor.Tractor, error) {
		return f.tractors.GetByIDs(ctx, repositories.GetTractorsByIDsRequest{
			TenantInfo: tenantInfo,
			TractorIDs: ids,
		})
	}, "Tractor not found within your organization")
}
