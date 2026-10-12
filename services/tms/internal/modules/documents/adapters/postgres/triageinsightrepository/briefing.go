package triageinsightrepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/uptrace/bun"
)

const (
	handledSamplesPerGroup = 3
	rankedAlias            = "ranked"
)

func (r *repository) BriefingCounts(
	ctx context.Context,
	req triageports.BriefingWindow,
) (*triageports.BriefingCounts, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.BriefingCounts, error) {
		counts := new(triageports.BriefingCounts)
		if err := briefingCountsQuery(r.db.DBForContext(ctx), req).Scan(ctx, counts); err != nil {
			return nil, err
		}

		return counts, nil
	})
}

func briefingCountsQuery(db bun.IDB, req triageports.BriefingWindow) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	tenant := inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)
	count := func(apply func(*bun.SelectQuery) *bun.SelectQuery) *bun.SelectQuery {
		return messages(db).ColumnExpr("COUNT(*)").Apply(tenant).Apply(apply)
	}

	waiting := count(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.Status.In(), bun.List(needsReview)).Where(cols.Parked.IsNull())
	})
	overdue := count(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.Status.In(), bun.List(needsReview)).
			Where(cols.Parked.IsNull()).
			Where(cols.DueAt.Lt(), req.Now)
	})
	closed := count(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.Status.In(), bun.List(settled)).
			Where(cols.ReviewedAt.Gte(), req.DayStart).
			Where(cols.ReviewedBy.IsNotNull())
	})
	handled := count(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.HandledByRuleID.IsNotNull()).
			Where(cols.ReceivedAt.Gte(), req.DayStart)
	})
	comingBack := count(func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.Status.In(), bun.List(needsReview)).Where(cols.Parked.IsNotNull())
	})
	anyMessage := messages(db).ColumnExpr("1").Apply(tenant).Limit(1)

	return db.NewSelect().
		ColumnExpr("(?) AS waiting", waiting).
		ColumnExpr("(?) AS overdue", overdue).
		ColumnExpr("(?) AS closed_by_people_today", closed).
		ColumnExpr("(?) AS handled_by_nova_today", handled).
		ColumnExpr("(?) AS coming_back", comingBack).
		ColumnExpr("EXISTS (?) AS any_message", anyMessage)
}

func (r *repository) HandledByNova(
	ctx context.Context,
	req triageports.BriefingWindow,
) ([]*triageports.HandledRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.HandledRow, error) {
		rows := make([]*triageports.HandledRow, 0, 16)
		if err := handledQuery(r.db.DBForContext(ctx), req).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return rows, nil
	})
}

func handledQuery(db bun.IDB, req triageports.BriefingWindow) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	partition := "PARTITION BY " + cols.HandledAction.Qualified() + ", " +
		cols.Classification.Qualified()

	inner := messages(db).
		Apply(messageRowColumns).
		ColumnExpr("COUNT(*) OVER ("+partition+") AS group_count").
		ColumnExpr("ROW_NUMBER() OVER ("+partition+" ORDER BY "+
			cols.ReceivedAt.OrderDesc()+", "+cols.ID.OrderDesc()+") AS sample_rank").
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.HandledByRuleID.IsNotNull()).
		Where(cols.ReceivedAt.Gte(), req.DayStart)

	return db.NewSelect().
		TableExpr("(?) AS "+rankedAlias, inner).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return rankedRowColumns(sq, rankedAlias, "group_count")
		}).
		Where(rankedAlias+".sample_rank <= ?", handledSamplesPerGroup).
		OrderExpr(rankedAlias + ".group_count DESC, " + rankedAlias + ".handled_action, " +
			rankedAlias + ".classification, " + rankedAlias + ".sample_rank")
}

func (r *repository) Arrivals(
	ctx context.Context,
	req triageports.BriefingWindow,
) ([]*triageports.ArrivalRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.ArrivalRow, error) {
		rows := make([]*triageports.ArrivalRow, 0, 24)
		if err := arrivalsQuery(r.db.DBForContext(ctx), req).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return rows, nil
	})
}

func arrivalsQuery(db bun.IDB, req triageports.BriefingWindow) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return messages(db).
		ColumnExpr(cols.ReceivedAt.Expr("FLOOR(({} - ?) / 3600)::int AS hour"), req.DayStart).
		ColumnExpr("COUNT(*) FILTER (WHERE "+settledByRule()+") AS handled_by_nova", bun.List(settled)).
		ColumnExpr("COUNT(*) FILTER (WHERE NOT "+settledByRule()+") AS left_for_person",
			bun.List(settled)).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ReceivedAt.Gte(), req.DayStart).
		Where(cols.ReceivedAt.Lte(), req.Now).
		GroupExpr("1").
		OrderExpr("1")
}

