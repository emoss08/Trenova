package triageinsightrepository

import (
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/shipmentcol"
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

func messageRowColumns(sq *bun.SelectQuery) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	sp := shipmentcol.ShipmentColumns

	return sq.
		ColumnExpr(cols.ID.Qualified()).
		ColumnExpr(cols.OrganizationID.Qualified()).
		ColumnExpr(cols.BusinessUnitID.Qualified()).
		ColumnExpr(cols.MailboxID.Qualified()).
		ColumnExpr(cols.FromAddress.Qualified()).
		ColumnExpr(cols.FromName.Qualified()).
		ColumnExpr(cols.Subject.Qualified()).
		ColumnExpr(cols.SenderDomain.Qualified()).
		ColumnExpr(cols.Classification.Qualified()).
		ColumnExpr(cols.Status.Qualified()).
		ColumnExpr(cols.ReceivedAt.Qualified()).
		ColumnExpr(cols.DueAt.Qualified()).
		ColumnExpr(cols.Parked.Qualified()).
		ColumnExpr(cols.OwnerID.Qualified()).
		ColumnExpr(cols.MatchedShipmentID.Qualified()).
		ColumnExpr(cols.MatchedCustomerID.Qualified()).
		ColumnExpr(cols.MatchedCarrierID.Qualified()).
		ColumnExpr(cols.HandledByRuleID.Qualified()).
		ColumnExpr(cols.HandledAction.Qualified()).
		ColumnExpr(cols.ClassificationCorrected.Qualified()).
		ColumnExpr(cols.ReviewedBy.Qualified()).
		ColumnExpr(cols.ReviewedAt.Qualified()).
		ColumnExpr(sp.ProNumber.Expr("COALESCE({}, '') AS " + referenceColumn)).
		Join("LEFT JOIN " + shipmentcol.ShipmentTable.Name + " AS " + shipmentcol.ShipmentTable.Alias).
		JoinOn(sp.ID.EqColumn(cols.MatchedShipmentID)).
		JoinOn(sp.OrganizationID.EqColumn(cols.OrganizationID)).
		JoinOn(sp.BusinessUnitID.EqColumn(cols.BusinessUnitID))
}

var messageRowNames = buildMessageRowNames()

func buildMessageRowNames() []string {
	cols := inboundmessagecol.InboundMessageColumns

	return []string{
		cols.ID.Bare(),
		cols.OrganizationID.Bare(),
		cols.BusinessUnitID.Bare(),
		cols.MailboxID.Bare(),
		cols.FromAddress.Bare(),
		cols.FromName.Bare(),
		cols.Subject.Bare(),
		cols.SenderDomain.Bare(),
		cols.Classification.Bare(),
		cols.Status.Bare(),
		cols.ReceivedAt.Bare(),
		cols.DueAt.Bare(),
		cols.Parked.Bare(),
		cols.OwnerID.Bare(),
		cols.MatchedShipmentID.Bare(),
		cols.MatchedCustomerID.Bare(),
		cols.MatchedCarrierID.Bare(),
		cols.HandledByRuleID.Bare(),
		cols.HandledAction.Bare(),
		cols.ClassificationCorrected.Bare(),
		cols.ReviewedBy.Bare(),
		cols.ReviewedAt.Bare(),
		referenceColumn,
	}
}

const referenceColumn = "reference"

func rankedRowColumns(sq *bun.SelectQuery, alias string, extra ...string) *bun.SelectQuery {
	for _, name := range messageRowNames {
		sq = sq.ColumnExpr(alias + "." + name)
	}
	for _, name := range extra {
		sq = sq.ColumnExpr(alias + "." + name)
	}

	return sq
}

func messages(db bun.IDB) *bun.SelectQuery {
	return db.NewSelect().Model((*inboundmessage.InboundMessage)(nil))
}

func settledByRule() string {
	cols := inboundmessagecol.InboundMessageColumns

	return "(" + cols.HandledByRuleID.Qualified() + " IS NOT NULL AND " +
		cols.ReviewedBy.Qualified() + " IS NULL AND " + cols.Status.In() + ")"
}

func settledByPerson() string {
	cols := inboundmessagecol.InboundMessageColumns

	return "(" + cols.ReviewedBy.Qualified() + " IS NOT NULL AND " + cols.Status.In() + ")"
}
