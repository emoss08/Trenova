package loaders

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/vikstrous/dataloadgen"
	"go.uber.org/fx"
)

type jurisdictionMilesByMoveIDsLister interface {
	ListByMoveIDs(
		ctx context.Context,
		req shipmentports.ListJurisdictionMilesByMoveIDsRequest,
	) ([]*shipment.ShipmentMoveJurisdictionMile, error)
}

type ShipmentMoveJurisdictionMilesByMoveIDLoaderFactoryParams struct {
	fx.In

	Repo shipmentports.ShipmentMoveJurisdictionMileRepository
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

func (f *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory) NewForTenant(
	tenantInfo pagination.TenantInfo,
) *dataloadgen.Loader[string, []*shipment.ShipmentMoveJurisdictionMile] {
	return newLoader(f.batchFunc(tenantInfo))
}

func (f *ShipmentMoveJurisdictionMilesByMoveIDLoaderFactory) batchFunc(
	tenantInfo pagination.TenantInfo,
) batchFetchFunc[[]*shipment.ShipmentMoveJurisdictionMile] {
	return batchGroupFunc(
		func(ctx context.Context, ids []pulid.ID) (map[pulid.ID][]*shipment.ShipmentMoveJurisdictionMile, error) {
			rows, err := f.miles.ListByMoveIDs(
				ctx,
				shipmentports.ListJurisdictionMilesByMoveIDsRequest{
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
