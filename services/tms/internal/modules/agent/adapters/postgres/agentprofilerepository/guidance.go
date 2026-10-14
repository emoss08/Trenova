package agentprofilerepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentprofile"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentprofileports"
	"github.com/emoss08/trenova/pkg/buncolgen/agentprofilecol"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/zap"
)

const (
	defaultGuidanceList = 100
	maxGuidanceList     = 200
)

type guidanceRepository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func NewGuidance(p Params) agentprofileports.GuidanceRepository {
	return &guidanceRepository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agent-profile-guidance-repository"),
	}
}

func scopeGuidanceSubject(
	sq *bun.SelectQuery,
	ref agentprofileports.SubjectRef,
) *bun.SelectQuery {
	cols := agentprofilecol.GuidanceColumns
	sq = agentprofilecol.GuidanceScopeTenant(sq, ref.Tenant()).
		Where(cols.Subject.Eq(), ref.Subject)
	if ref.Subject == agentprofile.SubjectUser {
		return sq.Where(cols.UserID.Eq(), ref.PersonID)
	}

	return sq.Where(cols.UserID.IsNull())
}

// ListActive is what the next version of the profile is written from, oldest
// first, so a later line reads as refining an earlier one.
func (r *guidanceRepository) ListActive(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) ([]*agentprofile.Guidance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentprofile.Guidance, error) {
		cols := agentprofilecol.GuidanceColumns
		rows := make([]*agentprofile.Guidance, 0, agentprofile.MaxActiveGuidance)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeGuidanceSubject(sq, ref).
					Where(cols.Status.Eq(), agentprofile.GuidanceStatusActive)
			}).
			OrderExpr(cols.CreatedAt.OrderAsc()).
			OrderExpr(cols.ID.OrderAsc()).
			Limit(agentprofile.MaxActiveGuidance).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list active agent profile guidance: %w", err)
		}

		return rows, nil
	})
}

// List is what AI Control and the Desk show: the active and the offered lines,
// and the closed ones when asked, newest first, with who gave each.
func (r *guidanceRepository) List(
	ctx context.Context,
	req *agentprofileports.ListGuidanceRequest,
) ([]*agentprofile.Guidance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentprofile.Guidance, error) {
		cols := agentprofilecol.GuidanceColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultGuidanceList
		}
		limit = min(limit, maxGuidanceList)

		rows := make([]*agentprofile.Guidance, 0, min(limit, agentprofile.MaxActiveGuidance))
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			Relation(agentprofilecol.GuidanceRelations.CreatedBy).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = scopeGuidanceSubject(sq, req.Ref)
				if !req.IncludeRetired {
					sq = sq.Where(cols.Status.In(), bun.List(openGuidanceStatuses()))
				}

				return sq
			}).
			OrderExpr(cols.Status.OrderAsc()).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderDesc()).
			Limit(int64(limit)).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list agent profile guidance: %w", err)
		}

		return rows, nil
	})
}

func (r *guidanceRepository) CountActive(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (int, error) {
	return r.count(ctx, ref, []agentprofile.GuidanceStatus{agentprofile.GuidanceStatusActive})
}

func (r *guidanceRepository) CountOpen(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
) (int, error) {
	return r.count(ctx, ref, openGuidanceStatuses())
}

func (r *guidanceRepository) count(
	ctx context.Context,
	ref agentprofileports.SubjectRef,
	statuses []agentprofile.GuidanceStatus,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := agentprofilecol.GuidanceColumns

		return dbhelper.Count(ctx, r.db.DBForContext(ctx).
			NewSelect().
			Model((*agentprofile.Guidance)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return scopeGuidanceSubject(sq, ref).
					Where(cols.Status.In(), bun.List(statuses))
			}))
	})
}

// openGuidanceStatuses are the lines a profile still holds: those shaping it
// and those offered and waiting.
func openGuidanceStatuses() []agentprofile.GuidanceStatus {
	return []agentprofile.GuidanceStatus{
		agentprofile.GuidanceStatusActive,
		agentprofile.GuidanceStatusSuggested,
	}
}

func (r *guidanceRepository) GetByID(
	ctx context.Context,
	req agentprofileports.GetGuidanceRequest,
) (*agentprofile.Guidance, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agentprofile.Guidance, error) {
		cols := agentprofilecol.GuidanceColumns
		entity := new(agentprofile.Guidance)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentprofilecol.GuidanceScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, errortypes.NewNotFoundError("Guidance not found")
			}

			return nil, fmt.Errorf("get agent profile guidance: %w", err)
		}

		return entity, nil
	})
}

func (r *guidanceRepository) Create(
	ctx context.Context,
	entity *agentprofile.Guidance,
) (*agentprofile.Guidance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentprofile.Guidance, error) {
		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			r.l.Error("failed to create agent profile guidance", zap.Error(err))

			return nil, fmt.Errorf("create agent profile guidance: %w", err)
		}

		return entity, nil
	})
}

// Update saves a change to a line under its version, so two people retiring
// the same line at once leave one record of who did.
func (r *guidanceRepository) Update(
	ctx context.Context,
	entity *agentprofile.Guidance,
) (*agentprofile.Guidance, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agentprofile.Guidance, error) {
		cols := agentprofilecol.GuidanceColumns
		previous := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			Column(
				cols.Status.Bare(),
				cols.RetiredByID.Bare(),
				cols.RetiredAt.Bare(),
				cols.Version.Bare(),
			).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return agentprofilecol.GuidanceScopeTenantUpdate(uq, entityTenant(entity)).
					Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), previous)
			}).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update agent profile guidance: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, "Guidance", entity.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}
