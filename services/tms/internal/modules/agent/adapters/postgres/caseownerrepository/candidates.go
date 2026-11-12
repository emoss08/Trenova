package caseownerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/ports/caseownerports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/collection/collectiondb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute/invoicedisputedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/equipment/domain/fleetcode/fleetcodedb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker/workerdb"
	"github.com/emoss08/trenova/shared/pulid"
)

const candidatesPerSource = 4

type source func(id pulid.ID) []keel.Projection

var sources = map[recordkind.Kind]source{
	recordkind.Kind(permission.ResourceShipment):       shipmentPeople,
	recordkind.Kind(permission.ResourceWorker):         workerPeople,
	recordkind.Kind(permission.ResourceInvoice):        invoicePeople,
	recordkind.Kind(permission.ResourceCustomer):       customerPeople,
	recordkind.Kind(permission.ResourceInboundMessage): messagePeople,
	recordkind.Kind(permission.ResourceInvoiceDispute): func(id pulid.ID) []keel.Projection {
		d := invoicedisputedb.InvoiceDisputes

		return invoicePeopleOf(
			func(c keel.Col) keel.Pred { return c.InQuery(d.Pick(d.InvoiceID).Where(d.ID.Eq(id))) },
			func(c keel.Col) keel.Pred { return c.InQuery(d.Pick(d.CustomerID).Where(d.ID.Eq(id))) },
		)
	},
	recordkind.DetentionOccurrence: func(id pulid.ID) []keel.Projection {
		o := detentiondb.DetentionOccurrences
		s, m := shipmentdb.Shipments, shipmentdb.ShipmentMoves

		return shipmentPeopleOf(
			s.ID.InQuery(o.Pick(o.ShipmentID).Where(o.ID.Eq(id))),
			shipmentdb.Assignments.ShipmentMoveID.InQuery(m.Pick(m.ID).Where(
				m.ShipmentID.InQuery(o.Pick(o.ShipmentID).Where(o.ID.Eq(id))),
			)),
		)
	},
}

func fleetManagers(workers keel.Pred) keel.Projection {
	fc, w := fleetcodedb.FleetCodes, workerdb.Workers

	return fc.Pick(fc.ManagerID).Where(
		fc.ManagerID.IsNotNull(),
		fc.ID.InQuery(w.Pick(w.FleetCodeID).Where(w.FleetCodeID.IsNotNull(), workers)),
	).Limit(candidatesPerSource)
}

func shipmentPeople(id pulid.ID) []keel.Projection {
	m := shipmentdb.ShipmentMoves

	return shipmentPeopleOf(
		shipmentdb.Shipments.ID.Eq(id),
		shipmentdb.Assignments.ShipmentMoveID.InQuery(m.Pick(m.ID).Where(m.ShipmentID.Eq(id))),
	)
}

func shipmentPeopleOf(shipment, moves keel.Pred) []keel.Projection {
	s, a, w := shipmentdb.Shipments, shipmentdb.Assignments, workerdb.Workers

	return []keel.Projection{
		s.Pick(s.OwnerID).Where(shipment, s.OwnerID.IsNotNull()).Limit(1),
		fleetManagers(w.ID.InQuery(
			a.Pick(a.PrimaryWorkerID).Where(a.PrimaryWorkerID.IsNotNull(), moves),
		)),
	}
}

func workerPeople(id pulid.ID) []keel.Projection {
	w := workerdb.Workers

	return []keel.Projection{
		w.Pick(w.ManagerID).Where(w.ID.Eq(id), w.ManagerID.IsNotNull()).Limit(1),
		fleetManagers(w.ID.Eq(id)),
	}
}

func invoicePeople(id pulid.ID) []keel.Projection {
	inv := invoicedb.Invoices

	return invoicePeopleOf(
		func(c keel.Col) keel.Pred { return c.InQuery(inv.Pick(inv.ID).Where(inv.ID.Eq(id))) },
		func(c keel.Col) keel.Pred {
			return c.InQuery(inv.Pick(inv.CustomerID).Where(inv.ID.Eq(id)))
		},
	)
}

func invoicePeopleOf(invoice, customer func(keel.Col) keel.Pred) []keel.Projection {
	bq := billingqueuedb.BillingQueueItems
	ca := collectiondb.Accounts
	bp := customerdb.CustomerBillingProfiles

	return []keel.Projection{
		bq.Pick(bq.AssignedBillerID).Where(
			bq.AssignedBillerID.IsNotNull(),
			invoice(bq.InvoiceID.Col),
		).Limit(1),
		ca.Pick(ca.CollectorID).Where(
			ca.CollectorID.IsNotNull(),
			customer(ca.CustomerID.Col),
		).Limit(1),
		bp.Pick(bp.DefaultBillerID).Where(
			bp.DefaultBillerID.IsNotNull(),
			customer(bp.CustomerID.Col),
		).Limit(1),
	}
}

func customerPeople(id pulid.ID) []keel.Projection {
	ca := collectiondb.Accounts
	bp := customerdb.CustomerBillingProfiles

	return []keel.Projection{
		ca.Pick(ca.CollectorID).Where(ca.CollectorID.IsNotNull(), ca.CustomerID.Eq(id)).Limit(1),
		bp.Pick(bp.DefaultBillerID).
			Where(bp.DefaultBillerID.IsNotNull(), bp.CustomerID.Eq(id)).
			Limit(1),
	}
}

func messagePeople(id pulid.ID) []keel.Projection {
	im := inboundmessagedb.InboundMessages

	return []keel.Projection{
		im.Pick(im.OwnerID).Where(im.ID.Eq(id), im.OwnerID.IsNotNull()).Limit(1),
	}
}

func (r *repository) Candidates(
	ctx context.Context,
	req caseownerports.CandidatesRequest,
) ([]pulid.ID, error) {
	people, known := sources[req.Kind]
	if !known {
		return []pulid.ID{}, nil
	}

	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pulid.ID, error) {
		reads := people(req.RecordID)
		found := make([][]pulid.ID, len(reads))
		items := make([]keel.BatchItem, 0, len(reads))
		for idx, read := range reads {
			items = append(items, keel.IntoSlice(read, &found[idx]))
		}
		if err := keel.Batch(ctx, r.db.Keel(ctx), items...); err != nil {
			return nil, fmt.Errorf("read who a record names for its case: %w", err)
		}

		out := make([]pulid.ID, 0, len(reads))
		for _, ids := range found {
			out = append(out, ids...)
		}

		return out, nil
	})
}
