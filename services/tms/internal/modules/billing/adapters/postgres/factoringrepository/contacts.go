package factoringrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
)

const moveRelationAlias = "shipment_move"

func (r *invoiceStateRepository) ListShipmentContacts(
	ctx context.Context,
	req *factoringports.ListShipmentContactsRequest,
) (map[pulid.ID]*factoringports.ShipmentContact, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]*factoringports.ShipmentContact, error) {
			result := make(map[pulid.ID]*factoringports.ShipmentContact, len(req.ShipmentIDs))
			if len(req.ShipmentIDs) == 0 {
				return result, nil
			}
			dba := r.db.DBForContext(ctx)
			inShipments := func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(
					shipmentcol.ShipmentMoveColumns.ShipmentID.WithAlias(moveRelationAlias).In(),
					bun.List(req.ShipmentIDs),
				)
			}

			assignments := make([]*shipment.Assignment, 0, len(req.ShipmentIDs))
			if err := dba.NewSelect().
				Model(&assignments).
				Relation(shipmentcol.AssignmentRelations.ShipmentMove).
				Relation(shipmentcol.AssignmentRelations.PrimaryWorker).
				Apply(shipmentcol.AssignmentApplyTenant(req.TenantInfo)).
				Apply(inShipments).
				Where(shipmentcol.AssignmentColumns.ArchivedAt.IsNull()).
				Where(shipmentcol.AssignmentColumns.PrimaryWorkerID.IsNotNull()).
				Order(shipmentcol.AssignmentColumns.CreatedAt.OrderDesc()).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list shipment drivers: %w", err)
			}
			for _, a := range assignments {
				if a.ShipmentMove == nil || a.PrimaryWorker == nil {
					continue
				}
				contact := contactFor(result, a.ShipmentMove.ShipmentID)
				if contact.DriverEmail == "" && strings.TrimSpace(a.PrimaryWorker.Email) != "" {
					contact.DriverName = a.PrimaryWorker.FullName()
					contact.DriverEmail = strings.TrimSpace(a.PrimaryWorker.Email)
				}
			}

			carriers := make([]*shipment.CarrierAssignment, 0, len(req.ShipmentIDs))
			if err := dba.NewSelect().
				Model(&carriers).
				Relation(shipmentcol.CarrierAssignmentRelations.ShipmentMove).
				Relation(shipmentcol.CarrierAssignmentRelations.Carrier).
				Apply(shipmentcol.CarrierAssignmentApplyTenant(req.TenantInfo)).
				Apply(inShipments).
				Where(shipmentcol.CarrierAssignmentColumns.Status.Ne(), shipment.CarrierAssignmentStatusCanceled).
				Order(shipmentcol.CarrierAssignmentColumns.CreatedAt.OrderDesc()).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("list shipment carriers: %w", err)
			}
			for _, ca := range carriers {
				if ca.ShipmentMove == nil || ca.Carrier == nil {
					continue
				}
				contact := contactFor(result, ca.ShipmentMove.ShipmentID)
				if contact.CarrierEmail == "" && strings.TrimSpace(ca.Carrier.Email) != "" {
					contact.CarrierName = ca.Carrier.Name
					contact.CarrierEmail = strings.TrimSpace(ca.Carrier.Email)
				}
			}

			return result, nil
		},
	)
}

func contactFor(
	contacts map[pulid.ID]*factoringports.ShipmentContact,
	shipmentID pulid.ID,
) *factoringports.ShipmentContact {
	contact, ok := contacts[shipmentID]
	if !ok {
		contact = &factoringports.ShipmentContact{ShipmentID: shipmentID}
		contacts[shipmentID] = contact
	}
	return contact
}
