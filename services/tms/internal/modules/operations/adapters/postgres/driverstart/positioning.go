package driverstart

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchmarket"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning"
	"github.com/emoss08/trenova/internal/modules/operations/domain/driverpositioning/driverpositioningdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
)

func LivePositioning(
	p *driverpositioningdb.PositioningsTable,
	worker keel.Expr,
	now int64,
) []keel.Pred {
	a := shipmentdb.Assignments.As("lp_newer")
	newer := a.
		Pick(a.ID).
		Where(
			a.PrimaryWorkerID.EqCol(worker),
			a.ArchivedAt.IsNull(),
			keel.Int64Of(a.CreatedAt).GtExpr(p.CreatedAt),
		)

	return []keel.Pred{
		p.Status.Eq(driverpositioning.StatusActive),
		p.SlotStart.Gt(now - dispatchmarket.SlotSeconds - driverpositioning.ExpirySeconds),
		keel.NotExists(newer),
	}
}
