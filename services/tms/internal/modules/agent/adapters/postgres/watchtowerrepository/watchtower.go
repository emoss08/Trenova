package watchtowerrepository

import (
	"context"
	"fmt"
	"strconv"

	"github.com/emoss08/trenova/pkg/buncolgen/watchtowercol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 50
	maxListLimit     = 200
	maxDeleteBatch   = 1000
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

func New(p Params) watchtowerports.WatchtowerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.watchtower-repository"),
	}
}

func (r *repository) Upsert(
	ctx context.Context,
	item *watchtower.Item,
) (*watchtower.Item, bool, error) {
	return dbtx.Write2(ctx, r.db, func(ctx context.Context) (*watchtower.Item, bool, error) {
		db := r.db.DBForContext(ctx)
		existing, found, err := r.lockBySource(ctx, db, item)
		if err != nil {
			return nil, false, err
		}
		if !found {
			existing = nil
		}

		plan := planUpsert(existing, item)
		if _, err = buildUpsert(db, item, plan).Exec(ctx); err != nil {
			r.l.Error("failed to upsert watchtower item",
				zap.String("kind", string(item.SourceKind)),
				zap.String("source", item.SourceID),
				zap.Error(err))

			return nil, false, fmt.Errorf("upsert watchtower item: %w", err)
		}

		item.Reopened = plan.reopen
		switch {
		case item.Inserted:
			err = r.insertEvents(ctx, db, []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventCreated, pulid.Nil).
					WithChange("", item.Severity.String()),
			})
		case plan.reopen:
			err = r.insertEvents(ctx, db, []*watchtower.Event{
				watchtower.NewEvent(item, watchtower.EventSourceReopened, pulid.Nil),
			})
		}
		if err != nil {
			return nil, false, err
		}

		return item, item.Inserted, nil
	})
}

type upsertPlan struct {
	reopen       bool
	keepSeverity bool
}

func planUpsert(existing, incoming *watchtower.Item) upsertPlan {
	if existing == nil {
		return upsertPlan{}
	}
	plan := upsertPlan{}
	if existing.ResolvedAt != nil &&
		(existing.ResolvedByID.IsNil() || existing.SourceClearedAt != nil) {
		plan.reopen = true
	}
	if !plan.reopen && existing.EscalationLevel > 0 &&
		existing.Severity.Rank() > incoming.Severity.Rank() {
		plan.keepSeverity = true
	}
	return plan
}

func (r *repository) lockBySource(
	ctx context.Context,
	db bun.IDB,
	item *watchtower.Item,
) (*watchtower.Item, bool, error) {
	cols := watchtowercol.ItemColumns
	existing := new(watchtower.Item)
	err := db.NewSelect().
		Model(existing).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return watchtowercol.ItemScopeTenant(sq, pagination.TenantInfo{
				OrgID: item.OrganizationID,
				BuID:  item.BusinessUnitID,
			}).
				Where(cols.SourceKind.Eq(), item.SourceKind).
				Where(cols.SourceID.Eq(), item.SourceID)
		}).
		For("UPDATE").
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, false, nil
		}
		return nil, false, fmt.Errorf("lock watchtower item: %w", err)
	}
	return existing, true, nil
}

