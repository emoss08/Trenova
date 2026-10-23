package agentrunrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/pkg/buncolgen/agentcol"
	"github.com/emoss08/trenova/pkg/buncolgen/conversationcol"

	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
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

func New(p Params) agentports.AgentRunRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentrun-repository"),
	}
}

func readableBy(db bun.IDB, reader agentports.RunReader) func(*bun.SelectQuery) *bun.SelectQuery {
	return func(q *bun.SelectQuery) *bun.SelectQuery {
		if !reader.ForPerson() {
			return q
		}

		cols := agentcol.AgentRunColumns
		if reader.UserID().IsNil() {
			return q.Where(cols.SubjectType.NotEq(), agent.SubjectAssistantThread)
		}

		threads := conversationcol.ThreadColumns
		owned := db.NewSelect().
			Model((*conversation.Thread)(nil)).
			ColumnExpr("1").
			Where(threads.ID.EqColumn(cols.SubjectID)).
			Where(threads.OrganizationID.EqColumn(cols.OrganizationID)).
			Where(threads.BusinessUnitID.EqColumn(cols.BusinessUnitID)).
			Where(threads.UserID.Eq(), reader.UserID())

		return q.WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
			return sq.Where(cols.SubjectType.NotEq(), agent.SubjectAssistantThread).
				WhereOr("EXISTS (?)", owned)
		})
	}
}

func (r *repository) filterQuery(
	db bun.IDB,
	q *bun.SelectQuery,
	req *agentports.ListAgentRunRequest,
) *bun.SelectQuery {
	cols := agentcol.AgentRunColumns
	q = querybuilder.ApplyFilters(
		q,
		agentcol.AgentRunTable.Alias,
		req.Filter,
		(*agent.AgentRun)(nil),
	)

	return q.Apply(agentcol.AgentRunApplyTenant(req.Filter.TenantInfo)).
		Apply(readableBy(db, req.Reader)).
		Limit(int64(req.Filter.Pagination.SafeLimit())).
		Offset(int64(req.Filter.Pagination.SafeOffset())).
		Order(cols.CreatedAt.OrderDesc())
}

func (r *repository) List(
	ctx context.Context,
	req *agentports.ListAgentRunRequest,
) (*pagination.ListResult[*agent.AgentRun], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.ListResult[*agent.AgentRun], error) {
			log := r.l.With(zap.String("operation", "List"))

			entities := make([]*agent.AgentRun, 0, req.Filter.Pagination.SafeLimit())
			db := r.db.DBForContext(ctx)
			total, err := db.
				NewSelect().
				Model(&entities).
				Apply(withoutTranscript).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					return r.filterQuery(db, sq, req)
				}).ScanAndCount(ctx)
			if err != nil {
				log.Error("failed to scan and count agent runs", zap.Error(err))
				return nil, err
			}

			return &pagination.ListResult[*agent.AgentRun]{Items: entities, Total: int(total)}, nil
		},
	)
}

func (r *repository) applyTotalCountFilters(
	db bun.IDB,
	q *bun.SelectQuery,
	req *agentports.ListAgentRunConnectionRequest,
) *bun.SelectQuery {
	q = querybuilder.ApplyFiltersWithoutSort(
		q,
		agentcol.AgentRunTable.Alias,
		req.Filter,
		(*agent.AgentRun)(nil),
	)

	return q.Apply(agentcol.AgentRunApplyTenant(req.Filter.TenantInfo)).
		Apply(readableBy(db, req.Reader))
}

func (r *repository) applyCursorPageFilters(
	db bun.IDB,
	q *bun.SelectQuery,
	req *agentports.ListAgentRunConnectionRequest,
) (*bun.SelectQuery, error) {
	return querybuilder.ApplyCursorFilters(
		q.Apply(readableBy(db, req.Reader)),
		agentcol.AgentRunTable.Alias,
		req.Filter,
		req.Cursor,
		(*agent.AgentRun)(nil),
	)
}

func applyAgentRunColumns(q *bun.SelectQuery, columns []string) *bun.SelectQuery {
	if len(columns) == 0 {
		return withoutTranscript(q)
	}

	return q.Column(columns...)
}

