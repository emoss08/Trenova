package ediboardreader

import (
	"context"
	"math"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediboard"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediinboundfile/ediinboundfiledb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	tenderAlias = "tender"
	answerAlias = "answer"

	rowsPerLine = 3
)

type PartnerDay struct{ db *postgres.Connection }

func NewPartnerDay(p Params) *PartnerDay { return &PartnerDay{db: p.DB} }

func (*PartnerDay) Metric() string { return ediboard.MetricPartnerDay }

type partnerRow struct {
	ID   pulid.ID
	Name string
}

type tenderRow struct {
	FileID         pulid.ID
	FileStatus     string
	FileReason     string
	ReceivedAt     int64
	TenderID       pulid.ID
	TenderStatus   string
	TransferID     pulid.ID
	TransferStatus string
	TransferReason string
	BOL            string
	MadeShipment   pulid.ID
	ShipmentID     pulid.ID
	ProNumber      string
	AnswerID       pulid.ID
	AnswerStatus   string
	Delivery       string
	Ack            string
	DeliveryError  string
	AckError       string
}

type dayFigures struct {
	Lines     int64
	Tenders   int64
	Shipments int64
	Visible   int64
	Answers   int64
	Delivered int64
	NeedHand  int64
	Median    *float64
}

type dayRead struct {
	partner *partnerRow
	rows    []tenderRow
	figures dayFigures
}

type dayTables struct {
	file     *ediinboundfiledb.EDIInboundFilesTable
	tender   *edimessagedb.EDIMessagesTable
	transfer *editransferdb.EDITransfersTable
	shipment *shipmentdb.ShipmentsTable
	answer   *edimessagedb.EDIMessagesTable
}

func newDayTables() *dayTables {
	return &dayTables{
		file:     ediinboundfiledb.EDIInboundFiles,
		tender:   edimessagedb.EDIMessages.As(tenderAlias),
		transfer: editransferdb.EDITransfers,
		shipment: shipmentdb.Shipments,
		answer:   edimessagedb.EDIMessages.As(answerAlias),
	}
}

type dayWindow struct {
	partner pulid.ID
	start   int64
	end     int64
}

func (r *PartnerDay) Read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
) (*boardcatalog.MetricRead, error) {
	if req.For.IsNil() {
		return nil, nil //nolint:nilnil // no partner is a metric the board cannot read
	}
	first, _, ok := req.Period.Days(req.Now, req.Location)
	if !ok {
		return nil, nil //nolint:nilnil // a period this metric does not take reads as nothing
	}
	window := dayWindow{
		partner: req.For,
		start:   first.Unix(),
		end:     first.AddDate(0, 0, 1).Unix(),
	}
	read, err := r.read(ctx, req, &window)
	if err != nil || read.partner == nil {
		return nil, err
	}

	return partnerDay(ctx, &partnerDayInput{
		partner: read.partner.Name,
		period:  req.Period,
		rows:    read.rows,
		figures: &read.figures,
	}), nil
}

func (r *PartnerDay) read(
	ctx context.Context,
	req *boardreaderports.MetricRequest,
	window *dayWindow,
) (*dayRead, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*dayRead, error) {
		p := edidb.EDIPartners
		tables := newDayTables()
		var (
			partners []partnerRow
			rows     []tenderRow
			figures  []dayFigures
		)
		err := keel.Batch(ctx, r.db.Keel(ctx),
			keel.IntoSlice(p.Pick(p.ID, text(p.Name)).Where(p.ID.Eq(window.partner)), &partners),
			keel.IntoSlice(tenderLines(tables, req, window), &rows),
			keel.IntoSlice(dayTotals(tables, req, window), &figures),
		)
		if err != nil {
			return nil, err
		}
		out := &dayRead{rows: rows}
		if len(partners) > 0 {
			out.partner = &partners[0]
		}
		if len(figures) > 0 {
			out.figures = figures[0]
		}

		return out, nil
	})
}

