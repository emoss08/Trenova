package watchtowerrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
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

func New(p Params) repositories.WatchtowerRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.watchtower-repository"),
	}
}

// Upsert writes the item for its source. A row that is already there keeps
// its id and its resolved state is cleared, since the source is reporting
// the record open again; what the source says now replaces what it said.
func (r *repository) Upsert(
	ctx context.Context,
	item *watchtower.Item,
) (*watchtower.Item, bool, error) {
	cols := buncolgen.ItemColumns
	target := "CONFLICT (" + cols.OrganizationID.Name + ", " + cols.BusinessUnitID.Name + ", " +
		cols.SourceKind.Name + ", " + cols.SourceID.Name + ") DO UPDATE"

	if _, err := r.db.DBForContext(ctx).
		NewInsert().
		Model(item).
		On(target).
		Set(cols.Severity.SetExcluded()).
		Set(cols.Title.SetExcluded()).
		Set(cols.Summary.SetExcluded()).
		Set(cols.SubjectType.SetExcluded()).
		Set(cols.SubjectID.SetExcluded()).
		Set(cols.EventKind.SetExcluded()).
		Set(cols.Path.SetExcluded()).
		Set(cols.OccurredAt.SetExcluded()).
		Set(cols.ResolvedAt.Set(), nil).
		Set(cols.Version.SetExpr("{} + 1")).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*, (xmax = 0) AS inserted").
		Exec(ctx); err != nil {
		r.l.Error("failed to upsert watchtower item",
			zap.String("kind", string(item.SourceKind)),
			zap.String("source", item.SourceID),
			zap.Error(err))

		return nil, false, fmt.Errorf("upsert watchtower item: %w", err)
	}

	return item, item.Inserted, nil
}

func (r *repository) Resolve(
	ctx context.Context,
	req repositories.ResolveWatchtowerItemRequest,
) (*watchtower.Item, error) {
	cols := buncolgen.ItemColumns
	item := new(watchtower.Item)

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(item).
		Set(cols.ResolvedAt.Set(), req.ResolvedAt).
		Set(cols.Version.SetExpr("{} + 1")).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ItemScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.SourceKind.Eq(), req.SourceKind).
				Where(cols.SourceID.Eq(), req.SourceID).
				Where(cols.ResolvedAt.IsNull())
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve watchtower item: %w", err)
	}
	if affected, _ := res.RowsAffected(); affected == 0 {
		return nil, nil
	}

	return item, nil
}

func (r *repository) ResolveByID(
	ctx context.Context,
	req repositories.GetWatchtowerItemRequest,
	resolvedAt int64,
) (*watchtower.Item, error) {
	cols := buncolgen.ItemColumns
	item := new(watchtower.Item)

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(item).
		Set(cols.ResolvedAt.Set(), resolvedAt).
		Set(cols.Version.SetExpr("{} + 1")).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.ItemScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("resolve watchtower item: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "Watchtower item", req.ID.String()); err != nil {
		return nil, err
	}

	return item, nil
}

// ResolveMissing closes the open items of a kind whose source is not among
// the ids still open. An empty list closes every open item of the kind,
// which is what a source that reports nothing open means.
func (r *repository) ResolveMissing(
	ctx context.Context,
	req repositories.ResolveMissingWatchtowerItemsRequest,
) (int, error) {
	cols := buncolgen.ItemColumns

	query := r.db.DBForContext(ctx).
		NewUpdate().
		Model((*watchtower.Item)(nil)).
		Set(cols.ResolvedAt.Set(), req.ResolvedAt).
		Set(cols.Version.SetExpr("{} + 1")).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			uq = buncolgen.ItemScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.SourceKind.Eq(), req.SourceKind).
				Where(cols.ResolvedAt.IsNull())
			if len(req.OpenSourceIDs) > 0 {
				uq = uq.Where(cols.SourceID.NotIn(), bun.In(req.OpenSourceIDs))
			}

			return uq
		})

	res, err := query.Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("resolve missing watchtower items: %w", err)
	}
	affected, _ := res.RowsAffected()

	return int(affected), nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetWatchtowerItemRequest,
) (*watchtower.Item, error) {
	cols := buncolgen.ItemColumns
	item := new(watchtower.Item)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(item).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.ItemScopeTenant(sq, req.TenantInfo).Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError("Watchtower item not found")
		}

		return nil, fmt.Errorf("get watchtower item: %w", err)
	}

	return item, nil
}