func buildUpsert(db bun.IDB, item *watchtower.Item, plan upsertPlan) *bun.InsertQuery {
	cols := watchtowercol.ItemColumns
	target := "CONFLICT (" + cols.OrganizationID.Name + ", " + cols.BusinessUnitID.Name + ", " +
		cols.SourceKind.Name + ", " + cols.SourceID.Name + ") DO UPDATE"

	if item.Status == "" {
		item.Status = watchtower.StatusOpen
	}

	q := db.NewInsert().
		Model(item).
		On(target).
		Set(cols.Title.SetExcluded()).
		Set(cols.Summary.SetExcluded()).
		Set(cols.SubjectType.SetExcluded()).
		Set(cols.SubjectID.SetExcluded()).
		Set(cols.EventKind.SetExcluded()).
		Set(cols.Path.SetExcluded()).
		Set(cols.OccurredAt.SetExcluded()).
		Set(cols.Version.IncConflict(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix())
	if !plan.keepSeverity {
		q = q.Set(cols.Severity.SetExcluded())
	}
	if plan.reopen {
		q = q.Set(cols.ResolvedAt.SetNull()).
			Set(cols.Status.Set(), watchtower.StatusOpen).
			Set(cols.ResolutionCode.SetNull()).
			Set(cols.ResolutionNote.SetNull()).
			Set(cols.ResolvedByID.SetNull()).
			Set(cols.SourceClearedAt.SetNull()).
			Set(cols.AcknowledgedAt.SetNull()).
			Set(cols.SnoozedUntil.SetNull()).
			Set(cols.DueAt.SetNull()).
			Set(cols.EscalationLevel.Set(), 0).
			Set(cols.LastEscalatedAt.SetNull())
	}

	return q.Returning("*, (xmax = 0) AS inserted")
}

func (r *repository) Resolve(
	ctx context.Context,
	req watchtowerports.ResolveWatchtowerItemRequest,
) (*watchtower.Item, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*watchtower.Item, error) {
		db := r.db.DBForContext(ctx)
		existing, found, err := r.lockBySource(ctx, db, &watchtower.Item{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			SourceKind:     req.SourceKind,
			SourceID:       req.SourceID,
		})
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, nil
		}

		cols := watchtowercol.ItemColumns
		if existing.ResolvedAt != nil {
			if existing.ResolvedByID.IsNil() || existing.SourceClearedAt != nil {
				return nil, nil
			}
			_, err = db.NewUpdate().
				Model((*watchtower.Item)(nil)).
				Set(cols.SourceClearedAt.Set(), req.ResolvedAt).
				Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
				WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
					return watchtowercol.ItemScopeTenantUpdate(uq, req.TenantInfo).
						Where(cols.ID.Eq(), existing.ID)
				}).
				Exec(ctx)
			if err != nil {
				return nil, fmt.Errorf("mark watchtower source cleared: %w", err)
			}
			return nil, nil
		}

		item := new(watchtower.Item)
		_, err = sourceResolveQuery(db, req.TenantInfo, req.ResolvedAt).
			Model(item).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return uq.Where(cols.ID.Eq(), existing.ID)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve watchtower item: %w", err)
		}
		if err = r.insertEvents(ctx, db, []*watchtower.Event{
			watchtower.NewEvent(item, watchtower.EventSourceResolved, pulid.Nil),
		}); err != nil {
			return nil, err
		}

		return item, nil
	})
}

func sourceResolveQuery(
	db bun.IDB,
	tenant pagination.TenantInfo,
	resolvedAt int64,
) *bun.UpdateQuery {
	cols := watchtowercol.ItemColumns
	return db.NewUpdate().
		Model((*watchtower.Item)(nil)).
		Set(cols.ResolvedAt.Set(), resolvedAt).
		Set(cols.Status.Set(), watchtower.StatusResolved).
		Set(cols.ResolutionCode.Set(), watchtower.ResolutionCleared).
		Set(cols.ResolvedByID.SetNull()).
		Set(cols.SnoozedUntil.SetNull()).
		Set(cols.Version.SetExpr("{} + 1")).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return watchtowercol.ItemScopeTenantUpdate(uq, tenant).
				Where(cols.ResolvedAt.IsNull())
		})
}

