package assistantcaserepository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/billingqueue"
	"github.com/emoss08/trenova/internal/core/domain/carrier"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/deskcase"
	"github.com/emoss08/trenova/internal/core/domain/detention"
	"github.com/emoss08/trenova/internal/core/domain/document"
	"github.com/emoss08/trenova/internal/core/domain/invoice"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) repositories.AssistantCaseRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.assistant-case-repository")}
}

type shipmentRecordRow struct {
	ID                pulid.ID        `bun:"id"`
	ProNumber         string          `bun:"pro_number"`
	Status            shipment.Status `bun:"status"`
	CustomerID        pulid.ID        `bun:"customer_id"`
	BillToID          pulid.ID        `bun:"bill_to_id"`
	NextStopID        pulid.ID        `bun:"next_stop_id"`
	NextAppointmentAt *int64          `bun:"next_appointment_at"`
	CarrierIDs        []string        `bun:"carrier_ids,array"`
}

// shipmentRecordsQuery reads each shipment with its next stop not yet
// reached (what a snooze to the appointment follows) and the carriers on
// its moves (who its case can wait on).
const shipmentRecordsQuery = `
SELECT
	sp.id,
	COALESCE(sp.pro_number, '') AS pro_number,
	sp.status,
	sp.customer_id,
	COALESCE(sp.bill_to_customer_id, sp.customer_id) AS bill_to_id,
	nxt.id AS next_stop_id,
	nxt.scheduled_window_start AS next_appointment_at,
	ARRAY(
		SELECT DISTINCT casn.carrier_id
		FROM carrier_assignments AS casn
		JOIN shipment_moves AS smv
			ON smv.id = casn.shipment_move_id
			AND smv.organization_id = casn.organization_id
			AND smv.business_unit_id = casn.business_unit_id
		WHERE smv.shipment_id = sp.id
			AND casn.organization_id = sp.organization_id
			AND casn.business_unit_id = sp.business_unit_id
			AND casn.status <> ?
	) AS carrier_ids
FROM shipments AS sp
LEFT JOIN LATERAL (
	SELECT stp.id, stp.scheduled_window_start
	FROM stops AS stp
	JOIN shipment_moves AS smv
		ON smv.id = stp.shipment_move_id
		AND smv.organization_id = stp.organization_id
		AND smv.business_unit_id = stp.business_unit_id
	WHERE smv.shipment_id = sp.id
		AND stp.organization_id = sp.organization_id
		AND stp.business_unit_id = sp.business_unit_id
		AND smv.status <> ?
		AND stp.status <> ?
		AND stp.actual_arrival IS NULL
	ORDER BY smv.sequence ASC, stp.sequence ASC
	LIMIT 1
) AS nxt ON TRUE
WHERE sp.organization_id = ?
	AND sp.business_unit_id = ?
	AND sp.id IN (?)
`

func (r *repository) ListRecords(
	ctx context.Context,
	req *repositories.ListCaseRecordsRequest,
) (map[pulid.ID]*deskcase.Record, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]*deskcase.Record, error) {
		out := make(map[pulid.ID]*deskcase.Record, len(req.Refs))
		byType := make(map[agent.SubjectType][]pulid.ID, len(deskcase.Subjects()))
		for _, ref := range req.Refs {
			if deskcase.IsSubject(ref.Type) && ref.ID.IsNotNil() {
				byType[ref.Type] = append(byType[ref.Type], ref.ID)
			}
		}

		if ids := byType[agent.SubjectShipment]; len(ids) > 0 {
			if err := r.shipmentRecords(ctx, req.TenantInfo, ids, out); err != nil {
				return nil, err
			}
		}
		if ids := byType[agent.SubjectInvoice]; len(ids) > 0 {
			if err := r.invoiceRecords(ctx, req.TenantInfo, ids, out); err != nil {
				return nil, err
			}
		}
		if ids := byType[agent.SubjectInvoiceDispute]; len(ids) > 0 {
			if err := r.disputeRecords(ctx, req.TenantInfo, ids, out); err != nil {
				return nil, err
			}
		}

		return out, nil
	})
}

