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
	"github.com/emoss08/trenova/internal/modules/billing/domain/billingqueue/billingqueuedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoice/invoicedb"
	"github.com/emoss08/trenova/internal/modules/billing/domain/invoicedispute"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document"
	"github.com/emoss08/trenova/internal/modules/documents/domain/document/documentdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention"
	"github.com/emoss08/trenova/internal/modules/operations/domain/detention/detentiondb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier/carrierdb"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer/customerdb"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation"
	"github.com/emoss08/trenova/internal/modules/rating/domain/rateconfirmation/rateconfirmationdb"
	"github.com/emoss08/trenova/internal/modules/reference/domain/documenttype/documenttypedb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
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

func shipmentRecordsQuery(ids []pulid.ID) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	casn := shipmentdb.CarrierAssignments
	smv := shipmentdb.ShipmentMoves
	nxt := shipmentdb.Stops.As("nxt")
	nmv := shipmentdb.ShipmentMoves.As("nmv")

	next := nxt.Join(nmv, nmv.ID.EqCol(nxt.ShipmentMoveID)).
		Pick(nxt.ID, nxt.ScheduledWindowStart).
		Where(
			nmv.ShipmentID.EqCol(sp.ID),
			nmv.Status.Ne(shipment.MoveStatusCanceled),
			nxt.Status.Ne(shipment.StopStatusCanceled),
			nxt.ActualArrival.IsNull(),
		).
		Order(nmv.Sequence.Asc(), nxt.Sequence.Asc()).
		Limit(1)
	carrierIDs := keel.ArrayOf(casn.Join(smv, smv.ID.EqCol(casn.ShipmentMoveID)).
		Pick(casn.CarrierID).
		Where(
			smv.ShipmentID.EqCol(sp.ID),
			casn.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
		).
		Distinct())

	return sp.LeftJoinLateral(next).
		Pick(
			sp.ID,
			sp.ProNumber.Coalesce(""),
			sp.Status,
			sp.CustomerID,
			sp.BillToCustomerID.Coalesce(sp.CustomerID),
			nxt.ID.Coalesce(""),
			nxt.ScheduledWindowStart,
			carrierIDs,
		).
		Where(sp.ID.In(ids...))
}

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
			if err := r.shipmentRecords(ctx, ids, out); err != nil {
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
	ids []pulid.ID,
	out map[pulid.ID]*deskcase.Record,
) error {
	rows, err := keel.Into[shipmentRecordRow](ctx, r.db.Keel(ctx), shipmentRecordsQuery(ids))
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
	Status        invoicedispute.DisputeCaseStatus
	Resolution    invoicedispute.DisputeResolution
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
		case invoicedispute.DisputeCaseStatusResolved, invoicedispute.DisputeCaseStatusWithdrawn:
			record.Closed = true
			record.ClosedAs = string(row.Status)
			if row.Resolution != "" {
				record.ClosedAs = string(row.Resolution)
			}
		case invoicedispute.DisputeCaseStatusOpen:
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

func shipmentFactsQuery(
	req *deskcaseports.GetShipmentCaseFactsRequest,
) *keel.Query[*shipment.Shipment] {
	sp := shipmentdb.Shipments
	doc := documentdb.Documents
	dt := documenttypedb.DocumentTypes
	casn := shipmentdb.CarrierAssignments
	smv := shipmentdb.ShipmentMoves
	car := carrierdb.Carriers
	rc := rateconfirmationdb.RateConfirmations
	bqis := billingqueuedb.Issues
	bqi := billingqueuedb.BillingQueueItems
	dto := detentiondb.DetentionOccurrences
	sc := shipmentdb.ShipmentComments
	shipmentResource := permission.ResourceShipment.String()

	currentDocuments := func() []keel.Pred {
		return []keel.Pred{
			doc.ResourceType.Eq(shipmentResource),
			doc.ResourceID.EqCol(sp.ID),
			doc.IsCurrentVersion.IsTrue(),
			doc.Status.NotIn(document.StatusRejected, document.StatusArchived),
		}
	}
	podOnFile := keel.Exists(doc.Join(dt, dt.ID.EqCol(doc.DocumentTypeID)).
		Pick(doc.ID).
		Where(append(
			currentDocuments(),
			dt.Code.Upper().EqExpr(keel.Text(req.PODCode).Upper()),
		)...))
	documentTypeIDs := keel.ArrayOf(doc.Pick(doc.DocumentTypeID).
		Where(append(currentDocuments(), doc.DocumentTypeID.IsNotNull())...).
		Distinct())
	confirmed := keel.Or(
		casn.Status.Eq(shipment.CarrierAssignmentStatusConfirmed),
		keel.Exists(rc.Pick(rc.ID).Where(
			rc.CarrierAssignmentID.EqCol(casn.ID),
			rc.Status.Eq(rateconfirmation.StatusConfirmed),
		)),
	)
	rateCons := keel.JSONBOf(keel.Scalar(casn.Join(smv, smv.ID.EqCol(casn.ShipmentMoveID)).
		LeftJoin(car, car.ID.EqCol(casn.CarrierID)).
		Pick(keel.JSONBAgg(keel.JSONBuildObject(
			keel.Pair("carrierName", car.Name.Coalesce("")),
			keel.Pair("confirmed", confirmed),
		)).OrderBy(smv.Sequence.Asc())).
		Where(
			smv.ShipmentID.EqCol(sp.ID),
			casn.Status.Ne(shipment.CarrierAssignmentStatusCanceled),
		))).Coalesce(keel.JSONBuildArray())
	openChargeIssues := keel.ArrayOf(bqis.Join(bqi, bqi.ID.EqCol(bqis.ItemID)).
		Pick(bqis.Code).
		Where(
			bqi.ShipmentID.EqCol(sp.ID),
			bqi.Status.Ne(billingqueue.StatusCanceled),
			bqis.ResolutionKey.IsNull(),
			bqis.Code.In(
				billingqueue.IssueAccessorialNotOnRateCon,
				billingqueue.IssueChargeOverRateCon,
			),
		).
		Distinct())
	detentionCount := func(statuses ...detention.OccurrenceStatus) keel.Expr {
		return keel.Scalar(dto.Pick(keel.Count()).
			Where(dto.ShipmentID.EqCol(sp.ID), dto.Status.In(statuses...)))
	}
	customerNotifiedAt := keel.Scalar(sc.Pick(sc.CreatedAt.Max()).Where(
		sc.ShipmentID.EqCol(sp.ID),
		sc.Type.Eq(shipment.CommentTypeCustomerUpdate),
		sc.Metadata.Text(shipment.CommentMetadataTool).In(shipment.CustomerEmailTools()...),
	))
	inBillingQueue := keel.Exists(bqi.Pick(bqi.ID).Where(
		bqi.ShipmentID.EqCol(sp.ID),
		bqi.Status.Ne(billingqueue.StatusCanceled),
	))

	return sp.Pick(
		sp.Status,
		sp.ActualDeliveryDate,
		podOnFile,
		documentTypeIDs,
		rateCons,
		openChargeIssues,
		detentionCount(detention.OccurrenceStatusAccruing),
		detentionCount(detention.OccurrenceStatusPending, detention.OccurrenceStatusDisputed),
		customerNotifiedAt,
		inBillingQueue,
	).Where(sp.ID.Eq(req.ShipmentID))
}

func (r *repository) ShipmentFacts(
	ctx context.Context,
	req *deskcaseports.GetShipmentCaseFactsRequest,
) (*deskcase.ShipmentFacts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*deskcase.ShipmentFacts, error) {
		rows, err := keel.Into[shipmentFactsRow](ctx, r.db.Keel(ctx), shipmentFactsQuery(req))
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

func invoiceRecordsQuery(ids []pulid.ID) *keel.Query[*invoice.Invoice] {
	inv := invoicedb.Invoices

	return inv.Pick(inv.ID, inv.Number, inv.Status, inv.SettlementStatus, inv.CustomerID).
		Where(inv.ID.In(ids...))
}

func disputeRecordsQuery(ids []pulid.ID) *keel.Query[*invoicedispute.InvoiceDispute] {
	d := invoicedb.InvoiceDisputes

	return d.Pick(
		d.ID,
		d.Status,
		d.Resolution.Coalesce(""),
		d.CustomerID,
		d.InvoiceID,
		d.Invoice().Number.Coalesce(""),
	).Where(d.ID.In(ids...))
}

func invoiceFactsQuery(id pulid.ID) *keel.Query[*invoice.Invoice] {
	inv := invoicedb.Invoices

	return inv.Pick(inv.Status, inv.SendStatus, inv.DisputeStatus, inv.SettlementStatus, inv.DueDate).
		Where(inv.ID.Eq(id))
}

func carriersQuery(ids []pulid.ID) *keel.Query[*carrier.Carrier] {
	ca := carrierdb.Carriers

	return ca.Pick(ca.ID, ca.Name).Where(ca.ID.In(ids...)).Order(ca.Name.Asc())
}
