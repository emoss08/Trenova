package shipmentrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/domain/order/orderdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/shared/sliceutils"
	"go.uber.org/zap"
)

const summaryOrderAlias = "sum_ord"

func (r *repository) ListSummariesByIDs(
	ctx context.Context,
	req *shipmentports.ListShipmentSummariesRequest,
) ([]*shipmentports.ShipmentSummary, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*shipmentports.ShipmentSummary, error) {
			if len(req.ShipmentIDs) == 0 {
				return nil, nil
			}

			sp := shipmentdb.Shipments
			ord := orderdb.Orders.As(summaryOrderAlias)
			origin := firstPickupPlace()
			dest := lastDeliveryPlace()

			q := dest.join(origin.join(sp.LeftJoin(ord, ord.ID.EqCol(sp.OrderID)))).
				Pick(
					sp.ID,
					sp.ProNumber.Coalesce(""),
					sp.BOL.Coalesce(""),
					ord.PONumber.Coalesce(""),
					sp.ActualDeliveryDate,
					origin.city(),
					origin.stateCode(),
					dest.city(),
					dest.stateCode(),
					sp.TotalChargeAmount,
				).
				Where(sp.ID.In(req.ShipmentIDs...)).
				Order(sp.ActualDeliveryDate.Asc().NullsLast(), sp.ProNumber.Asc())

			summaries, err := keel.Into[shipmentports.ShipmentSummary](ctx, r.db.Keel(ctx), q)
			if err != nil {
				r.l.Error("failed to list shipment summaries", zap.Error(err))
				return nil, fmt.Errorf("list shipment summaries: %w", err)
			}

			return sliceutils.Pointers(summaries), nil
		},
	)
}