func withoutTranscript(q *bun.SelectQuery) *bun.SelectQuery {
	return q.ExcludeColumn(agentcol.AgentRunColumns.Transcript.String())
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentRunConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRun], error) {
	return dbtx.Write(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.AgentRun], error) {
			log := r.l.With(zap.String("operation", "ListConnection"))

			dba := r.db.DBForContext(ctx)
			var totalCount *int
			if req.Cursor.IncludeTotalCount {
				total, err := dba.
					NewSelect().
					Model((*agent.AgentRun)(nil)).
					Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
						return r.applyTotalCountFilters(dba, sq, req)
					}).
					Count(ctx)
				if err != nil {
					log.Error("failed to count agent runs", zap.Error(err))
					return nil, err
				}
				totalCount = new(int(total))
			}

			result, err := dbhelper.CursorList(
				ctx,
				dbhelper.CursorListParams[*agent.AgentRun]{
					Filter:     req.Filter,
					Cursor:     req.Cursor,
					TotalCount: totalCount,
					Query: func(entities *[]*agent.AgentRun) *bun.SelectQuery {
						return dba.
							NewSelect().
							Model(entities).
							Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
								return applyAgentRunColumns(sq, req.Columns)
							})
					},
					Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
						return r.applyCursorPageFilters(dba, sq, req)
					},
				})
			if err != nil {
				log.Error("failed to scan agent runs", zap.Error(err))
				return nil, err
			}

			return result, nil
		},
	)
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentRun{}, nil
		}

		cols := agentcol.AgentRunColumns
		runs := make([]*agent.AgentRun, 0, len(req.IDs))
		db := r.db.DBForContext(ctx)
		err := db.
			NewSelect().
			Model(&runs).
			Apply(withoutTranscript).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentcol.AgentRunScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.In(), bun.In(req.IDs))
			}).
			Apply(readableBy(db, req.Reader)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list agent runs by ids", zap.Error(err))

			return nil, fmt.Errorf("list agent runs by ids: %w", err)
		}

		return runs, nil
	})
}

func (r *repository) ListTranscriptsByIDs(
	ctx context.Context,
	req agentports.ListAgentRunsByIDsRequest,
) ([]*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.AgentRun, error) {
		if len(req.IDs) == 0 {
			return []*agent.AgentRun{}, nil
		}

		cols := agentcol.AgentRunColumns
		runs := make([]*agent.AgentRun, 0, len(req.IDs))
		db := r.db.DBForContext(ctx)
		err := db.
			NewSelect().
			Model(&runs).
			Column(
				cols.ID.String(),
				cols.BusinessUnitID.String(),
				cols.OrganizationID.String(),
				cols.Transcript.String(),
			).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentcol.AgentRunScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.In(), bun.List(req.IDs))
			}).
			Apply(readableBy(db, req.Reader)).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list agent run transcripts by ids", zap.Error(err))

			return nil, fmt.Errorf("list agent run transcripts by ids: %w", err)
		}

		return runs, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		log := r.l.With(zap.String("operation", "GetByID"), zap.String("id", req.ID.String()))

		entity := new(agent.AgentRun)
		cols := agentcol.AgentRunColumns
		db := r.db.DBForContext(ctx)
		err := db.
			NewSelect().
			Model(entity).
			Apply(withoutTranscript).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentcol.AgentRunScopeTenant(sq, *req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Apply(readableBy(db, req.Reader)).
			Scan(ctx)
		if err != nil {
			log.Error("failed to get agent run", zap.Error(err))
			return nil, dberror.HandleNotFoundError(err, "AgentRun")
		}

		return entity, nil
	})
}

func (r *repository) Create(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		log := r.l.With(zap.String("operation", "Create"))

		if _, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			Returning("*").
			Exec(ctx); err != nil {
			log.Error("failed to create agent run", zap.Error(err))
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) Update(
	ctx context.Context,
	entity *agent.AgentRun,
) (*agent.AgentRun, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.AgentRun, error) {
		log := r.l.With(zap.String("operation", "Update"), zap.String("id", entity.ID.String()))

		ov := entity.Version
		entity.Version++
		cols := agentcol.AgentRunColumns

		results, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).WherePK().
			Where(cols.Version.Eq(), ov).
			OmitZero().
			Returning("*").
			Exec(ctx)
		if err != nil {
			log.Error("failed to update agent run", zap.Error(err))
			return nil, err
		}

		if err = dberror.CheckRowsAffected(results, "AgentRun", entity.ID.String()); err != nil {
			return nil, err
		}

		return entity, nil
	})
}

func (r *repository) CountOpen(
	ctx context.Context,
	req agentports.CountOpenAgentRunsRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := agentcol.AgentRunColumns

		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agent.AgentRun)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = agentcol.AgentRunScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.DefinitionID).
					Where(cols.Status.NotIn(), bun.In([]agent.RunStatus{
						agent.RunStatusCompleted,
						agent.RunStatusShadowCompleted,
						agent.RunStatusFailed,
					}))
				if req.SubjectID.IsNotNil() {
					sq = sq.Where(cols.SubjectID.Eq(), req.SubjectID)
				}

				return sq
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count open agent runs", zap.Error(err))
			return 0, err
		}

		return int(count), nil
	})
}

func (r *repository) CountSince(
	ctx context.Context,
	req agentports.CountAgentRunsSinceRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		cols := agentcol.AgentRunColumns

		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agent.AgentRun)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return agentcol.AgentRunScopeTenant(sq, req.TenantInfo).
					Where(cols.AgentDefinitionID.Eq(), req.DefinitionID).
					Where(cols.CreatedAt.Gte(), req.Since)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count agent runs since", zap.Error(err))

			return 0, fmt.Errorf("count agent runs since: %w", err)
		}

		return int(count), nil
	})
}