func (r *repository) shipmentRecords(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
	out map[pulid.ID]*deskcase.Record,
) error {
	rows := make([]shipmentRecordRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).NewRaw(
		shipmentRecordsQuery,
		shipment.CarrierAssignmentStatusCanceled,
		shipment.MoveStatusCanceled,
		shipment.StopStatusCanceled,
		tenant.OrgID,
		tenant.BuID,
		bun.List(ids),
	).Scan(ctx, &rows); err != nil {
		return fmt.Errorf("read the cases' shipments: %w", err)
	}

	for i := range rows {
		row := &rows[i]
		record := &deskcase.Record{
			Type:              agent.SubjectShipment,
			ID:                row.ID,
			Label:             row.ProNumber,
			Status:            string(row.Status),
			CustomerID:        row.CustomerID,
			BillToID:          row.BillToID,
			NextStopID:        row.NextStopID,
			NextAppointmentAt: row.NextAppointmentAt,
			CarrierIDs:        make([]pulid.ID, 0, len(row.CarrierIDs)),
		}
		for _, id := range row.CarrierIDs {
			record.CarrierIDs = append(record.CarrierIDs, pulid.ID(id))
		}
		switch row.Status {
		case shipment.StatusInvoiced, shipment.StatusCanceled:
			record.Closed = true
			record.ClosedAs = string(row.Status)
		case shipment.StatusNew,
			shipment.StatusPartiallyAssigned,
			shipment.StatusAssigned,
			shipment.StatusInTransit,
			shipment.StatusDelayed,
			shipment.StatusPartiallyCompleted,
			shipment.StatusReadyToInvoice,
			shipment.StatusCompleted:
		}
		out[row.ID] = record
	}

	return nil
}

func (r *repository) invoiceRecords(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
	out map[pulid.ID]*deskcase.Record,
) error {
	cols := buncolgen.InvoiceColumns
	rows := make([]*invoice.Invoice, 0, len(ids))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&rows).
		Column(
			cols.ID.Bare(),
			cols.Number.Bare(),
			cols.Status.Bare(),
			cols.SettlementStatus.Bare(),
			cols.CustomerID.Bare(),
		).
		Apply(buncolgen.InvoiceApplyTenant(tenant)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx); err != nil {
		return fmt.Errorf("read the cases' invoices: %w", err)
	}

	for _, row := range rows {
		record := &deskcase.Record{
			Type:       agent.SubjectInvoice,
			ID:         row.ID,
			Label:      row.Number,
			Status:     string(row.Status),
			CustomerID: row.CustomerID,
			BillToID:   row.CustomerID,
		}
		switch {
		case row.Status == invoice.StatusVoided:
			record.Closed = true
			record.ClosedAs = string(invoice.StatusVoided)
		case row.SettlementStatus == invoice.SettlementStatusPaid:
			record.Closed = true
			record.ClosedAs = string(invoice.SettlementStatusPaid)
			record.Status = string(row.SettlementStatus)
		}
		out[row.ID] = record
	}

	return nil
}

type disputeRecordRow struct {
	ID            pulid.ID                  `bun:"id"`
	Status        invoice.DisputeCaseStatus `bun:"status"`
	Resolution    invoice.DisputeResolution `bun:"resolution"`
	CustomerID    pulid.ID                  `bun:"customer_id"`
	InvoiceID     pulid.ID                  `bun:"invoice_id"`
	InvoiceNumber string                    `bun:"invoice_number"`
}

