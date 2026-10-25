package assistantcaserepository

import (
	"context"
	"errors"
	"fmt"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/deskcase"
	"github.com/emoss08/trenova/internal/modules/agent/ports/deskcaseports"
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
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

func New(p Params) deskcaseports.AssistantCaseRepository {
	return &repository{db: p.DB, l: p.Logger.Named("postgres.assistant-case-repository")}
}

var (
	invoiceErrors  = dberror.Mapping{Entity: "Invoice"}
	shipmentErrors = dberror.Mapping{Entity: "Shipment"}
)

type shipmentRecordRow struct {
	ID                pulid.ID
	ProNumber         string
	Status            shipment.Status
	CustomerID        pulid.ID
	BillToID          pulid.ID
	NextStopID        pulid.ID
	NextAppointmentAt *int64
	CarrierIDs        []string
}

const shipmentRecordsQuery = `
SELECT
	sp.id,
	COALESCE(sp.pro_number, '') AS pro_number,
	sp.status,
	sp.customer_id,
	COALESCE(sp.bill_to_customer_id, sp.customer_id) AS bill_to_id,
	COALESCE(nxt.id, '') AS next_stop_id,
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
			AND casn.status <> $1
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
		AND smv.status <> $2
		AND stp.status <> $3
		AND stp.actual_arrival IS NULL
	ORDER BY smv.sequence ASC, stp.sequence ASC
	LIMIT 1
) AS nxt ON TRUE
WHERE sp.organization_id = $4
	AND sp.business_unit_id = $5
	AND sp.id = ANY($6)
`

func (r *repository) ListRecords(
	ctx context.Context,
	req *deskcaseports.ListCaseRecordsRequest,
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
			if err := r.invoiceRecords(ctx, ids, out); err != nil {
				return nil, err
			}
		}
		if ids := byType[agent.SubjectInvoiceDispute]; len(ids) > 0 {
			if err := r.disputeRecords(ctx, ids, out); err != nil {
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
	rows, err := keel.QueryAll(
		ctx,
		r.db.Keel(ctx),
		pgx.RowToStructByPos[shipmentRecordRow],
		shipmentRecordsQuery,
		string(shipment.CarrierAssignmentStatusCanceled),
		string(shipment.MoveStatusCanceled),
		string(shipment.StopStatusCanceled),
		tenant.OrgID.String(),
		tenant.BuID.String(),
		idStrings(ids),
	)
	if err != nil {
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
	ids []pulid.ID,
	out map[pulid.ID]*deskcase.Record,
) error {
	rows, err := invoiceRecordsQuery(ids).All(ctx, r.db.Keel(ctx))
	if err != nil {
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
	ID            pulid.ID
	Status        invoice.DisputeCaseStatus
	Resolution    invoice.DisputeResolution
	CustomerID    pulid.ID
	InvoiceID     pulid.ID
	InvoiceNumber string
}

func (r *repository) disputeRecords(
	ctx context.Context,
	ids []pulid.ID,
	out map[pulid.ID]*deskcase.Record,
) error {
	rows, err := keel.Into[disputeRecordRow](ctx, r.db.Keel(ctx), disputeRecordsQuery(ids))
	if err != nil {
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
	Status             shipment.Status
	DeliveredAt        *int64
	PODOnFile          bool
	DocumentTypeIDs    []string
	RateCons           []byte
	OpenChargeIssues   []string
	DetentionAccruing  int64
	DetentionUnapprove int64
	CustomerNotifiedAt *int64
	InBillingQueue     bool
}

type rateConRow struct {
	CarrierName string `json:"carrierName"`
	Confirmed   bool   `json:"confirmed"`
}

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
			AND doc.resource_type = $1
			AND doc.resource_id = sp.id
			AND doc.is_current_version
			AND doc.status <> ALL($2)
			AND upper(dt.code) = upper($3)
	) AS pod_on_file,
	ARRAY(
		SELECT DISTINCT doc.document_type_id
		FROM documents AS doc
		WHERE doc.organization_id = sp.organization_id
			AND doc.business_unit_id = sp.business_unit_id
			AND doc.resource_type = $4
			AND doc.resource_id = sp.id
			AND doc.is_current_version
			AND doc.status <> ALL($5)
			AND doc.document_type_id IS NOT NULL
	) AS document_type_ids,
	COALESCE((
		SELECT jsonb_agg(jsonb_build_object(
			'carrierName', COALESCE(car.name, ''),
			'confirmed', casn.status = $6 OR EXISTS (
				SELECT 1
				FROM rate_confirmations AS rc
				WHERE rc.organization_id = casn.organization_id
					AND rc.business_unit_id = casn.business_unit_id
					AND rc.carrier_assignment_id = casn.id
					AND rc.status = $7
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
			AND casn.status <> $8
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
			AND bqi.status <> $9
			AND bqis.resolution_key IS NULL
			AND bqis.code = ANY($10)
	) AS open_charge_issues,
	(
		SELECT count(*)
		FROM detention_occurrences AS dto
		WHERE dto.organization_id = sp.organization_id
			AND dto.business_unit_id = sp.business_unit_id
			AND dto.shipment_id = sp.id
			AND dto.status = $11
	) AS detention_accruing,
	(
		SELECT count(*)
		FROM detention_occurrences AS dto
		WHERE dto.organization_id = sp.organization_id
			AND dto.business_unit_id = sp.business_unit_id
			AND dto.shipment_id = sp.id
			AND dto.status = ANY($12)
	) AS detention_unapproved,
	(
		SELECT max(sc.created_at)
		FROM shipment_comments AS sc
		WHERE sc.organization_id = sp.organization_id
			AND sc.business_unit_id = sp.business_unit_id
			AND sc.shipment_id = sp.id
			AND sc.type = $13
			AND sc.metadata ->> $14 = ANY($15)
	) AS customer_notified_at,
	EXISTS (
		SELECT 1
		FROM billing_queue_items AS bqi
		WHERE bqi.organization_id = sp.organization_id
			AND bqi.business_unit_id = sp.business_unit_id
			AND bqi.shipment_id = sp.id
			AND bqi.status <> $16
	) AS in_billing_queue
FROM shipments AS sp
WHERE sp.organization_id = $17
	AND sp.business_unit_id = $18
	AND sp.id = $19
`

func (r *repository) ShipmentFacts(
	ctx context.Context,
	req *deskcaseports.GetShipmentCaseFactsRequest,
) (*deskcase.ShipmentFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ShipmentFacts, error) {
		closedDocuments := []string{
			string(document.StatusRejected),
			string(document.StatusArchived),
		}
		rows, err := keel.QueryAll(
			ctx,
			r.db.Keel(ctx),
			pgx.RowToStructByPos[shipmentFactsRow],
			shipmentFactsQuery,
			permission.ResourceShipment.String(),
			closedDocuments,
			req.PODCode,
			permission.ResourceShipment.String(),
			closedDocuments,
			string(shipment.CarrierAssignmentStatusConfirmed),
			string(rateconfirmation.StatusConfirmed),
			string(shipment.CarrierAssignmentStatusCanceled),
			string(billingqueue.StatusCanceled),
			[]string{
				string(billingqueue.IssueAccessorialNotOnRateCon),
				string(billingqueue.IssueChargeOverRateCon),
			},
			string(detention.OccurrenceStatusAccruing),
			[]string{
				string(detention.OccurrenceStatusPending),
				string(detention.OccurrenceStatusDisputed),
			},
			string(shipment.CommentTypeCustomerUpdate),
			shipment.CommentMetadataTool,
			shipment.CustomerEmailTools(),
			string(billingqueue.StatusCanceled),
			req.TenantInfo.OrgID.String(),
			req.TenantInfo.BuID.String(),
			req.ShipmentID.String(),
		)
		if err != nil {
			return nil, fmt.Errorf("read the shipment's case facts: %w", err)
		}
		if len(rows) == 0 {
			return nil, shipmentErrors.Map(keel.ErrNotFound, req.ShipmentID.String())
		}

		row := &rows[0]
		var rateCons []rateConRow
		if len(row.RateCons) > 0 {
			if err = sonic.Unmarshal(row.RateCons, &rateCons); err != nil {
				return nil, fmt.Errorf("decode the shipment's rate confirmations: %w", err)
			}
		}
		facts := &deskcase.ShipmentFacts{
			Status:              row.Status,
			DeliveredAt:         row.DeliveredAt,
			PODCode:             req.PODCode,
			PODOnFile:           row.PODOnFile,
			RateCons:            make([]deskcase.RateCon, 0, len(rateCons)),
			OpenChargeIssues:    row.OpenChargeIssues,
			DetentionAccruing:   int(row.DetentionAccruing),
			DetentionUnapproved: int(row.DetentionUnapprove),
			CustomerNotifiedAt:  row.CustomerNotifiedAt,
			InBillingQueue:      row.InBillingQueue,
		}
		facts.DocumentTypesOnFile = make(map[pulid.ID]struct{}, len(row.DocumentTypeIDs))
		for _, id := range row.DocumentTypeIDs {
			facts.DocumentTypesOnFile[pulid.ID(id)] = struct{}{}
		}
		for _, rateCon := range rateCons {
			facts.RateCons = append(facts.RateCons, deskcase.RateCon(rateCon))
		}

		return facts, nil
	})
}

func (r *repository) InvoiceFacts(
	ctx context.Context,
	req *deskcaseports.GetInvoiceCaseFactsRequest,
) (*deskcase.InvoiceFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.InvoiceFacts, error) {
		entity, err := invoiceFactsQuery(req.InvoiceID).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, invoiceErrors.Map(err, req.InvoiceID.String())
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
	req *deskcaseports.ListCasePartiesRequest,
) ([]deskcase.Party, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]deskcase.Party, error) {
		db := r.db.Keel(ctx)
		out := make([]deskcase.Party, 0, len(req.CarrierIDs)+1)

		if req.CustomerID.IsNotNil() {
			c := customerdb.Customers
			entity, err := c.Pick(c.ID, c.Name).Where(c.ID.Eq(req.CustomerID)).One(ctx, db)
			if err != nil && !errors.Is(err, keel.ErrNotFound) {
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
			carriers, err := carriersQuery(req.CarrierIDs).All(ctx, db)
			if err != nil {
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

func idStrings(ids []pulid.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}

	return out
}

func invoiceRecordsQuery(ids []pulid.ID) *keel.Query[invoice.Invoice] {
	inv := invoicedb.Invoices

	return inv.Pick(inv.ID, inv.Number, inv.Status, inv.SettlementStatus, inv.CustomerID).
		Where(inv.ID.In(ids...))
}

func disputeRecordsQuery(ids []pulid.ID) *keel.Query[invoice.InvoiceDispute] {
	d := invoicedb.InvoiceDisputes

	return d.Pick(
		d.ID,
		d.Status,
		keel.Raw("COALESCE(?, '')", d.Resolution),
		d.CustomerID,
		d.InvoiceID,
		keel.Raw("COALESCE(?, '')", d.Invoice().Number),
	).Where(d.ID.In(ids...))
}

func invoiceFactsQuery(id pulid.ID) *keel.Query[invoice.Invoice] {
	inv := invoicedb.Invoices

	return inv.Pick(inv.Status, inv.SendStatus, inv.DisputeStatus, inv.SettlementStatus, inv.DueDate).
		Where(inv.ID.Eq(id))
}

func carriersQuery(ids []pulid.ID) *keel.Query[carrier.Carrier] {
	ca := carrierdb.Carriers

	return ca.Pick(ca.ID, ca.Name).Where(ca.ID.In(ids...)).Order(ca.Name.Asc())
}