func (r *repository) List(
	ctx context.Context,
	req repositories.ListWatchtowerItemsRequest,
) ([]*watchtower.Item, error) {
	cols := buncolgen.ItemColumns

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
			sq = buncolgen.ItemScopeTenant(sq, req.TenantInfo)
			if len(req.Kinds) > 0 {
				sq = sq.Where(cols.SourceKind.In(), bun.In(req.Kinds))
			}
			if len(req.Severities) > 0 {
				sq = sq.Where(cols.Severity.In(), bun.In(req.Severities))
			}
			if req.UnresolvedOnly {
				sq = sq.Where(cols.ResolvedAt.IsNull())
			}
			if req.Since > 0 {
				sq = sq.Where(cols.OccurredAt.Gt(), req.Since)
			}
			if req.BeforeOccurredAt > 0 && req.BeforeID.IsNotNil() {
				sq = sq.Where(
					"("+cols.OccurredAt.Qualified()+", "+cols.ID.Qualified()+") < (?, ?)",
					req.BeforeOccurredAt, req.BeforeID,
				)
			}

			return sq
		}).
		Order(cols.OccurredAt.OrderDesc(), cols.ID.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		r.l.Error("failed to list watchtower items", zap.Error(err))

		return nil, fmt.Errorf("list watchtower items: %w", err)
	}

	return items, nil
}

func (r *repository) Counts(
	ctx context.Context,
	req repositories.CountWatchtowerItemsRequest,
) (*repositories.WatchtowerCounts, error) {
	cols := buncolgen.ItemColumns
	db := r.db.DBForContext(ctx)

	open := func() *bun.SelectQuery {
		return db.NewSelect().
			Model((*watchtower.Item)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = buncolgen.ItemScopeTenant(sq, req.TenantInfo).Where(cols.ResolvedAt.IsNull())
				if len(req.Kinds) > 0 {
					sq = sq.Where(cols.SourceKind.In(), bun.In(req.Kinds))
				}

				return sq
			})
	}

	var totals struct {
		Unresolved     int `bun:"unresolved"`
		Critical       int `bun:"critical"`
		UnseenCritical int `bun:"unseen_critical"`
		Unseen         int `bun:"unseen"`
	}
	if err := open().
		ColumnExpr("COUNT(*) AS unresolved").
		ColumnExpr("COUNT(*) FILTER (WHERE ? = ?) AS critical", bun.Ident(cols.Severity.Qualified()), watchtower.SeverityCritical).
		ColumnExpr("COUNT(*) FILTER (WHERE ? = ? AND ? > ?) AS unseen_critical",
			bun.Ident(cols.Severity.Qualified()), watchtower.SeverityCritical,
			bun.Ident(cols.OccurredAt.Qualified()), req.SeenAt).
		ColumnExpr("COUNT(*) FILTER (WHERE ? > ?) AS unseen", bun.Ident(cols.OccurredAt.Qualified()), req.SeenAt).
		Scan(ctx, &totals); err != nil {
		return nil, fmt.Errorf("count watchtower items: %w", err)
	}

	byKind := make([]repositories.WatchtowerKindCount, 0, len(watchtower.AllSourceKinds()))
	if err := open().
		ColumnExpr("? AS source_kind", bun.Ident(cols.SourceKind.Qualified())).
		ColumnExpr("COUNT(*) AS count").
		GroupExpr("?", bun.Ident(cols.SourceKind.Qualified())).
		Scan(ctx, &byKind); err != nil {
		return nil, fmt.Errorf("count watchtower items by kind: %w", err)
	}

	return &repositories.WatchtowerCounts{
		Unresolved:     totals.Unresolved,
		Critical:       totals.Critical,
		UnseenCritical: totals.UnseenCritical,
		Unseen:         totals.Unseen,
		ByKind:         byKind,
	}, nil
}

func (r *repository) GetCursor(
	ctx context.Context,
	req repositories.GetWatchtowerCursorRequest,
) (*watchtower.Cursor, error) {
	cols := buncolgen.CursorColumns
	cursor := new(watchtower.Cursor)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(cursor).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.CursorScopeTenant(sq, req.TenantInfo).Where(cols.UserID.Eq(), req.UserID)
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
}

// SetCursor moves a reader's cursor forward. It never moves back: two tabs
// marking seen at different moments both leave the later one.
func (r *repository) SetCursor(
	ctx context.Context,
	cursor *watchtower.Cursor,
) (*watchtower.Cursor, error) {
	cols := buncolgen.CursorColumns
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
}

func (r *repository) DeleteResolvedBefore(
	ctx context.Context,
	req repositories.DeleteResolvedWatchtowerItemsRequest,
) (int, error) {
	cols := buncolgen.ItemColumns
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
		Limit(limit)

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
}
