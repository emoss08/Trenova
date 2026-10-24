package shipmentboardrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/operations/adapters/postgres/shipmentrepository"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure"
	"github.com/emoss08/trenova/internal/modules/operations/domain/servicefailure/servicefailuredb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/ports/shipmentports"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/zap"
)

func NewBriefingRepository(r *Repository) shipmentports.ShipmentBriefingRepository { return r }

func (r *Repository) LeadingLateReason(
	ctx context.Context,
	_ pagination.TenantInfo,
) (*shipmentports.ShipmentLateReason, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*shipmentports.ShipmentLateReason, error) {
			sf := servicefailuredb.ServiceFailures
			rc := servicefailuredb.ReasonCodes
			sp := shipmentdb.Shipments
			count := keel.Raw("COUNT(DISTINCT ?)", sf.ShipmentID)

			rows, err := keel.Into[shipmentports.ShipmentLateReason](ctx, r.db.Keel(ctx), sf.
				Join(sp, sp.ID.EqCol(sf.ShipmentID)).
				Join(rc, rc.ID.EqCol(sf.ReasonCodeID)).
				Pick(rc.Label, count).
				Where(
					sf.Status.In(servicefailure.UnresolvedStatuses()...),
					shipmentrepository.StageCondition(shipment.StageLate),
				).
				Group(rc.Label).
				Order(keel.Desc(count), rc.Label.Asc()).
				Limit(1))
			if err != nil {
				r.l.Error("failed to read the leading late reason", zap.Error(err))
				return nil, err
			}
			if len(rows) == 0 {
				return nil, nil //nolint:nilnil // no unresolved late failures
			}

			return &rows[0], nil
		},
	)
}