func (r *repository) River(
	ctx context.Context,
	req triageports.BriefingWindow,
) ([]*triageports.RiverRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triageports.RiverRow, error) {
		rows := make([]*triageports.RiverRow, 0, req.Limit*2)
		if err := riverQuery(r.db.DBForContext(ctx), req).Scan(ctx, &rows); err != nil {
			return nil, err
		}

		return rows, nil
	})
}

func riverQuery(db bun.IDB, req triageports.BriefingWindow) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	inner := messages(db).
		Apply(messageRowColumns).
		ColumnExpr(settledByRule()+" AS nova", bun.List(settled)).
		ColumnExpr("ROW_NUMBER() OVER (PARTITION BY "+settledByRule()+" ORDER BY "+
			cols.ReceivedAt.OrderDesc()+", "+cols.ID.OrderDesc()+") AS lane_rank",
			bun.List(settled)).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.ReceivedAt.Gte(), req.DayStart).
		Where(cols.ReceivedAt.Lte(), req.Now).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.
				Where(settledByRule(), bun.List(settled)).
				WhereOr(cols.Status.In(), bun.List(needsReview)).
				WhereOr(settledByPerson(), bun.List(settled))
		})

	return db.NewSelect().
		TableExpr("(?) AS "+rankedAlias, inner).
		Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
			return rankedRowColumns(sq, rankedAlias, "nova")
		}).
		Where(rankedAlias+".lane_rank <= ?", req.Limit).
		OrderExpr(rankedAlias + ".received_at ASC, " + rankedAlias + ".id ASC")
}

func (r *repository) Waiting(
	ctx context.Context,
	req triageports.WaitingRequest,
) ([]*inboundmessage.InboundMessage, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]*inboundmessage.InboundMessage, error) {
			entities := make([]*inboundmessage.InboundMessage, 0, req.Limit)
			if err := waitingQuery(r.db.DBForContext(ctx), req, &entities).Scan(ctx); err != nil {
				return nil, err
			}

			return entities, nil
		},
	)
}

func waitingQuery(
	db bun.IDB,
	req triageports.WaitingRequest,
	entities *[]*inboundmessage.InboundMessage,
) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return db.NewSelect().
		Model(entities).
		ExcludeColumn(cols.TextBody.Bare(), cols.HTMLKey.Bare()).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.Status.In(), bun.List(needsReview)).
		Where(cols.Parked.IsNull()).
		OrderExpr(cols.DueAt.Qualified()+" ASC NULLS LAST").
		Order(cols.ReceivedAt.OrderAsc(), cols.ID.OrderAsc()).
		Limit(int64(req.Limit))
}

func (r *repository) CorrectionTotals(
	ctx context.Context,
	req triageports.CorrectionTotalsRequest,
) (*triageports.CorrectionTotals, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.CorrectionTotals, error) {
		totals := new(triageports.CorrectionTotals)
		if err := correctionTotalsQuery(r.db.DBForContext(ctx), req).Scan(ctx, totals); err != nil {
			return nil, err
		}

		return totals, nil
	})
}

func correctionTotalsQuery(db bun.IDB, req triageports.CorrectionTotalsRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	act := triagecol.ActivityColumns

	corrections := db.NewSelect().
		Model((*triage.Activity)(nil)).
		ColumnExpr("COUNT(*)").
		Apply(triagecol.ActivityApplyTenant(req.TenantInfo)).
		Where(act.Kind.In(), bun.List([]triage.ActivityKind{
			triage.ActivityCorrected,
			triage.ActivityFactFlagged,
		})).
		Where(act.ActorKind.Eq(), triage.ActorPerson).
		Where(act.CreatedAt.Gte(), req.WeekStart)

	corrected := "(" + cols.ClassificationCorrected.Qualified() + " OR COALESCE(" +
		cols.Facts.Qualified() + " @> '[{\"flagged\": true}]'::jsonb, FALSE))"

	return messages(db).
		ColumnExpr("(?) AS corrections_this_week", corrections).
		ColumnExpr("COUNT(*) FILTER (WHERE "+cols.ReceivedAt.Gte()+") AS read_this_week",
			req.WeekStart).
		ColumnExpr("COUNT(*) FILTER (WHERE "+cols.ReceivedAt.Gte()+" AND "+corrected+
			") AS corrected_this_week", req.WeekStart).
		ColumnExpr("COUNT(*) FILTER (WHERE "+cols.ReceivedAt.Lt()+") AS read_last_week",
			req.WeekStart).
		ColumnExpr("COUNT(*) FILTER (WHERE "+cols.ReceivedAt.Lt()+" AND "+corrected+
			") AS corrected_last_week", req.WeekStart).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
		Where(cols.NovaRead.IsTrue()).
		Where(cols.ReceivedAt.Gte(), req.LastWeekStart)
}
