package factoringrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipmentvocab"
	"github.com/emoss08/trenova/shared/pulid"
)

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
			db := r.db.Keel(ctx)

			a := shipmentdb.Assignments
			am := a.ShipmentMove()
			assignments, err := a.Where(
				am.ShipmentID.In(req.ShipmentIDs...),
				a.ArchivedAt.IsNull(),
				a.PrimaryWorkerID.IsNotNull(),
			).
				Load(am, a.PrimaryWorker()).
				Order(a.CreatedAt.Desc()).
				All(ctx, db)
			if err != nil {
				return nil, err
			}
			for _, assignment := range assignments {
				if assignment.ShipmentMove == nil || assignment.PrimaryWorker == nil {
					continue
				}
				contact := contactFor(result, assignment.ShipmentMove.ShipmentID)
				email := strings.TrimSpace(assignment.PrimaryWorker.Email)
				if contact.DriverEmail == "" && email != "" {
					contact.DriverName = assignment.PrimaryWorker.FullName()
					contact.DriverEmail = email
				}
			}

			ca := shipmentdb.CarrierAssignments
			cm := ca.ShipmentMove()
			carriers, err := ca.Where(
				cm.ShipmentID.In(req.ShipmentIDs...),
				ca.Status.Ne(shipmentvocab.CarrierAssignmentStatusCanceled),
			).
				Load(cm, ca.Carrier()).
				Order(ca.CreatedAt.Desc()).
				All(ctx, db)
			if err != nil {
				return nil, err
			}
			for _, carrierAssignment := range carriers {
				if carrierAssignment.ShipmentMove == nil || carrierAssignment.Carrier == nil {
					continue
				}
				contact := contactFor(result, carrierAssignment.ShipmentMove.ShipmentID)
				email := strings.TrimSpace(carrierAssignment.Carrier.Email)
				if contact.CarrierEmail == "" && email != "" {
					contact.CarrierName = carrierAssignment.Carrier.Name
					contact.CarrierEmail = email
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
