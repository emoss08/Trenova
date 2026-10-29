package shipmentrepository

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

// LoadGraphSource loads a tenant-scoped shipment with its ordered
// moves/stops, additional charges, and commodities — the full graph needed to
// produce copies.
func LoadGraphSource(
	ctx context.Context,
	db *keel.DB,
	tenantInfo pagination.TenantInfo,
	shipmentID pulid.ID,
) (*shipment.Shipment, error) {
	q, err := tenantShipments(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	sp := shipmentdb.Shipments
	m := sp.Moves()
	st := m.Stops()
	entity, err := q.Where(sp.ID.Eq(shipmentID)).
		Load(
			m.Order(m.Sequence.Asc()).Load(st.Order(st.Sequence.Asc())),
			sp.AdditionalCharges(),
			sp.ChargeAllocations(),
			sp.Commodities(),
		).
		One(ctx, db)
	if err != nil {
		return nil, shipmentErrors.Map(err, shipmentID.String())
	}

	return entity, nil
}
