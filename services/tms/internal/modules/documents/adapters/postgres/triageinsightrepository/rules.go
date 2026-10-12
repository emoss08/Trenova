package triageinsightrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

const (
	outcomesCTE      = "manual_outcomes"
	arrivalsCTE      = "manual_arrivals"
	outcomeAlias     = "mo"
	arrivalAlias     = "ma"
	outcomeMessageID = "message_id"
	outcomeColumn    = "outcome"
)

func (r *repository) RuleCandidates(
	ctx context.Context,
	req triageports.RuleCandidatesRequest,
) (*triageports.RuleCandidates, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.RuleCandidates, error) {
		db := r.db.DBForContext(ctx)

		counted, err := ruleCandidatesCountQuery(db, req).Count(ctx)
		if err != nil {
			return nil, err
		}
		total := int(counted)

		rows := make([]*triageports.MessageRow, 0, min(req.Limit, total))
		if total > 0 {
			if err = ruleCandidatesQuery(db, req).Scan(ctx, &rows); err != nil {
				return nil, err
			}
		}

		return &triageports.RuleCandidates{Total: total, Rows: rows}, nil
	})
}

func ruleCandidatesCountQuery(db bun.IDB, req triageports.RuleCandidatesRequest) *bun.SelectQuery {
	return messages(db).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery { return narrowToRule(sq, req) })
}

func ruleCandidatesQuery(db bun.IDB, req triageports.RuleCandidatesRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	att := inboundmessagecol.InboundAttachmentColumns

	personFiled := db.NewSelect().
		Model((*inboundmessage.InboundAttachment)(nil)).
		ColumnExpr("1").
		Where(att.MessageID.EqColumn(cols.ID)).
		Where(att.OrganizationID.EqColumn(cols.OrganizationID)).
		Where(att.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
		Where(att.FiledAt.IsNotNull()).
		Where(att.FiledBy.IsNotNull())

	return messages(db).
		Apply(messageRowColumns).
		ColumnExpr("EXISTS (?) AS person_filed", personFiled).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery { return narrowToRule(sq, req) }).
		Order(cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(int64(req.Limit))
}

func narrowToRule(
	sq *bun.SelectQuery,
	req triageports.RuleCandidatesRequest,
) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	sq = sq.
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.Classification.Eq(), req.Classification).
		Where(cols.ReceivedAt.Gte(), req.Since).
		Where(cols.NovaRead.IsTrue())
	if req.MailboxID.IsNotNil() {
		sq = sq.Where(cols.MailboxID.Eq(), req.MailboxID)
	}

	switch req.SenderScope {
	case triage.SenderDomain:
		domain := strings.ToLower(req.Sender)
		sq = sq.WhereGroup(" AND ", func(q *bun.SelectQuery) *bun.SelectQuery {
			return q.
				Where(cols.SenderDomain.Eq(), domain).
				WhereOr(cols.SenderDomain.Like(), "%."+stringutils.EscapeLikePattern(domain))
		})
	case triage.SenderAddress:
		sq = sq.
			Where(cols.SenderDomain.Eq(), emailutils.Domain(req.Sender)).
			Where(cols.FromAddress.Expr("LOWER(BTRIM({})) = ?"), strings.ToLower(req.Sender))
	case triage.SenderAnyCustomer:
		sq = sq.Where(cols.MatchedCustomerID.IsNotNull())
	case triage.SenderAnyCarrier:
		sq = sq.Where(cols.MatchedCarrierID.IsNotNull())
	case triage.SenderAnyone:
	}

	return sq
}

func (r *repository) ManualOutcomes(
	ctx context.Context,
	req triageports.ManualOutcomesRequest,
) ([]*triageports.ManualOutcomeRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.ManualOutcomeRow, error) {
		rows := make([]*triageports.ManualOutcomeRow, 0, 32)
		if err := manualOutcomesQuery(r.db.DBForContext(ctx), req).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return rows, nil
	})
}