func dayJoins(
	tables *dayTables,
	req *boardreaderports.MetricRequest,
	window *dayWindow,
) keel.Query[*ediinboundfile.EDIInboundFile] {
	f, m, t, s, a := tables.file, tables.tender, tables.transfer, tables.shipment, tables.answer

	return f.LeftJoin(m,
		m.InboundFileID.EqCol(f.ID),
		m.Direction.Eq(edidocument.DocumentDirectionInbound),
		m.TransactionSet.Eq(edidocument.TransactionSet204),
	).
		LeftJoin(t,
			t.InboundMessageID.EqCol(m.ID),
			t.TargetOrganizationID.Eq(req.Tenant.OrgID),
			t.TargetBusinessUnitID.Eq(req.Tenant.BuID),
		).
		LeftJoin(s, s.ID.EqCol(t.TargetShipmentID)).
		LeftJoin(a,
			a.TransferID.EqCol(t.ID),
			a.Direction.Eq(edidocument.DocumentDirectionOutbound),
			a.TransactionSet.Eq(edidocument.TransactionSet990),
		).
		Where(
			f.EDIPartnerID.Eq(window.partner),
			f.ReceivedAt.Gte(window.start),
			f.ReceivedAt.Lt(window.end),
			keel.Or(
				m.ID.IsNotNull(),
				f.Status.NotIn(
					ediinboundfile.InboundFileStatusProcessed,
					ediinboundfile.InboundFileStatusDuplicate,
				),
			),
		)
}

func tenderLines(
	tables *dayTables,
	req *boardreaderports.MetricRequest,
	window *dayWindow,
) keel.Query[*ediinboundfile.EDIInboundFile] {
	f, m, t, s, a := tables.file, tables.tender, tables.transfer, tables.shipment, tables.answer

	return dayJoins(tables, req, window).
		Pick(
			f.ID, text(f.Status), text(f.FailureReason), f.ReceivedAt,
			m.ID, text(m.Status),
			t.ID, text(t.Status), text(t.FailureReason.Coalesce(t.RejectionReason)),
			t.TenderPayload.Text(tenderBOLKey).Coalesce(""),
			t.TargetShipmentID, s.ID, text(s.ProNumber),
			a.ID, text(a.Status), text(a.DeliveryStatus), text(a.AckStatus),
			text(a.DeliveryLastError), text(a.AckLastError),
		).
		Order(f.ReceivedAt.Desc(), f.ID.Desc(), m.ID.Asc(), a.GeneratedAt.Desc()).
		Limit(ediboard.PartnerDayRows * rowsPerLine)
}

func dayTotals(
	tables *dayTables,
	req *boardreaderports.MetricRequest,
	window *dayWindow,
) keel.Query[*ediinboundfile.EDIInboundFile] {
	f, m, t, s, a := tables.file, tables.tender, tables.transfer, tables.shipment, tables.answer
	line := m.ID.Text().Coalesce(f.ID.Text())

	return dayJoins(tables, req, window).Pick(
		keel.CountOf(line).Distinct(),
		keel.CountOf(m.ID).Distinct(),
		keel.CountOf(t.TargetShipmentID).Distinct(),
		keel.CountOf(s.ID).Distinct(),
		keel.CountOf(a.ID).Distinct(),
		keel.CountOf(a.ID).Distinct().Filter(
			a.DeliveryStatus.Eq(edimessage.MessageDeliveryStatusSent),
		),
		keel.CountOf(line).Distinct().Filter(needsHand(tables)),
		keel.PercentileCont(0.5, a.DeliverySentAt.Sub(f.ReceivedAt).Asc()).
			Filter(a.DeliverySentAt.IsNotNull()),
	)
}

func needsHand(tables *dayTables) keel.Pred {
	f, m, t, a := tables.file, tables.tender, tables.transfer, tables.answer

	return keel.Or(
		f.Status.In(
			ediinboundfile.InboundFileStatusQuarantined,
			ediinboundfile.InboundFileStatusPartiallyProcessed,
		),
		m.Status.Eq(edimessage.MessageStatusFailed),
		t.Status.In(
			editransfer.TransferStatusMappingRequired,
			editransfer.TransferStatusPendingApproval,
			editransfer.TransferStatusFailed,
		),
		a.Status.Eq(edimessage.MessageStatusFailed),
		a.DeliveryStatus.In(
			edimessage.MessageDeliveryStatusFailed,
			edimessage.MessageDeliveryStatusDeadLettered,
		),
		a.AckStatus.In(
			edimessage.MessageAcknowledgmentStatusRejected,
			edimessage.MessageAcknowledgmentStatusFailed,
			edimessage.MessageAcknowledgmentStatusAcceptedWithErrors,
		),
	)
}

func medianSeconds(median *float64) *boardcatalog.Value {
	if median == nil || math.IsNaN(*median) {
		return nil
	}
	value := boardcatalog.Duration(int64(math.Round(max(*median, 0))))

	return &value
}