type resolvedRow struct {
	ID             pulid.ID `bun:"id"`
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func (r *repository) ResolveMissing(
	ctx context.Context,
	req *watchtowerports.ResolveMissingWatchtowerItemsRequest,
) (int, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		db := r.db.DBForContext(ctx)
		cols := watchtowercol.ItemColumns
		notOpen := func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = uq.Where(cols.SourceKind.Eq(), req.SourceKind)
			if len(req.OpenSourceIDs) > 0 {
				uq = uq.Where(cols.SourceID.NotIn(), bun.List(req.OpenSourceIDs))
			}
			return uq
		}

		rows := make([]resolvedRow, 0)
		_, err := sourceResolveQuery(db, req.TenantInfo, req.ResolvedAt).
			WhereGroup(" AND ", notOpen).
			Returning(cols.ID.Bare()+", "+cols.OrganizationID.Bare()+", "+cols.BusinessUnitID.Bare()).
			Exec(ctx, &rows)
		if err != nil {
			return 0, fmt.Errorf("resolve missing watchtower items: %w", err)
		}

		_, err = db.NewUpdate().
			Model((*watchtower.Item)(nil)).
			Set(cols.SourceClearedAt.Set(), req.ResolvedAt).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return notOpen(watchtowercol.ItemScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ResolvedAt.IsNotNull()).
					Where(cols.ResolvedByID.IsNotNull()).
					Where(cols.SourceClearedAt.IsNull()))
			}).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("mark watchtower sources cleared: %w", err)
		}

		if len(rows) == 0 {
			return 0, nil
		}
		events := make([]*watchtower.Event, 0, len(rows))
		for _, row := range rows {
			events = append(events, watchtower.NewEvent(&watchtower.Item{
				ID:             row.ID,
				OrganizationID: row.OrganizationID,
				BusinessUnitID: row.BusinessUnitID,
			}, watchtower.EventSourceResolved, pulid.Nil))
		}
		if err = r.insertEvents(ctx, db, events); err != nil {
			return 0, err
		}

		return len(rows), nil
	})
}

func (r *repository) UpdateWorkflow(
	ctx context.Context,
	req *watchtowerports.UpdateWatchtowerWorkflowRequest,
) (*watchtower.Item, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*watchtower.Item, error) {
		db := r.db.DBForContext(ctx)
		cols := watchtowercol.ItemColumns
		item := req.Item
		tenant := pagination.TenantInfo{OrgID: item.OrganizationID, BuID: item.BusinessUnitID}

		res, err := db.NewUpdate().
			Model(item).
			Column(
				cols.Status.Bare(),
				cols.AssigneeID.Bare(),
				cols.DueAt.Bare(),
				cols.SnoozedUntil.Bare(),
				cols.AcknowledgedAt.Bare(),
				cols.ResolutionCode.Bare(),
				cols.ResolutionNote.Bare(),
				cols.ResolvedByID.Bare(),
				cols.ResolvedAt.Bare(),
				cols.SourceClearedAt.Bare(),
				cols.Severity.Bare(),
				cols.EscalationLevel.Bare(),
				cols.LastEscalatedAt.Bare(),
			).
			Set(cols.Version.SetExpr("{} + 1")).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return watchtowercol.ItemScopeTenantUpdate(uq, tenant).
					Where(cols.ID.Eq(), item.ID).
					Where(cols.Version.Eq(), req.ExpectedVersion)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update watchtower item workflow: %w", err)
		}
		if affected, _ := res.RowsAffected(); affected == 0 {
			return nil, dberror.CreateVersionMismatchError("Exception", item.ID.String())
		}
		if err = r.insertEvents(ctx, db, req.Events); err != nil {
			return nil, err
		}

		return item, nil
	})
}

func (r *repository) InsertEvents(ctx context.Context, events []*watchtower.Event) error {
	if len(events) == 0 {
		return nil
	}
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		return r.insertEvents(ctx, r.db.DBForContext(ctx), events)
	})
}

func (r *repository) insertEvents(
	ctx context.Context,
	db bun.IDB,
	events []*watchtower.Event,
) error {
	if len(events) == 0 {
		return nil
	}
	if _, err := db.NewInsert().Model(&events).Exec(ctx); err != nil {
		return fmt.Errorf("insert watchtower item events: %w", err)
	}
	return nil
}

