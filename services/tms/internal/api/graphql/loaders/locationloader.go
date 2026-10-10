package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/modules/places/ports/locationports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type LocationByIDLoaderFactoryParams struct {
	fx.In

	LocationRepo locationports.LocationRepository
}

type LocationByIDLoaderFactory struct {
	locationRepo locationports.LocationRepository
}

func NewLocationByIDLoaderFactory(
	p LocationByIDLoaderFactoryParams,
) *LocationByIDLoaderFactory {
	return &LocationByIDLoaderFactory{
		locationRepo: p.LocationRepo,
	}
}

func (f *LocationByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *location.Location] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *LocationByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*location.Location] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*location.Location, error) {
		return f.locationRepo.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
			TenantInfo:  tenantInfo,
			LocationIDs: ids,
		})
	}, "Location not found")
}
