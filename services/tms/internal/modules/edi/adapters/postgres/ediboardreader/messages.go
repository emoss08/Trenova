package ediboardreader

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edi/edidb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/ediboard"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage"
	"github.com/emoss08/trenova/internal/modules/edi/domain/edimessage/edimessagedb"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer/editransferdb"
	"github.com/emoss08/trenova/internal/modules/identity/domain/recordkind"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardcatalog"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/boardreading"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/boardreaderports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
)

const tenderBOLKey = "bol"

type Messages struct{ db *postgres.Connection }

type MessageWaiting struct{ db *postgres.Connection }

func NewMessages(p Params) *Messages { return &Messages{db: p.DB} }

func NewMessageWaiting(p Params) *MessageWaiting { return &MessageWaiting{db: p.DB} }

func (*Messages) Kind() recordkind.Kind       { return recordkind.EDIMessage }
func (*MessageWaiting) Kind() recordkind.Kind { return recordkind.EDIMessage }
func (*MessageWaiting) Measure() string       { return ediboard.MeasureWaiting }

type messageRow struct {
	ID             pulid.ID
	Direction      string
	TransactionSet string
	Status         string
	Delivery       string
	Ack            string
	DeliveryError  string
	AckError       string
	GeneratedAt    int64
	SentAt         *int64
	AckReceivedAt  *int64
	Partner        string
	ProNumber      string
	TenderBOL      string
}

func (row *messageRow) state() *messageState {
	return &messageState{
		Direction: row.Direction,
		Status:    row.Status,
		Delivery:  row.Delivery,
		Ack:       row.Ack,
	}
}

func queryMessages(
	ctx context.Context,
	db *postgres.Connection,
	req *boardreaderports.RecordRequest,
) ([]messageRow, error) {
	m := edimessagedb.EDIMessages
	p := edidb.EDIPartners
	s := shipmentdb.Shipments
	t := editransferdb.EDITransfers

	return keel.Into[messageRow](ctx, db.Keel(ctx),
		m.LeftJoin(p, p.ID.EqCol(m.EDIPartnerID)).
			LeftJoin(s, s.ID.EqCol(m.ShipmentID)).
			LeftJoin(t, t.ID.EqCol(m.TransferID), transferOfTenant(req.Tenant)).
			Pick(
				m.ID, text(m.Direction), text(m.TransactionSet), text(m.Status),
				text(m.DeliveryStatus), text(m.AckStatus), text(m.DeliveryLastError),
				text(m.AckLastError), m.GeneratedAt, m.DeliverySentAt, m.AckReceivedAt,
				text(p.Name), text(s.ProNumber), t.TenderPayload.Text(tenderBOLKey).Coalesce(""),
			).
			Where(m.ID.In(req.IDs...), keel.Or(m.ShipmentID.IsNull(), s.ID.IsNotNull())))
}

func transferOfTenant(tenant pagination.TenantInfo) keel.Pred {
	t := editransferdb.EDITransfers

	return keel.Or(
		keel.And(t.TargetOrganizationID.Eq(tenant.OrgID), t.TargetBusinessUnitID.Eq(tenant.BuID)),
		keel.And(t.SourceOrganizationID.Eq(tenant.OrgID), t.SourceBusinessUnitID.Eq(tenant.BuID)),
	)
}

func (r *Messages) Summaries(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Summary, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Summary{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Summary, error) {
			rows, err := queryMessages(ctx, r.db, req)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Summary, len(rows))
			for idx := range rows {
				row := &rows[idx]
				status, tone := messageStatus(ctx, row.state())
				out[row.ID] = boardreading.Summary{
					Label:      messageLabel(ctx, row),
					Subtitle:   strings.TrimSpace(row.Partner),
					Status:     status,
					StatusTone: tone,
				}
			}

			return out, nil
		},
	)
}

