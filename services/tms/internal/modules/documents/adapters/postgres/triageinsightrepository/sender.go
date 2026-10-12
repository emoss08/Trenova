package triageinsightrepository

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/internal/modules/documents/ports/triageports"
	"github.com/emoss08/trenova/internal/modules/parties/domain/carrier"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/buncolgen/carriercol"
	"github.com/emoss08/trenova/pkg/buncolgen/customercol"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/workercol"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/uptrace/bun"
)

func (r *repository) SenderStats(
	ctx context.Context,
	req *triageports.SenderStatsRequest,
) (*triageports.SenderStats, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.SenderStats, error) {
		db := r.db.DBForContext(ctx)

		kinds := make([]triageports.SenderKindRow, 0, 8)
		if err := senderKindsQuery(db, req).Scan(ctx, &kinds); err != nil {
			return nil, err
		}

		stats := &triageports.SenderStats{
			Kinds:  make([]triageports.SenderKindRow, 0, len(kinds)),
			Recent: []*triageports.MessageRow{},
		}
		for _, kind := range kinds {
			stats.Messages += kind.Count
			if kind.Classification != "" {
				stats.Kinds = append(stats.Kinds, kind)
			}
		}
		if stats.Messages == 0 {
			return stats, nil
		}

		summary := new(senderSummary)
		if err := senderSummaryQuery(db, req).Scan(ctx, summary); err != nil {
			return nil, err
		}
		stats.UsualSettleSeconds = summary.UsualSettleSeconds
		stats.CustomerID = summary.CustomerID
		stats.CarrierID = summary.CarrierID

		recent := make([]*triageports.MessageRow, 0, req.RecentLimit)
		if err := senderRecentQuery(db, req).Scan(ctx, &recent); err != nil {
			return nil, err
		}
		stats.Recent = recent

		return stats, nil
	})
}

type senderSummary struct {
	UsualSettleSeconds *float64 `bun:"usual_settle_seconds"`
	CustomerID         pulid.ID `bun:"customer_id"`
	CarrierID          pulid.ID `bun:"carrier_id"`
}

func fromSender(req *triageports.SenderStatsRequest) func(*bun.SelectQuery) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.
			Apply(inboundmessagecol.InboundMessageApplyTenant(req.TenantInfo)).
			Where(cols.SenderDomain.Eq(), req.SenderDomain).
			Where(cols.ReceivedAt.Gte(), req.Since)
	}
}

func senderKindsQuery(db bun.IDB, req *triageports.SenderStatsRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return messages(db).
		ColumnExpr(cols.Classification.Expr("COALESCE({}, '') AS classification")).
		ColumnExpr("COUNT(*) AS count").
		Apply(fromSender(req)).
		GroupExpr("1").
		OrderExpr("2 DESC, 1 ASC")
}

func senderSummaryQuery(db bun.IDB, req *triageports.SenderStatsRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	return messages(db).
		ColumnExpr(
			"PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY "+cols.ReviewedAt.Qualified()+" - "+
				cols.ReceivedAt.Qualified()+") FILTER (WHERE "+settledByPerson()+" AND "+
				cols.ReviewedAt.Qualified()+" >= "+cols.ReceivedAt.Qualified()+
				") AS usual_settle_seconds",
			bun.List(settled),
		).
		ColumnExpr(cols.MatchedCustomerID.Expr("MODE() WITHIN GROUP (ORDER BY {}) AS customer_id")).
		ColumnExpr(cols.MatchedCarrierID.Expr("MODE() WITHIN GROUP (ORDER BY {}) AS carrier_id")).
		Apply(fromSender(req))
}

func senderRecentQuery(db bun.IDB, req *triageports.SenderStatsRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns

	query := messages(db).Apply(messageRowColumns).Apply(fromSender(req))
	if req.ExcludeID.IsNotNil() {
		query = query.Where(cols.ID.NotEq(), req.ExcludeID)
	}

	return query.
		Order(cols.ReceivedAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(int64(req.RecentLimit))
}

func (r *repository) SenderParty(
	ctx context.Context,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triageports.SenderParty, error) {
		db := r.db.DBForContext(ctx)

		if req.CustomerID.IsNotNil() {
			party, found, err := customerParty(ctx, db, req)
			if err != nil || found {
				return party, err
			}
		}
		if req.CarrierID.IsNotNil() {
			party, found, err := carrierParty(ctx, db, req)
			if err != nil || found {
				return party, err
			}
		}

		return driverParty(ctx, db, req)
	})
}

func customerParty(
	ctx context.Context,
	db bun.IDB,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, bool, error) {
	cols := customercol.CustomerColumns
	rows := make([]struct {
		Broker    bool  `bun:"broker"`
		CreatedAt int64 `bun:"created_at"`
	}, 0, 1)

	if err := db.NewSelect().
		Model((*customer.Customer)(nil)).
		ColumnExpr(cols.BrokerVettingEnabled.As("broker")).
		ColumnExpr(cols.CreatedAt.Qualified()).
		Apply(customercol.CustomerApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.CustomerID).
		Limit(1).
		Scan(ctx, &rows); err != nil {
		return nil, false, err
	}
	if len(rows) == 0 {
		return nil, false, nil
	}

	role := inboundmessageports.SenderRoleCustomer
	if rows[0].Broker {
		role = inboundmessageports.SenderRoleBroker
	}

	return &triageports.SenderParty{Role: role, Since: rows[0].CreatedAt}, true, nil
}

func carrierParty(
	ctx context.Context,
	db bun.IDB,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, bool, error) {
	cols := carriercol.CarrierColumns
	since := make([]int64, 0, 1)

	if err := db.NewSelect().
		Model((*carrier.Carrier)(nil)).
		ColumnExpr(cols.CreatedAt.Qualified()).
		Apply(carriercol.CarrierApplyTenant(req.TenantInfo)).
		Where(cols.ID.Eq(), req.CarrierID).
		Limit(1).
		Scan(ctx, &since); err != nil {
		return nil, false, err
	}
	if len(since) == 0 {
		return nil, false, nil
	}

	return &triageports.SenderParty{
		Role:  inboundmessageports.SenderRoleCarrier,
		Since: since[0],
	}, true, nil
}

func driverParty(
	ctx context.Context,
	db bun.IDB,
	req *triageports.SenderPartyRequest,
) (*triageports.SenderParty, error) {
	unknown := &triageports.SenderParty{Role: inboundmessageports.SenderRoleUnknown}
	address := stringutils.NormalizeEmailAddress(req.Address)
	if address == "" && req.SenderDomain == "" {
		return unknown, nil
	}

	cols := workercol.WorkerColumns
	exists, err := db.NewSelect().
		Model((*worker.Worker)(nil)).
		Apply(workercol.WorkerApplyTenant(req.TenantInfo)).
		Where(cols.Status.Eq(), domaintypes.StatusActive).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			if address != "" {
				sq = sq.Where(cols.Email.Expr("LOWER(BTRIM({})) = ?"), address)
			}
			if req.SenderDomain != "" {
				sq = sq.WhereOr(
					cols.Email.Expr("LOWER(BTRIM({})) LIKE ?"),
					"%@"+stringutils.EscapeLikePattern(strings.ToLower(req.SenderDomain)),
				)
			}

			return sq
		}).
		Exists(ctx)
	if err != nil {
		return nil, err
	}
	if !exists {
		return unknown, nil
	}

	return &triageports.SenderParty{Role: inboundmessageports.SenderRoleDriver}, nil
}
