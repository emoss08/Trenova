package operationsloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmenttrackingports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type ShipmentEtaLoaderFactoryParams struct {
	fx.In

	Etas shipmenttrackingports.ShipmentEtaReader
}

type ShipmentEtaLoaderFactory struct {
	etas shipmenttrackingports.ShipmentEtaReader
}

func NewShipmentEtaLoaderFactory(p ShipmentEtaLoaderFactoryParams) *ShipmentEtaLoaderFactory {
	return &ShipmentEtaLoaderFactory{etas: p.Etas}
}

func newShipmentEtaLoader(f *ShipmentEtaLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, *shipmenttrackingports.ShipmentEta] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentEtaLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[*shipmenttrackingports.ShipmentEta] {
	return func(ctx context.Context, keys []string) ([]*shipmenttrackingports.ShipmentEta, []error) {
		values := make([]*shipmenttrackingports.ShipmentEta, len(keys))
		errs := make([]error, len(keys))

		ids, indexesByID := loadercore.ParseBatchKeys(keys, errs)
		if len(ids) == 0 {
			return values, errs
		}

		etas, err := f.etas.EtasByShipmentIDs(ctx, tenantInfo, ids)
		if err != nil {
			loadercore.FillMissingErrors(errs, err)
			return values, errs
		}

		for _, id := range ids {
			for _, idx := range indexesByID[id] {
				values[idx] = etas[id]
			}
		}

		return values, errs
	}
}

var shipmentEtaByIDKey = loadercore.NewKey(newShipmentEtaLoader)

func ShipmentEtaByID(ctx context.Context, key string) (*shipmenttrackingports.ShipmentEta, error) {
	return loadercore.Load(ctx, shipmentEtaByIDKey, key)
}

func (f *ShipmentEtaLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
