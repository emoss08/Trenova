package triageinsightrepository

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage/inboundmessagedb"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment/shipmentdb"
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

func New(p Params) triageports.TriageInsightRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-insight-repository"),
	}
}

var (
	needsReview = []inboundmessage.Status{
		inboundmessage.StatusInReview,
		inboundmessage.StatusQuarantined,
	}
	settled = []inboundmessage.Status{
		inboundmessage.StatusActioned,
		inboundmessage.StatusIgnored,
	}
)

var messageRowNames = []string{
	"id",
	"organization_id",
	"business_unit_id",
	"mailbox_id",
	"from_address",
	"from_name",
	"subject",
	"sender_domain",
	"classification",
	"status",
	"received_at",
	"due_at",
	"parked",
	"owner_id",
	"matched_shipment_id",
	"matched_customer_id",
	"matched_carrier_id",
	"handled_by_rule_id",
	"handled_action",
	"classification_corrected",
	"reviewed_by",
	"reviewed_at",
	"reference",
}

type scannedRow[X any] struct {
	ID                      pulid.ID
	OrganizationID          pulid.ID
	BusinessUnitID          pulid.ID
	MailboxID               pulid.ID
	FromAddress             string
	FromName                string
	Subject                 string
	SenderDomain            string
	Classification          inboundmessage.Classification
	Status                  inboundmessage.Status
	ReceivedAt              int64
	DueAt                   int64
	Parked                  inboundmessage.Parking
	OwnerID                 pulid.ID
	MatchedShipmentID       pulid.ID
	MatchedCustomerID       pulid.ID
	MatchedCarrierID        pulid.ID
	HandledByRuleID         pulid.ID
	HandledAction           inboundmessage.NovaAction
	ClassificationCorrected bool
	ReviewedBy              pulid.ID
	ReviewedAt              int64
	Reference               string
	Extra                   X
}

func (s *scannedRow[X]) row() triageports.MessageRow {
	return triageports.MessageRow{
		ID:                      s.ID,
		OrganizationID:          s.OrganizationID,
		BusinessUnitID:          s.BusinessUnitID,
		MailboxID:               s.MailboxID,
		FromAddress:             s.FromAddress,
		FromName:                s.FromName,
		Subject:                 s.Subject,
		SenderDomain:            s.SenderDomain,
		Classification:          s.Classification,
		Status:                  s.Status,
		ReceivedAt:              s.ReceivedAt,
		DueAt:                   s.DueAt,
		Parked:                  s.Parked,
		OwnerID:                 s.OwnerID,
		MatchedShipmentID:       s.MatchedShipmentID,
		MatchedCustomerID:       s.MatchedCustomerID,
		MatchedCarrierID:        s.MatchedCarrierID,
		HandledByRuleID:         s.HandledByRuleID,
		HandledAction:           s.HandledAction,
		ClassificationCorrected: s.ClassificationCorrected,
		ReviewedBy:              s.ReviewedBy,
		ReviewedAt:              s.ReviewedAt,
		Reference:               s.Reference,
	}
}

func messageRows(scanned []scannedRow[bool], personFiled bool) []*triageports.MessageRow {
	rows := make([]*triageports.MessageRow, 0, len(scanned))
	for i := range scanned {
		row := scanned[i].row()
		if personFiled {
			row.PersonFiled = scanned[i].Extra
		}
		rows = append(rows, &row)
	}

	return rows
}

func text(e keel.Expr) keel.StringExpr {
	return keel.StringOf(e).Coalesce("")
}

func number(e keel.Expr) keel.Int64Expr {
	return keel.Int64Of(e).Coalesce(0)
}

func messages() *keel.Query[*inboundmessage.InboundMessage] {
	return inboundmessagedb.InboundMessages.Query()
}

func withReference(
	q *keel.Query[*inboundmessage.InboundMessage],
) *keel.Query[*inboundmessage.InboundMessage] {
	m := inboundmessagedb.InboundMessages
	sp := shipmentdb.Shipments

	return q.LeftJoin(sp, sp.ID.EqCol(m.MatchedShipmentID))
}

func messageRowColumns(extra ...keel.Expr) []keel.Expr {
	m := inboundmessagedb.InboundMessages

	return append([]keel.Expr{
		m.ID,
		m.OrganizationID,
		m.BusinessUnitID,
		m.MailboxID,
		m.FromAddress,
		text(m.FromName),
		text(m.Subject),
		m.SenderDomain,
		text(m.Classification),
		m.Status,
		m.ReceivedAt,
		number(m.DueAt),
		text(m.Parked),
		text(m.OwnerID),
		text(m.MatchedShipmentID),
		text(m.MatchedCustomerID),
		text(m.MatchedCarrierID),
		text(m.HandledByRuleID),
		text(m.HandledAction),
		m.ClassificationCorrected,
		text(m.ReviewedBy),
		number(m.ReviewedAt),
		text(shipmentdb.Shipments.ProNumber),
	}, extra...)
}

func settledByRule() keel.Pred {
	m := inboundmessagedb.InboundMessages

	return keel.And(m.HandledByRuleID.IsNotNull(), m.ReviewedBy.IsNull(), m.Status.In(settled...))
}

func settledByPerson() keel.Pred {
	m := inboundmessagedb.InboundMessages

	return keel.And(m.ReviewedBy.IsNotNull(), m.Status.In(settled...))
}

func rankedColumns(cte *keel.CTETable, extra string) []keel.Expr {
	cols := make([]keel.Expr, 0, len(messageRowNames)+1)
	for _, name := range messageRowNames {
		cols = append(cols, cte.Col(name))
	}

	return append(cols, cte.Col(extra))
}