func (r *repository) disputeRecords(
	ctx context.Context,
	tenant pagination.TenantInfo,
	ids []pulid.ID,
	out map[pulid.ID]*deskcase.Record,
) error {
	cols := buncolgen.InvoiceDisputeColumns
	inv := buncolgen.InvoiceColumns
	rows := make([]disputeRecordRow, 0, len(ids))
	if err := r.db.DBForContext(ctx).
		NewSelect().
		Model((*invoice.InvoiceDispute)(nil)).
		ColumnExpr(cols.ID.Qualified()).
		ColumnExpr(cols.Status.Qualified()).
		ColumnExpr(cols.Resolution.Qualified()).
		ColumnExpr(cols.CustomerID.Qualified()).
		ColumnExpr(cols.InvoiceID.Qualified()).
		ColumnExpr("COALESCE("+inv.Number.Qualified()+", '') AS invoice_number").
		Join("LEFT JOIN "+buncolgen.InvoiceTable.Name+" AS "+buncolgen.InvoiceTable.Alias).
		JoinOn(inv.ID.EqColumn(cols.InvoiceID)).
		JoinOn(inv.OrganizationID.EqColumn(cols.OrganizationID)).
		JoinOn(inv.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
		Apply(buncolgen.InvoiceDisputeApplyTenant(tenant)).
		Where(cols.ID.In(), bun.List(ids)).
		Scan(ctx, &rows); err != nil {
		return fmt.Errorf("read the cases' disputes: %w", err)
	}

	for i := range rows {
		row := &rows[i]
		record := &deskcase.Record{
			Type:       agent.SubjectInvoiceDispute,
			ID:         row.ID,
			Label:      row.InvoiceNumber,
			Status:     string(row.Status),
			CustomerID: row.CustomerID,
			BillToID:   row.CustomerID,
			InvoiceID:  row.InvoiceID,
		}
		switch row.Status {
		case invoice.DisputeCaseStatusResolved, invoice.DisputeCaseStatusWithdrawn:
			record.Closed = true
			record.ClosedAs = string(row.Status)
			if row.Resolution != "" {
				record.ClosedAs = string(row.Resolution)
			}
		case invoice.DisputeCaseStatusOpen:
		}
		out[row.ID] = record
	}

	return nil
}

type shipmentFactsRow struct {
	Status             shipment.Status `bun:"status"`
	DeliveredAt        *int64          `bun:"delivered_at"`
	PODOnFile          bool            `bun:"pod_on_file"`
	DocumentTypeIDs    []string        `bun:"document_type_ids,array"`
	RateCons           []rateConRow    `bun:"rate_cons,type:jsonb"`
	OpenChargeIssues   []string        `bun:"open_charge_issues,array"`
	DetentionAccruing  int             `bun:"detention_accruing"`
	DetentionUnapprove int             `bun:"detention_unapproved"`
	CustomerNotifiedAt *int64          `bun:"customer_notified_at"`
	InBillingQueue     bool            `bun:"in_billing_queue"`
}

type rateConRow struct {
	CarrierName string `json:"carrierName"`
	Confirmed   bool   `json:"confirmed"`
}

// shipmentFactsQuery reads, in one round trip, everything the ready-to-bill
// checklist needs that billing readiness does not already say. Every
// subquery is keyed by the shipment and the tenant, and each table is read
// through an index on its shipment.
const shipmentFactsQuery = `
SELECT
	sp.status,
	sp.actual_delivery_date AS delivered_at,
	EXISTS (
		SELECT 1
		FROM documents AS doc
		JOIN document_types AS dt
			ON dt.id = doc.document_type_id
			AND dt.organization_id = doc.organization_id
			AND dt.business_unit_id = doc.business_unit_id
		WHERE doc.organization_id = sp.organization_id
			AND doc.business_unit_id = sp.business_unit_id
			AND doc.resource_type = ?
			AND doc.resource_id = sp.id
			AND doc.is_current_version
			AND doc.status NOT IN (?)
			AND upper(dt.code) = upper(?)
	) AS pod_on_file,
	ARRAY(
		SELECT DISTINCT doc.document_type_id
		FROM documents AS doc
		WHERE doc.organization_id = sp.organization_id
			AND doc.business_unit_id = sp.business_unit_id
			AND doc.resource_type = ?
			AND doc.resource_id = sp.id
			AND doc.is_current_version
			AND doc.status NOT IN (?)
			AND doc.document_type_id IS NOT NULL
	) AS document_type_ids,
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'carrierName', COALESCE(car.name, ''),
			'confirmed', casn.status = ? OR EXISTS (
				SELECT 1
				FROM rate_confirmations AS rc
				WHERE rc.organization_id = casn.organization_id
					AND rc.business_unit_id = casn.business_unit_id
					AND rc.carrier_assignment_id = casn.id
					AND rc.status = ?
			)
		) ORDER BY smv.sequence)
		FROM carrier_assignments AS casn
		JOIN shipment_moves AS smv
			ON smv.id = casn.shipment_move_id
			AND smv.organization_id = casn.organization_id
			AND smv.business_unit_id = casn.business_unit_id
		LEFT JOIN carriers AS car
			ON car.id = casn.carrier_id
			AND car.organization_id = casn.organization_id
			AND car.business_unit_id = casn.business_unit_id
		WHERE smv.shipment_id = sp.id
			AND casn.organization_id = sp.organization_id
			AND casn.business_unit_id = sp.business_unit_id
			AND casn.status <> ?
	), '[]'::jsonb) AS rate_cons,
	ARRAY(
		SELECT DISTINCT bqis.code
		FROM billing_queue_issues AS bqis
		JOIN billing_queue_items AS bqi
			ON bqi.id = bqis.item_id
			AND bqi.organization_id = bqis.organization_id
			AND bqi.business_unit_id = bqis.business_unit_id
		WHERE bqi.organization_id = sp.organization_id
			AND bqi.business_unit_id = sp.business_unit_id
			AND bqi.shipment_id = sp.id
			AND bqi.status <> ?
			AND bqis.resolution_key IS NULL
			AND bqis.code IN (?)
	) AS open_charge_issues,
	(
		SELECT count(*)
		FROM detention_occurrences AS dto
		WHERE dto.organization_id = sp.organization_id
			AND dto.business_unit_id = sp.business_unit_id
			AND dto.shipment_id = sp.id
			AND dto.status = ?
	) AS detention_accruing,
	(
		SELECT count(*)
		FROM detention_occurrences AS dto
		WHERE dto.organization_id = sp.organization_id
			AND dto.business_unit_id = sp.business_unit_id
			AND dto.shipment_id = sp.id
			AND dto.status IN (?)
	) AS detention_unapproved,
	(
		SELECT max(sc.created_at)
		FROM shipment_comments AS sc
		WHERE sc.organization_id = sp.organization_id
			AND sc.business_unit_id = sp.business_unit_id
			AND sc.shipment_id = sp.id
			AND sc.type = ?
			AND sc.metadata ->> ? IN (?)
	) AS customer_notified_at,
	EXISTS (
		SELECT 1
		FROM billing_queue_items AS bqi
		WHERE bqi.organization_id = sp.organization_id
			AND bqi.business_unit_id = sp.business_unit_id
			AND bqi.shipment_id = sp.id
			AND bqi.status <> ?
	) AS in_billing_queue
FROM shipments AS sp
WHERE sp.organization_id = ?
	AND sp.business_unit_id = ?
	AND sp.id = ?
`

func (r *repository) ShipmentFacts(
	ctx context.Context,
	req *repositories.GetShipmentCaseFactsRequest,
) (*deskcase.ShipmentFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ShipmentFacts, error) {
		rows := make([]shipmentFactsRow, 0, 1)
		if err := r.db.DBForContext(ctx).NewRaw(
			shipmentFactsQuery,
			permission.ResourceShipment.String(),
			bun.List([]document.Status{document.StatusRejected, document.StatusArchived}),
			req.PODCode,
			permission.ResourceShipment.String(),
			bun.List([]document.Status{document.StatusRejected, document.StatusArchived}),
			shipment.CarrierAssignmentStatusConfirmed,
			rateconfirmation.StatusConfirmed,
			shipment.CarrierAssignmentStatusCanceled,
			billingqueue.StatusCanceled,
			bun.List([]billingqueue.IssueCode{
				billingqueue.IssueAccessorialNotOnRateCon,
				billingqueue.IssueChargeOverRateCon,
			}),
			detention.OccurrenceStatusAccruing,
			bun.List([]detention.OccurrenceStatus{
				detention.OccurrenceStatusPending,
				detention.OccurrenceStatusDisputed,
			}),
			shipment.CommentTypeCustomerUpdate,
			shipment.CommentMetadataTool,
			bun.List(shipment.CustomerEmailTools()),
			billingqueue.StatusCanceled,
			req.TenantInfo.OrgID,
			req.TenantInfo.BuID,
			req.ShipmentID,
		).Scan(ctx, &rows); err != nil {
			return nil, fmt.Errorf("read the shipment's case facts: %w", err)
		}
		if len(rows) == 0 {
			return nil, dberror.HandleNotFoundError(sql.ErrNoRows, "Shipment")
		}

		row := &rows[0]
		facts := &deskcase.ShipmentFacts{
			Status:              row.Status,
			DeliveredAt:         row.DeliveredAt,
			PODCode:             req.PODCode,
			PODOnFile:           row.PODOnFile,
			RateCons:            make([]deskcase.RateCon, 0, len(row.RateCons)),
			OpenChargeIssues:    row.OpenChargeIssues,
			DetentionAccruing:   row.DetentionAccruing,
			DetentionUnapproved: row.DetentionUnapprove,
			CustomerNotifiedAt:  row.CustomerNotifiedAt,
			InBillingQueue:      row.InBillingQueue,
		}
		facts.DocumentTypesOnFile = make(map[pulid.ID]struct{}, len(row.DocumentTypeIDs))
		for _, id := range row.DocumentTypeIDs {
			facts.DocumentTypesOnFile[pulid.ID(id)] = struct{}{}
		}
		for _, rateCon := range row.RateCons {
			facts.RateCons = append(facts.RateCons, deskcase.RateCon(rateCon))
		}

		return facts, nil
	})
}