func manualOutcomesQuery(db bun.IDB, req triageports.ManualOutcomesRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	act := triagecol.ActivityColumns
	table := inboundmessagecol.InboundMessageTable

	outcomes := db.NewSelect().
		Model((*triage.Activity)(nil)).
		ColumnExpr(act.MessageID.As(outcomeMessageID)).
		ColumnExpr(
			"CASE WHEN BOOL_OR("+act.Kind.Eq()+") THEN ? WHEN BOOL_OR("+act.Kind.Eq()+
				") THEN ? ELSE ? END AS "+outcomeColumn,
			triage.ActivityIgnored, triage.ActivityIgnored,
			triage.ActivityFiled, triage.ActivityFiled,
			triage.ActivityHandled,
		).
		Apply(triagecol.ActivityApplyTenant(req.TenantInfo)).
		Where(act.ActorKind.Eq(), triage.ActorPerson).
		Where(act.Kind.In(), bun.In([]triage.ActivityKind{
			triage.ActivityFiled,
			triage.ActivityHandled,
			triage.ActivityIgnored,
		})).
		Where(act.CreatedAt.Gte(), req.Since).
		GroupExpr(act.MessageID.Qualified())

	arrivals := messages(db).
		ColumnExpr(cols.SenderDomain.Qualified()).
		ColumnExpr(cols.Classification.Qualified()).
		ColumnExpr("COUNT(*) AS arrived").
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ReceivedAt.Gte(), req.Since).
		Where(cols.Classification.IsNotNull()).
		Where(cols.SenderDomain.NotEq(), "").
		GroupExpr(cols.SenderDomain.Qualified()).
		GroupExpr(cols.Classification.Qualified())

	return db.NewSelect().
		With(outcomesCTE, outcomes).
		With(arrivalsCTE, arrivals).
		TableExpr(outcomesCTE+" AS "+outcomeAlias).
		Join("JOIN "+table.Name+" AS "+table.Alias).
		JoinOn(cols.ID.Qualified()+" = "+outcomeAlias+"."+outcomeMessageID).
		JoinOn(cols.OrganizationID.Eq(), req.TenantInfo.OrgID).
		JoinOn(cols.BusinessUnitID.Eq(), req.TenantInfo.BuID).
		Join("JOIN "+arrivalsCTE+" AS "+arrivalAlias).
		JoinOn(arrivalAlias+"."+cols.SenderDomain.Bare()+" = "+cols.SenderDomain.Qualified()).
		JoinOn(arrivalAlias+"."+cols.Classification.Bare()+" = "+cols.Classification.Qualified()).
		ColumnExpr(cols.SenderDomain.Qualified()).
		ColumnExpr(cols.Classification.Qualified()).
		ColumnExpr(outcomeAlias+"."+outcomeColumn).
		ColumnExpr("COUNT(*) AS count").
		ColumnExpr("MAX("+arrivalAlias+".arrived) AS arrived").
		ColumnExpr("COUNT(DISTINCT "+cols.MailboxID.Qualified()+") AS mailboxes").
		ColumnExpr("MIN("+cols.MailboxID.Qualified()+") AS mailbox_id").
		ColumnExpr(cols.MatchedCustomerID.Expr("BOOL_AND({} IS NOT NULL) AS all_customers")).
		ColumnExpr(cols.MatchedCarrierID.Expr("BOOL_AND({} IS NOT NULL) AS all_carriers")).
		Where(cols.ReceivedAt.Gte(), req.Since).
		GroupExpr(cols.SenderDomain.Qualified()).
		GroupExpr(cols.Classification.Qualified()).
		GroupExpr(outcomeAlias + "." + outcomeColumn)
}

func (r *repository) MessageRows(
	ctx context.Context,
	req triageports.MessageRowsRequest,
) ([]*triageports.MessageRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.MessageRow, error) {
		rows := make([]*triageports.MessageRow, 0, len(req.IDs))
		if len(req.IDs) == 0 {
			return rows, nil
		}

		if err := messageRowsQuery(r.db.DBForContext(ctx), req).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return rows, nil
	})
}

func messageRowsQuery(db bun.IDB, req triageports.MessageRowsRequest) *bun.SelectQuery {
	return messages(db).
		Apply(messageRowColumns).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(inboundmessagecol.InboundMessageColumns.ID.In(), bun.In(req.IDs))
}

func (r *repository) SenderDomains(
	ctx context.Context,
	req triageports.SenderDomainsRequest,
) ([]string, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]string, error) {
		domains := make([]string, 0, req.Limit)
		if err := senderDomainsQuery(r.db.DBForContext(ctx), req).Scan(ctx, &domains); err != nil {
			return nil, err
		}

		return domains, nil
	})
}

func senderDomainsQuery(db bun.IDB, req triageports.SenderDomainsRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	query := messages(db).
		ColumnExpr(cols.SenderDomain.Qualified()).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ReceivedAt.Gte(), req.Since).
		Where(cols.SenderDomain.NotEq(), "")
	if term := strings.ToLower(strings.TrimSpace(req.Query)); term != "" {
		query = query.Where(cols.SenderDomain.Like(), "%"+stringutils.EscapeLikePattern(term)+"%")
	}

	return query.
		GroupExpr(cols.SenderDomain.Qualified()).
		OrderExpr("COUNT(*) DESC").
		Order(cols.SenderDomain.OrderAsc()).
		Limit(int64(req.Limit))
}