func (r *repository) ListEvents(
	ctx context.Context,
	req watchtowerports.ListWatchtowerEventsRequest,
) ([]*watchtower.Event, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtower.Event, error) {
		cols := watchtowercol.EventColumns
		limit := req.Limit
		if limit <= 0 {
			limit = watchtower.DefaultEventPageSize
		}
		limit = min(limit, watchtower.MaxEventPageSize)

		events := make([]*watchtower.Event, 0, limit)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&events).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return watchtowercol.EventScopeTenant(sq, req.TenantInfo).
					Where(cols.ItemID.Eq(), req.ItemID)
			}).
			Order(cols.CreatedAt.OrderAsc(), cols.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list watchtower item events: %w", err)
		}
		return events, nil
	})
}

func (r *repository) ListOverdue(
	ctx context.Context,
	req watchtowerports.ListAttentionWatchtowerItemsRequest,
) ([]*watchtower.Item, error) {
	cols := watchtowercol.ItemColumns
	return r.listAttention(ctx, req, func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.ResolvedAt.IsNull()).
			Where(cols.Status.Ne(), watchtower.StatusSnoozed).
			Where(cols.DueAt.IsNotNull()).
			Where(cols.DueAt.Lt(), req.Now).
			Where(cols.EscalationLevel.Lt(), req.BelowEscalationCap)
	}, cols.DueAt.OrderAsc())
}

func (r *repository) ListSnoozeExpired(
	ctx context.Context,
	req watchtowerports.ListAttentionWatchtowerItemsRequest,
) ([]*watchtower.Item, error) {
	cols := watchtowercol.ItemColumns
	return r.listAttention(ctx, req, func(sq *bun.SelectQuery) *bun.SelectQuery {
		return sq.Where(cols.ResolvedAt.IsNull()).
			Where(cols.Status.Eq(), watchtower.StatusSnoozed).
			Where(cols.SnoozedUntil.Lte(), req.Now)
	}, cols.SnoozedUntil.OrderAsc())
}

func (r *repository) listAttention(
	ctx context.Context,
	req watchtowerports.ListAttentionWatchtowerItemsRequest,
	filter func(*bun.SelectQuery) *bun.SelectQuery,
	order string,
) ([]*watchtower.Item, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtower.Item, error) {
		cols := watchtowercol.ItemColumns
		limit := req.Limit
		if limit <= 0 || limit > maxListLimit {
			limit = maxListLimit
		}
		items := make([]*watchtower.Item, 0, limit)
		err := r.db.DBForContext(ctx).NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return filter(watchtowercol.ItemScopeTenant(sq, req.TenantInfo))
			}).
			Order(order, cols.ID.OrderAsc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list watchtower items needing attention: %w", err)
		}
		return items, nil
	})
}

type tenantRow struct {
	OrganizationID pulid.ID `bun:"organization_id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id"`
}