func (r *repository) InvoiceFacts(
	ctx context.Context,
	req *repositories.GetInvoiceCaseFactsRequest,
) (*deskcase.InvoiceFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.InvoiceFacts, error) {
		cols := buncolgen.InvoiceColumns
		entity := new(invoice.Invoice)
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			Column(
				cols.Status.Bare(),
				cols.SendStatus.Bare(),
				cols.DisputeStatus.Bare(),
				cols.SettlementStatus.Bare(),
				cols.DueDate.Bare(),
			).
			Apply(buncolgen.InvoiceApplyTenant(req.TenantInfo)).
			Where(cols.ID.Eq(), req.InvoiceID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "Invoice")
		}

		return &deskcase.InvoiceFacts{
			Status:           entity.Status,
			SendStatus:       entity.SendStatus,
			DisputeStatus:    entity.DisputeStatus,
			SettlementStatus: entity.SettlementStatus,
			DueDate:          entity.DueDate,
		}, nil
	})
}

func (r *repository) Parties(
	ctx context.Context,
	req *repositories.ListCasePartiesRequest,
) ([]deskcase.Party, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]deskcase.Party, error) {
		db := r.db.DBForContext(ctx)
		out := make([]deskcase.Party, 0, len(req.CarrierIDs)+1)

		if req.CustomerID.IsNotNil() {
			cols := buncolgen.CustomerColumns
			entity := new(customer.Customer)
			err := db.NewSelect().
				Model(entity).
				Column(cols.ID.Bare(), cols.Name.Bare()).
				Apply(buncolgen.CustomerApplyTenant(req.TenantInfo)).
				Where(cols.ID.Eq(), req.CustomerID).
				Scan(ctx)
			if err != nil && !dberror.IsNotFoundError(err) {
				return nil, fmt.Errorf("read the case's customer: %w", err)
			}
			if err == nil {
				out = append(out, deskcase.Party{
					Kind: deskcase.WaitingOnCustomer,
					ID:   entity.ID,
					Name: entity.Name,
				})
			}
		}

		if len(req.CarrierIDs) > 0 {
			cols := buncolgen.CarrierColumns
			carriers := make([]*carrier.Carrier, 0, len(req.CarrierIDs))
			if err := db.NewSelect().
				Model(&carriers).
				Column(cols.ID.Bare(), cols.Name.Bare()).
				Apply(buncolgen.CarrierApplyTenant(req.TenantInfo)).
				Where(cols.ID.In(), bun.List(req.CarrierIDs)).
				Order(cols.Name.OrderAsc()).
				Scan(ctx); err != nil {
				return nil, fmt.Errorf("read the case's carriers: %w", err)
			}
			for _, entity := range carriers {
				out = append(out, deskcase.Party{
					Kind: deskcase.WaitingOnCarrier,
					ID:   entity.ID,
					Name: entity.Name,
				})
			}
		}

		return out, nil
	})
}
