package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/services/fuelpurchaseservice"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type fuelCardsByIDsGetter interface {
	GetCardsByIDs(
		ctx context.Context,
		req *repositories.GetFuelCardsByIDsRequest,
	) ([]*fuelpurchase.FuelCard, error)
}

type FuelCardByIDLoaderFactoryParams struct {
	fx.In

	FuelPurchaseService *fuelpurchaseservice.Service
}

type FuelCardByIDLoaderFactory struct {
	cards fuelCardsByIDsGetter
}

func NewFuelCardByIDLoaderFactory(
	p FuelCardByIDLoaderFactoryParams,
) *FuelCardByIDLoaderFactory {
	return &FuelCardByIDLoaderFactory{
		cards: p.FuelPurchaseService,
	}
}

func (f *FuelCardByIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *fuelpurchase.FuelCard] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *FuelCardByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[*fuelpurchase.FuelCard] {
	return batchByIDFunc(func(ctx context.Context, ids []pulid.ID) ([]*fuelpurchase.FuelCard, error) {
		return f.cards.GetCardsByIDs(ctx, &repositories.GetFuelCardsByIDsRequest{
			TenantInfo: tenantInfo,
			IDs:        ids,
		})
	}, "Fuel card not found within your organization")
}