func messageLabel(ctx context.Context, row *messageRow) string {
	set := strings.TrimSpace(row.TransactionSet)
	if bol := strings.TrimSpace(row.TenderBOL); bol != "" {
		return i18n.T(ctx, "{0} for tender {1}", set, bol)
	}
	if pro := strings.TrimSpace(row.ProNumber); pro != "" {
		return i18n.T(ctx, "{0} for {1}", set, pro)
	}
	partner := strings.TrimSpace(row.Partner)
	if partner == "" {
		return set
	}
	if edidocument.DocumentDirection(row.Direction) == edidocument.DocumentDirectionInbound {
		return i18n.T(ctx, "{0} from {1}", set, partner)
	}

	return i18n.T(ctx, "{0} to {1}", set, partner)
}

func (r *MessageWaiting) Readings(
	ctx context.Context,
	req *boardreaderports.RecordRequest,
) (map[pulid.ID]boardreading.Reading, error) {
	if len(req.IDs) == 0 {
		return map[pulid.ID]boardreading.Reading{}, nil
	}

	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (map[pulid.ID]boardreading.Reading, error) {
			rows, err := queryMessages(ctx, r.db, req)
			if err != nil {
				return nil, err
			}
			out := make(map[pulid.ID]boardreading.Reading, len(rows))
			for idx := range rows {
				out[rows[idx].ID] = messageWaiting(ctx, &rows[idx], req.Now)
			}

			return out, nil
		},
	)
}

func messageWaiting(ctx context.Context, row *messageRow, now int64) boardreading.Reading {
	reading := boardreading.Reading{Value: boardcatalog.Duration(0)}
	reason := firstLine(stringutils.FirstNonEmpty(row.DeliveryError, row.AckError))
	if reason != "" {
		reading.Fact = &boardreading.Words{
			Text:  "{0}",
			Value: &boardcatalog.Value{Kind: boardcatalog.ValueText, Text: reason},
		}
	}
	state := row.state()
	if !messageNeedsHand(state) && !messageUnderWay(state) {
		reading.Standing = boardreading.Words{Text: messageDoneWords(ctx, row), At: doneAt(row)}

		return reading
	}
	waited := boardcatalog.Duration(max(now-row.GeneratedAt, 0))
	reading.Value = waited
	reading.Standing = boardreading.Words{Text: i18n.T(ctx, "{0} waiting"), Value: &waited}
	if messageNeedsHand(state) {
		reading.Tone = boardcatalog.ToneWarn
	}

	return reading
}

func messageUnderWay(state *messageState) bool {
	if edidocument.DocumentDirection(state.Direction) == edidocument.DocumentDirectionInbound {
		return false
	}
	switch edimessage.MessageDeliveryStatus(state.Delivery) {
	case edimessage.MessageDeliveryStatusQueued, edimessage.MessageDeliveryStatusSending:
		return true
	case edimessage.MessageDeliveryStatusSent:
		return edimessage.MessageAcknowledgmentStatus(state.Ack) ==
			edimessage.MessageAcknowledgmentStatusPending
	case edimessage.MessageDeliveryStatusFailed, edimessage.MessageDeliveryStatusDeadLettered:
	}

	return false
}

func messageDoneWords(ctx context.Context, row *messageRow) string {
	if edidocument.DocumentDirection(row.Direction) == edidocument.DocumentDirectionInbound {
		return i18n.T(ctx, "Received {0}")
	}
	if edimessage.MessageAcknowledgmentStatus(row.Ack) ==
		edimessage.MessageAcknowledgmentStatusAccepted {
		return i18n.T(ctx, "Acknowledged {0}")
	}
	if row.SentAt != nil {
		return i18n.T(ctx, "Delivered {0}")
	}

	return i18n.T(ctx, "Generated {0}")
}

func doneAt(row *messageRow) int64 {
	switch {
	case row.AckReceivedAt != nil && *row.AckReceivedAt > 0 &&
		edimessage.MessageAcknowledgmentStatus(row.Ack) ==
			edimessage.MessageAcknowledgmentStatusAccepted:
		return *row.AckReceivedAt
	case row.SentAt != nil && *row.SentAt > 0:
		return *row.SentAt
	}

	return row.GeneratedAt
}