func (r *repository) ListTenantsNeedingAttention(
	ctx context.Context,
	now int64,
) ([]pagination.TenantInfo, error) {
	ctx = dbscope.WithSystem(ctx, "list organizations with overdue or waking exceptions")
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]pagination.TenantInfo, error) {
		cols := watchtowercol.ItemColumns
		rows := make([]tenantRow, 0)
		err := r.db.DBForContext(ctx).NewSelect().
			Model((*watchtower.Item)(nil)).
			Distinct().
			ColumnExpr(cols.OrganizationID.As("organization_id")).
			ColumnExpr(cols.BusinessUnitID.As("business_unit_id")).
			Where(cols.ResolvedAt.IsNull()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.
					WhereGroup(" OR ", func(dq *bun.SelectQuery) *bun.SelectQuery {
						return dq.Where(cols.Status.Ne(), watchtower.StatusSnoozed).
							Where(cols.DueAt.Lt(), now).
							Where(cols.EscalationLevel.Lt(), dispatchcontrol.MaxExceptionEscalations)
					}).
					WhereGroup(" OR ", func(wq *bun.SelectQuery) *bun.SelectQuery {
						return wq.Where(cols.Status.Eq(), watchtower.StatusSnoozed).
							Where(cols.SnoozedUntil.Lte(), now)
					})
			}).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("list tenants needing exception attention: %w", err)
		}
		tenants := make([]pagination.TenantInfo, 0, len(rows))
		for _, row := range rows {
			tenants = append(tenants, pagination.TenantInfo{
				OrgID: row.OrganizationID,
				BuID:  row.BusinessUnitID,
			})
		}
		return tenants, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req watchtowerports.GetWatchtowerItemRequest,
) (*watchtower.Item, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*watchtower.Item, error) {
		cols := watchtowercol.ItemColumns
		item := new(watchtower.Item)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(item).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return watchtowercol.ItemScopeTenant(sq, req.TenantInfo).Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("Watchtower item not found")
			}

			return nil, fmt.Errorf("get watchtower item: %w", err)
		}

		return item, nil
	})
}

func (r *repository) List(
	ctx context.Context,
	req watchtowerports.ListWatchtowerItemsQuery,
) ([]*watchtower.Item, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*watchtower.Item, error) {
		cols := watchtowercol.ItemColumns

		limit := req.Limit
		if limit <= 0 {
			limit = defaultListLimit
		}
		if limit > maxListLimit {
			limit = maxListLimit
		}

		items := make([]*watchtower.Item, 0, limit)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&items).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = watchtowercol.ItemScopeTenant(sq, req.TenantInfo)
				if len(req.Kinds) > 0 {
					sq = sq.Where(cols.SourceKind.In(), bun.List(req.Kinds))
				}
				if len(req.Severities) > 0 {
					sq = sq.Where(cols.Severity.In(), bun.List(req.Severities))
				}
				if req.UnresolvedOnly {
					sq = sq.Where(cols.ResolvedAt.IsNull())
				}
				if req.Since > 0 {
					sq = sq.Where(cols.OccurredAt.Gt(), req.Since)
				}
				if len(req.Statuses) > 0 {
					sq = sq.Where(cols.Status.In(), bun.List(req.Statuses))
				}
				switch {
				case req.Unassigned && len(req.AssigneeIDs) > 0:
					sq = sq.WhereGroup(" AND ", func(aq *bun.SelectQuery) *bun.SelectQuery {
						return aq.Where(cols.AssigneeID.IsNull()).
							WhereOr(cols.AssigneeID.In(), bun.List(req.AssigneeIDs))
					})
				case req.Unassigned:
					sq = sq.Where(cols.AssigneeID.IsNull())
				case len(req.AssigneeIDs) > 0:
					sq = sq.Where(cols.AssigneeID.In(), bun.List(req.AssigneeIDs))
				}
				if req.OverdueAt > 0 {
					sq = sq.Where(cols.ResolvedAt.IsNull()).
						Where(cols.Status.Ne(), watchtower.StatusSnoozed).
						Where(cols.DueAt.Lt(), req.OverdueAt)
				}
				if req.SubjectID.IsNotNil() {
					sq = sq.Where(cols.SubjectID.Eq(), req.SubjectID)
				}
				if req.BeforeID.IsNotNil() {
					if req.OrderByDue {
						sq = sq.Where(
							"("+dueSortExpr()+", "+cols.ID.Qualified()+") > (?, ?)",
							req.BeforeDueAt, req.BeforeID,
						)
					} else if req.BeforeOccurredAt > 0 {
						sq = sq.Where(
							"("+cols.OccurredAt.Qualified()+", "+cols.ID.Qualified()+") < (?, ?)",
							req.BeforeOccurredAt, req.BeforeID,
						)
					}
				}

				return sq
			}).
			Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
				if req.OrderByDue {
					return sq.OrderExpr(dueSortExpr() + " ASC").Order(cols.ID.OrderAsc())
				}
				return sq.Order(cols.OccurredAt.OrderDesc(), cols.ID.OrderDesc())
			}).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list watchtower items", zap.Error(err))

			return nil, fmt.Errorf("list watchtower items: %w", err)
		}

		return items, nil
	})
}

