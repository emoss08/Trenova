package agentreflectionrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/timeutils"
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

func New(p Params) repositories.AgentReflectionRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentreflection-repository"),
	}
}

func (r *repository) Claim(
	ctx context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Reflection, error) {
		err := dbtx.Savepoint(ctx, r.db, func(ctx context.Context) error {
			_, insertErr := r.db.DBForContext(ctx).
				NewInsert().
				Model(entity).
				Returning("*").
				Exec(ctx)

			return insertErr
		})
		if err == nil {
			return entity, nil
		}
		if !dberror.IsUniqueConstraintViolation(err) {
			r.l.Error("failed to claim an agent reflection", zap.Error(err))

			return nil, fmt.Errorf("claim agent reflection: %w", err)
		}

		return r.claimed(ctx, entity)
	})
}

func (r *repository) claimed(
	ctx context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	cols := buncolgen.ReflectionColumns
	held := new(agent.Reflection)
	tenant := pagination.TenantInfo{OrgID: entity.OrganizationID, BuID: entity.BusinessUnitID}

	err := r.db.DBForContext(ctx).
		NewSelect().
		Model(held).
		WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			sq = buncolgen.ReflectionScopeTenant(sq, tenant).
				Where(cols.SubjectType.Eq(), entity.SubjectType)
			if entity.SubjectType == agent.ReflectionSubjectRun {
				return sq.Where(cols.RunID.Eq(), entity.RunID)
			}

			return sq.Where(cols.ThreadID.Eq(), entity.ThreadID).
				Where(cols.ThroughSequence.Eq(), entity.ThroughSequence)
		}).
		Limit(1).
		Scan(ctx)
	if err != nil {
		return nil, dberror.HandleNotFoundError(err, "AgentReflection")
	}

	return held, nil
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentReflectionRequest,
) (*agent.Reflection, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Reflection, error) {
		cols := buncolgen.ReflectionColumns
		entity := new(agent.Reflection)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ReflectionScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "AgentReflection")
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agent.Reflection,
) (*agent.Reflection, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Reflection, error) {
		cols := buncolgen.ReflectionColumns
		ov := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.ReflectionScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), ov)
			}).
			Set(cols.Status.Set(), entity.Status).
			Set(cols.SkipReason.Set(), nullableSkip(entity.SkipReason)).
			Set(cols.Signals.Set(), entity.Signals).
			Set(cols.Changes.Set(), entity.Changes).
			Set(cols.Notes.Set(), entity.Notes).
			Set(cols.Tainted.Set(), entity.Tainted).
			Set(cols.Model.Set(), entity.Model).
			Set(cols.ProviderID.Set(), entity.ProviderID).
			Set(cols.InputTokens.Set(), entity.InputTokens).
			Set(cols.OutputTokens.Set(), entity.OutputTokens).
			Set(cols.ErrorMessage.Set(), entity.ErrorMessage).
			Set(cols.FinishedAt.Set(), entity.FinishedAt).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.Set(), entity.Version).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update agent reflection: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("update agent reflection rows: %w", err)
		}
		if rows == 0 {
			return nil, dberror.CreateVersionMismatchError("AgentReflection", entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ThreadReflectedThrough(
	ctx context.Context,
	req repositories.ThreadReflectedThroughRequest,
) (int, bool, error) {
	type result struct {
		through int
		found   bool
	}

	got, err := dbtx.Read(ctx, r.db, func(ctx context.Context) (result, error) {
		cols := buncolgen.ReflectionColumns
		latest := new(agent.Reflection)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(latest).
			Column(cols.ThroughSequence.Bare()).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.ReflectionScopeTenant(sq, req.TenantInfo).
					Where(cols.SubjectType.Eq(), agent.ReflectionSubjectThread).
					Where(cols.ThreadID.Eq(), req.ThreadID).
					Where(cols.Status.NotEq(), agent.ReflectionStatusFailed)
			}).
			OrderExpr(cols.ThroughSequence.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return result{}, nil
			}

			return result{}, fmt.Errorf("read how far a conversation was looked back over: %w", err)
		}

		return result{through: latest.ThroughSequence, found: true}, nil
	})
	if err != nil {
		return 0, false, err
	}

	return got.through, got.found, nil
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentReflectionConnectionRequest,
) (*pagination.CursorListResult[*agent.Reflection], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.Reflection], error) {
			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*agent.Reflection)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						sq = querybuilder.ApplyFiltersWithoutSort(
							sq,
							buncolgen.ReflectionTable.Alias,
							req.Filter,
							(*agent.Reflection)(nil),
						)

						return sq.Apply(buncolgen.ReflectionApplyTenant(req.Filter.TenantInfo))
					}).
					Count(ctx)
				if err != nil {
					return nil, fmt.Errorf("count agent reflections: %w", err)
				}
				totalCount = &total
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*agent.Reflection]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*agent.Reflection) *bun.SelectQuery {
						q := dba.NewSelect().Model(entities)
						if len(req.Columns) > 0 {
							q = q.Column(req.Columns...)
						}

						return q
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return querybuilder.ApplyCursorFilters(
							sq,
							buncolgen.ReflectionTable.Alias,
							req.Filter,
							req.Cursor,
							(*agent.Reflection)(nil),
						)
					},
				})
			if err != nil {
				return nil, fmt.Errorf("list agent reflections: %w", err)
			}

			return result, nil
		},
	)
}

func nullableSkip(reason agent.ReflectionSkip) any {
	if reason == "" {
		return nil
	}

	return reason
}
