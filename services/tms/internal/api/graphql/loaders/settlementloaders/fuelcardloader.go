package settlementloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/settlement/app/fuelpurchaseservice"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/fuelpurchase"
	"github.com/emoss08/trenova/internal/modules/settlement/ports/fuelpurchaseports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type fuelCardsByIDsGetter interface {
	GetCardsByIDs(
		ctx context.Context,
		req *fuelpurchaseports.GetFuelCardsByIDsRequest,
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

func newFuelCardByIDLoader(f *FuelCardByIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *fuelpurchase.FuelCard] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *FuelCardByIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*fuelpurchase.FuelCard] {
	return loadercore.BatchByIDFunc(
		func(ctx context.Context, ids []pulid.ID) ([]*fuelpurchase.FuelCard, error) {
			return f.cards.GetCardsByIDs(ctx, &fuelpurchaseports.GetFuelCardsByIDsRequest{
				TenantInfo: tenantInfo,
				IDs:        ids,
			})
		},
		"Fuel card not found within your organization",
	)
}

var fuelCardByIDKey = loadercore.NewKey(newFuelCardByIDLoader)

func FuelCardByID(ctx context.Context, key string) (*fuelpurchase.FuelCard, error) {
	return loadercore.Load(ctx, fuelCardByIDKey, key)
}

func (f *FuelCardByIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