func dueSortExpr() string {
	return watchtowercol.ItemColumns.DueAt.Expr(
		"COALESCE({}, " + strconv.FormatInt(watchtower.NoDueSortKey, 10) + ")",
	)
}

func (r *repository) Counts(
	ctx context.Context,
	req watchtowerports.CountWatchtowerItemsRequest,
) (*watchtowerports.WatchtowerCountsRow, error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*watchtowerports.WatchtowerCountsRow, error) {
			cols := watchtowercol.ItemColumns
			db := r.db.DBForContext(ctx)

			open := func() *bun.SelectQuery {
				return db.NewSelect().
					Model((*watchtower.Item)(nil)).
					WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
						sq = watchtowercol.ItemScopeTenant(sq, req.TenantInfo).
							Where(cols.ResolvedAt.IsNull())
						if len(req.Kinds) > 0 {
							sq = sq.Where(cols.SourceKind.In(), bun.List(req.Kinds))
						}

						return sq
					})
			}

			var totals struct {
				Unresolved     int `bun:"unresolved"`
				Critical       int `bun:"critical"`
				UnseenCritical int `bun:"unseen_critical"`
				Unseen         int `bun:"unseen"`
				Overdue        int `bun:"overdue"`
				Unassigned     int `bun:"unassigned"`
				Mine           int `bun:"mine"`
				MineOverdue    int `bun:"mine_overdue"`
			}
			if err := open().
				ColumnExpr("COUNT(*) AS unresolved").
				ColumnExpr("COUNT(*) FILTER (WHERE ? = ?) AS critical", bun.Ident(cols.Severity.Qualified()), watchtower.SeverityCritical).
				ColumnExpr("COUNT(*) FILTER (WHERE ? = ? AND ? > ?) AS unseen_critical",
					bun.Ident(cols.Severity.Qualified()), watchtower.SeverityCritical,
					bun.Ident(cols.OccurredAt.Qualified()), req.SeenAt).
				ColumnExpr("COUNT(*) FILTER (WHERE ? > ?) AS unseen", bun.Ident(cols.OccurredAt.Qualified()), req.SeenAt).
				ColumnExpr("COUNT(*) FILTER (WHERE ? <> ? AND ? < ?) AS overdue",
					bun.Ident(cols.Status.Qualified()), watchtower.StatusSnoozed,
					bun.Ident(cols.DueAt.Qualified()), req.Now).
				ColumnExpr("COUNT(*) FILTER (WHERE ? IS NULL) AS unassigned",
					bun.Ident(cols.AssigneeID.Qualified())).
				ColumnExpr("COUNT(*) FILTER (WHERE ? = ?) AS mine",
					bun.Ident(cols.AssigneeID.Qualified()), req.AssigneeID).
				ColumnExpr("COUNT(*) FILTER (WHERE ? = ? AND ? <> ? AND ? < ?) AS mine_overdue",
					bun.Ident(cols.AssigneeID.Qualified()), req.AssigneeID,
					bun.Ident(cols.Status.Qualified()), watchtower.StatusSnoozed,
					bun.Ident(cols.DueAt.Qualified()), req.Now).
				Scan(ctx, &totals); err != nil {
				return nil, fmt.Errorf("count watchtower items: %w", err)
			}

			byKind := make(
				[]watchtowerports.WatchtowerKindCount,
				0,
				len(watchtower.AllSourceKinds()),
			)
			if err := open().
				ColumnExpr("? AS source_kind", bun.Ident(cols.SourceKind.Qualified())).
				ColumnExpr("COUNT(*) AS count").
				GroupExpr("?", bun.Ident(cols.SourceKind.Qualified())).
				Scan(ctx, &byKind); err != nil {
				return nil, fmt.Errorf("count watchtower items by kind: %w", err)
			}

			return &watchtowerports.WatchtowerCountsRow{
				Unresolved:     totals.Unresolved,
				Critical:       totals.Critical,
				UnseenCritical: totals.UnseenCritical,
				Unseen:         totals.Unseen,
				Overdue:        totals.Overdue,
				Unassigned:     totals.Unassigned,
				Mine:           totals.Mine,
				MineOverdue:    totals.MineOverdue,
				ByKind:         byKind,
			}, nil
		},
	)
}

