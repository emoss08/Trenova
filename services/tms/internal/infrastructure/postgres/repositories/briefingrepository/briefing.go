package briefingrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/briefing"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	defaultListLimit = 14
	maxListLimit     = 90
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

func New(p Params) repositories.BriefingRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.briefing-repository"),
	}
}

// Upsert writes the day's briefing, replacing whatever was written for the
// same organization, role, reader and day. The unique index carries the
// reader as COALESCE(user_id, ”), so the conflict target names the same
// expression: a shared briefing and a per-person one are different rows,
// and a rerun of either replaces itself rather than stacking.
func (r *repository) Upsert(
	ctx context.Context,
	entity *briefing.Briefing,
) (*briefing.Briefing, error) {
	if _, err := buildUpsert(r.db.DBForContext(ctx), entity).Exec(ctx); err != nil {
		r.l.Error("failed to upsert briefing",
			zap.String("role", string(entity.RoleKey)),
			zap.String("date", entity.BriefingDate),
			zap.Error(err))

		return nil, fmt.Errorf("upsert briefing: %w", err)
	}

	return entity, nil
}

func buildUpsert(db bun.IDB, entity *briefing.Briefing) *bun.InsertQuery {
	cols := buncolgen.BriefingColumns
	target := "CONFLICT (" + cols.OrganizationID.Name + ", " + cols.BusinessUnitID.Name + ", " +
		cols.RoleKey.Name + ", " + cols.BriefingDate.Name + ", COALESCE(" + cols.UserID.Name +
		", '')) DO UPDATE"

	return db.NewInsert().
		Model(entity).
		On(target).
		Set(cols.RunID.SetExcluded()).
		Set(cols.Status.SetExcluded()).
		Set(cols.Headline.SetExcluded()).
		Set(cols.Sections.SetExcluded()).
		Set(cols.Facts.SetExcluded()).
		Set(cols.Narrated.SetExcluded()).
		Set(cols.ModelIdentifier.SetExcluded()).
		Set(cols.ProviderID.SetExcluded()).
		Set(cols.FailureReason.SetExcluded()).
		Set(cols.Version.IncConflict(1)).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		Returning("*")
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetBriefingByIDRequest,
) (*briefing.Briefing, error) {
	cols := buncolgen.BriefingColumns
	entity := new(briefing.Briefing)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return buncolgen.BriefingScopeTenant(sq, req.TenantInfo).Where(cols.ID.Eq(), req.ID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, errortypes.NewNotFoundError("Briefing not found")
		}

		return nil, fmt.Errorf("get briefing: %w", err)
	}

	return entity, nil
}

// GetForDay reads one role's briefing for a day, or nil when it has not
// been written. A missing briefing is not an error: a reader who opens the
// page before the morning job has run is shown that it is on its way.
func (r *repository) GetForDay(
	ctx context.Context,
	req repositories.GetBriefingForDayRequest,
) (*briefing.Briefing, error) {
	cols := buncolgen.BriefingColumns
	entity := new(briefing.Briefing)

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(entity).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.BriefingScopeTenant(sq, req.TenantInfo).
				Where(cols.RoleKey.Eq(), req.RoleKey).
				Where(cols.BriefingDate.Eq(), req.BriefingDate)

			return scopeReader(sq, req.UserID)
		}).
		Scan(ctx)
	if err != nil {
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, fmt.Errorf("get briefing for day: %w", err)
	}

	return entity, nil
}

func (r *repository) List(
	ctx context.Context,
	req repositories.ListBriefingsRequest,
) ([]*briefing.Briefing, error) {
	cols := buncolgen.BriefingColumns
	limit := req.Limit
	if limit <= 0 {
		limit = defaultListLimit
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	entities := make([]*briefing.Briefing, 0, limit)
	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(&entities).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.BriefingScopeTenant(sq, req.TenantInfo)
			if req.RoleKey != "" {
				sq = sq.Where(cols.RoleKey.Eq(), req.RoleKey)
			}

			return scopeReader(sq, req.UserID)
		}).
		Order(cols.BriefingDate.OrderDesc()).
		Limit(limit).
		Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("list briefings: %w", err)
	}

	return entities, nil
}

func (r *repository) MarkRead(
	ctx context.Context,
	req repositories.GetBriefingByIDRequest,
	readAt int64,
) (*briefing.Briefing, error) {
	return r.stamp(ctx, req, buncolgen.BriefingColumns.ReadAt, readAt)
}

func (r *repository) MarkEmailed(
	ctx context.Context,
	req repositories.GetBriefingByIDRequest,
	emailedAt int64,
) (*briefing.Briefing, error) {
	return r.stamp(ctx, req, buncolgen.BriefingColumns.EmailedAt, emailedAt)
}

// DeleteBefore removes briefings for days before a cut-off. The day is a
// string, so one comparison serves every timezone without converting
// anything.
func (r *repository) DeleteBefore(
	ctx context.Context,
	req repositories.DeleteBriefingsBeforeRequest,
) (int, error) {
	cols := buncolgen.BriefingColumns
	limit := req.Limit
	if limit <= 0 || limit > maxDeleteBatch {
		limit = maxDeleteBatch
	}

	res, err := r.db.DBForContext(ctx).
		NewDelete().
		Model((*briefing.Briefing)(nil)).
		Where(
			cols.ID.In()+" (SELECT "+cols.ID.Qualified()+" FROM "+
				buncolgen.BriefingTable.Name+" AS "+buncolgen.BriefingTable.Alias+
				" WHERE "+cols.BriefingDate.Qualified()+" < ? ORDER BY "+
				cols.BriefingDate.Qualified()+" LIMIT ?)",
			req.BeforeDate,
			limit,
		).
		Exec(ctx)
	if err != nil {
		return 0, fmt.Errorf("delete briefings: %w", err)
	}
	affected, _ := res.RowsAffected()

	return int(affected), nil
}

func (r *repository) stamp(
	ctx context.Context,
	req repositories.GetBriefingByIDRequest,
	column buncolgen.Column,
	at int64,
) (*briefing.Briefing, error) {
	cols := buncolgen.BriefingColumns
	entity := new(briefing.Briefing)

	res, err := r.db.DBForContext(ctx).
		NewUpdate().
		Model(entity).
		Set(column.Set(), at).
		Set(cols.Version.SetExpr("{} + 1")).
		Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
		WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
			return buncolgen.BriefingScopeTenantUpdate(uq, req.TenantInfo).
				Where(cols.ID.Eq(), req.ID)
		}).
		Returning("*").
		Exec(ctx)
	if err != nil {
		return nil, fmt.Errorf("stamp briefing: %w", err)
	}
	if err = dberror.CheckRowsAffected(res, "Briefing", req.ID.String()); err != nil {
		return nil, err
	}

	return entity, nil
}

// scopeReader narrows to the shared briefing or to one person's. A nil
// reader must match IS NULL rather than an empty string, so the two never
// read each other's row.
func scopeReader(q *bun.SelectQuery, userID *pulid.ID) *bun.SelectQuery {
	cols := buncolgen.BriefingColumns
	if userID == nil || userID.IsNil() {
		return q.Where(cols.UserID.IsNull())
	}

	return q.Where(cols.UserID.Eq(), *userID)
}
