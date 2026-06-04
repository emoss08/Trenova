package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/location"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/graph-gophers/dataloader/v7"
	"go.uber.org/fx"
)

type LocationByIDLoaderFactoryParams struct {
	fx.In

	LocationRepo repositories.LocationRepository
}

type LocationByIDLoaderFactory struct {
	locationRepo repositories.LocationRepository
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
) *dataloader.Loader[string, *location.Location] {
	return dataloader.NewBatchedLoader(f.batchFunc(tenantInfo))
}

func (f *LocationByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) dataloader.BatchFunc[string, *location.Location] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*location.Location, error) {
		return f.locationRepo.GetByIDs(ctx, repositories.GetLocationsByIDsRequest{
			TenantInfo:  tenantInfo,
			LocationIDs: ids,
		})
	}, "Location not found")
}
