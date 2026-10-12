//nolint:gocritic // Repository request structs follow the existing value-parameter port contracts.
package inboundmessagerepository

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/documents/domain/inboundmessage"
	"github.com/emoss08/trenova/internal/modules/documents/domain/triage"
	"github.com/emoss08/trenova/internal/modules/documents/ports/inboundmessageports"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/buncolgen/inboundmessagecol"
	"github.com/emoss08/trenova/pkg/buncolgen/triagecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

type viewRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewViewRepository(p Params) inboundmessageports.TriageViewRepository {
	return &viewRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.triage-view-repository"),
	}
}

func (r *viewRepository) List(
	ctx context.Context,
	req inboundmessageports.ListViewsRequest,
) ([]*triage.View, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*triage.View, error) {
		views := make([]*triage.View, 0, triage.MaxViewsPerUser)
		cols := triagecol.ViewColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&views).
			Apply(triagecol.ViewApplyTenant(req.TenantInfo)).
			Where(cols.UserID.Eq(), req.UserID).
			Order(cols.Position.OrderAsc(), cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Scan(ctx); err != nil {
			return nil, err
		}

		return views, nil
	})
}

func (r *viewRepository) Get(
	ctx context.Context,
	req inboundmessageports.GetViewRequest,
) (*triage.View, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*triage.View, error) {
		view := new(triage.View)
		cols := triagecol.ViewColumns
		if err := r.db.DBForContext(ctx).
			NewSelect().
			Model(view).
			Apply(triagecol.ViewApplyTenant(req.TenantInfo)).
			Where(cols.ID.Eq(), req.ID).
			Where(cols.UserID.Eq(), req.UserID).
			Scan(ctx); err != nil {
			return nil, dberror.HandleNotFoundError(err, "View")
		}

		return view, nil
	})
}

func (r *viewRepository) Create(
	ctx context.Context,
	view *triage.View,
	limit int,
) (*triage.View, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.View, error) {
		dba := r.db.DBForContext(ctx)
		if _, err := dba.NewRaw(
			"SELECT pg_advisory_xact_lock(hashtextextended(?, 0))",
			"triage-views:"+view.OrganizationID.String()+":"+view.UserID.String(),
		).Exec(ctx); err != nil {
			return nil, err
		}

		cols := triagecol.ViewColumns
		var stats struct {
			Count        int `bun:"count"`
			NextPosition int `bun:"next_position"`
		}
		if err := dba.NewSelect().
			Model((*triage.View)(nil)).
			ColumnExpr(buncolgen.Count("count")).
			ColumnExpr(cols.Position.Expr("COALESCE(MAX({}) + 1, 0) AS next_position")).
			Apply(triagecol.ViewApplyTenant(pagination.TenantInfo{
				OrgID: view.OrganizationID,
				BuID:  view.BusinessUnitID,
			})).
			Where(cols.UserID.Eq(), view.UserID).
			Scan(ctx, &stats); err != nil {
			return nil, err
		}
		if stats.Count >= limit {
			return nil, errortypes.NewBusinessError(
				"You can keep at most {0} saved views. Delete one to save another.", limit,
			)
		}

		view.Position = stats.NextPosition
		if _, err := dba.NewInsert().Model(view).Returning("*").Exec(ctx); err != nil {
			return nil, err
		}

		return view, nil
	})
}

func (r *viewRepository) Rename(ctx context.Context, view *triage.View) (*triage.View, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*triage.View, error) {
		cols := triagecol.ViewColumns
		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(view).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return triagecol.ViewScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: view.OrganizationID,
					BuID:  view.BusinessUnitID,
				}).
					Where(cols.ID.Eq(), view.ID).
					Where(cols.UserID.Eq(), view.UserID)
			}).
			Set(cols.Name.Set(), view.Name).
			Set(cols.UpdatedAt.Set(), view.UpdatedAt).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, err
		}
		if err = dberror.CheckRowsAffected(results, "View", view.ID.String()); err != nil {
			return nil, err
		}

		return view, nil
	})
}

func (r *viewRepository) Delete(ctx context.Context, req inboundmessageports.GetViewRequest) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		cols := triagecol.ViewColumns
		results, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*triage.View)(nil)).
			WhereGroup(" AND ", func(dq *bun.DeleteQuery) *bun.DeleteQuery {
				return triagecol.ViewScopeTenantDelete(dq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Exec(ctx)
		if err != nil {
			return err
		}

		return dberror.CheckRowsAffected(results, "View", req.ID.String())
	})
}

type waitingSearchRequest struct {
	Tenant pagination.TenantInfo
	ViewID pulid.ID
	Query  string
}

func (r *viewRepository) WaitingCounts(
	ctx context.Context,
	req inboundmessageports.ViewWaitingCountsRequest,
) (map[pulid.ID]int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (map[pulid.ID]int, error) {
		counts := make(map[pulid.ID]int, len(req.ViewIDs))
		if len(req.ViewIDs) == 0 {
			return counts, nil
		}

		dba := r.db.DBForContext(ctx)
		views := make([]*triage.View, 0, len(req.ViewIDs))
		cols := triagecol.ViewColumns
		if err := dba.NewSelect().
			Model(&views).
			Column(cols.ID.Bare(), cols.Query.Bare()).
			Apply(triagecol.ViewApplyTenant(req.TenantInfo)).
			Where(cols.ID.In(), bun.List(req.ViewIDs)).
			Scan(ctx); err != nil {
			return nil, err
		}
		if len(views) == 0 {
			return counts, nil
		}

		rows := make([]struct {
			ViewID pulid.ID `bun:"view_id"`
			Count  int      `bun:"count"`
		}, 0, len(views))
		if err := waitingCountsQuery(dba, req.TenantInfo, views).Scan(ctx, &rows); err != nil {
			return nil, err
		}
		for _, row := range rows {
			counts[row.ViewID] = row.Count
		}

		return counts, nil
	})
}

func waitingCountsQuery(
	db bun.IDB,
	tenantInfo pagination.TenantInfo,
	views []*triage.View,
) *bun.SelectQuery {
	query := waitingSearchCount(db, waitingSearchRequest{
		Tenant: tenantInfo,
		ViewID: views[0].ID,
		Query:  views[0].Query,
	})
	for _, view := range views[1:] {
		query = query.UnionAll(waitingSearchCount(db, waitingSearchRequest{
			Tenant: tenantInfo,
			ViewID: view.ID,
			Query:  view.Query,
		}))
	}

	return query
}

func waitingSearchCount(db bun.IDB, req waitingSearchRequest) *bun.SelectQuery {
	cols := inboundmessagecol.InboundMessageColumns
	query := db.NewSelect().
		Model((*inboundmessage.InboundMessage)(nil)).
		ColumnExpr("?::text AS view_id", req.ViewID.String()).
		ColumnExpr(buncolgen.Count("count")).
		Apply(inboundmessagecol.InboundMessageApplyTenant(req.Tenant)).
		Where(cols.Status.In(), bun.List(needsReviewStatuses)).
		Where(cols.Parked.IsNull())

	return searchPredicate(query, inboundmessagecol.InboundMessageTable.Alias, req.Query)
}