func (r *repository) GetCursor(
	ctx context.Context,
	req watchtowerports.GetWatchtowerCursorRequest,
) (*watchtower.Cursor, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*watchtower.Cursor, error) {
		cols := watchtowercol.CursorColumns
		cursor := new(watchtower.Cursor)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(cursor).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return watchtowercol.CursorScopeTenant(sq, req.TenantInfo).
					Where(cols.UserID.Eq(), req.UserID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return &watchtower.Cursor{
					UserID:         req.UserID,
					OrganizationID: req.TenantInfo.OrgID,
					BusinessUnitID: req.TenantInfo.BuID,
				}, nil
			}

			return nil, fmt.Errorf("get watchtower cursor: %w", err)
		}

		return cursor, nil
	})
}

// SetCursor moves a reader's cursor forward. It never moves back: two tabs
// marking seen at different moments both leave the later one.
func (r *repository) SetCursor(
	ctx context.Context,
	cursor *watchtower.Cursor,
) (*watchtower.Cursor, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*watchtower.Cursor, error) {
		cols := watchtowercol.CursorColumns
		target := "CONFLICT (" + cols.UserID.Name + ", " + cols.BusinessUnitID.Name + ", " +
			cols.OrganizationID.Name + ") DO UPDATE"

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(cursor).
			On(target).
			Set("? = GREATEST(?, EXCLUDED.?)", bun.Ident(cols.SeenAt.Name), bun.Ident(cols.SeenAt.Qualified()), bun.Ident(cols.SeenAt.Name)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Returning("*").
			Exec(ctx); err != nil {
			return nil, fmt.Errorf("set watchtower cursor: %w", err)
		}

		return cursor, nil
	})
}

func (r *repository) DeleteResolvedBefore(
	ctx context.Context,
	req watchtowerports.DeleteResolvedWatchtowerItemsRequest,
) (int, error) {
	ctx = dbscope.WithSystem(
		ctx,
		"delete resolved watchtower items past retention across every organization",
	)
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := watchtowercol.ItemColumns
		limit := req.Limit
		if limit <= 0 || limit > maxDeleteBatch {
			limit = maxDeleteBatch
		}

		subquery := r.db.DBForContext(ctx).
			NewSelect().
			Model((*watchtower.Item)(nil)).
			Column(cols.ID.Bare()).
			Where(cols.ResolvedAt.IsNotNull()).
			Where(cols.ResolvedAt.Lt(), req.Before).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return sq.Where(cols.ResolvedByID.IsNull()).
					WhereOr(cols.SourceClearedAt.IsNotNull())
			}).
			Limit(int64(limit))

		res, err := r.db.DBForContext(ctx).
			NewDelete().
			Model((*watchtower.Item)(nil)).
			Where(cols.ID.Qualified()+" IN (?)", subquery).
			Exec(ctx)
		if err != nil {
			return 0, fmt.Errorf("delete resolved watchtower items: %w", err)
		}
		affected, _ := res.RowsAffected()

		return int(affected), nil
	})
}
