package placesloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/places/domain/location"
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

func newLocationByIDLoader(f *LocationByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *location.Location] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *LocationByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*location.Location] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*location.Location, error) {
			return f.locationRepo.GetByIDs(ctx, locationports.GetLocationsByIDsRequest{
				TenantInfo:  tenantInfo,
				LocationIDs: ids,
			})
		},
		"Location not found",
	)
}

var locationByIDKey = loadercore.NewKey(newLocationByIDLoader)

func LocationByID(ctx context.Context, key string) (*location.Location, error) {
	return loadercore.Load(ctx, locationByIDKey, key)
}

func (f *LocationByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
