package operationsloaders

import (
	"context"

	"github.com/emoss08/trenova/internal/api/graphql/loaders/loadercore"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentdistanceports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type jurisdictionMilesByMoveIDsLister interface {
	ListByMoveIDs(
		ctx context.Context,
		req shipmentdistanceports.ListJurisdictionMilesByMoveIDsRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
}

type ShipmentMoveJurisdictionMilesByMoveIDLoaderFactoryParams struct {
	fx.In

	Repo shipmentdistanceports.ShipmentMoveJurisdictionMileRepository
}

type ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory struct {
	miles jurisdictionMilesByMoveIDsLister
}

func NewShipmentMoveJurisdictionMilesByMoveIDLoaderFactory(
	p ShipmentMoveJurisdictionMilesByMoveIDLoaderFactoryParams,
) *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory {
	return &ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory{
		miles: p.Repo,
	}
}

func newShipmentMoveJurisdictionMilesByMoveIDLoader(
	f *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory,
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*shipment.ShipmentMoveJurisdictionMile] {
	return loadercore.NewLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) loadercore.BatchFetchFunc[[]*shipment.ShipmentMoveJurisdictionMile] {
	return loadercore.BatchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*shipment.ShipmentMoveJurisdictionMile, error) {
			rows, err := f.miles.ListByMoveIDs(
				ctx,
				shipmentdistanceports.ListJurisdictionMilesByMoveIDsRequest{
					TenantInfo: tenantInfo,
					MoveIDs:    ids,
				},
			)
			if err != nil {
				return nil, err
			}

			groups := make(map[pulid.ID][]*shipment.ShipmentMoveJurisdictionMile, len(ids))
			for _, row := range rows {
				groups[row.ShipmentMoveID] = append(groups[row.ShipmentMoveID], row)
			}
			return groups, nil
		},
	)
}

var shipmentMoveJurisdictionMilesByMoveIDKey = loadercore.NewKey(
	newShipmentMoveJurisdictionMilesByMoveIDLoader,
)

func ShipmentMoveJurisdictionMilesByMoveID(
	ctx context.Context,
	key string,
) ([]*shipment.ShipmentMoveJurisdictionMile, error) {
	return loadercore.Load(ctx, shipmentMoveJurisdictionMilesByMoveIDKey, key)
}

func (f *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory) Attach(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) context.Context {
	return loadercore.Attach(ctx, tenantInfo, f)
}
