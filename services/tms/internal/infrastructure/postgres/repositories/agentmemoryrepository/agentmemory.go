package agentmemoryrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/dbhelper"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/querybuilder"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	candidatePrealloc         = 64
	maxSuggestionContextLimit = 500
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

func New(p Params) repositories.AgentMemoryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentmemory-repository"),
	}
}

func (r *repository) Create(ctx context.Context, entity *agent.Memory) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		if _, err := r.db.DBForContext(ctx).NewInsert().Model(entity).Returning("*").Exec(ctx); err != nil {
			r.l.Error("failed to create agent memory", zap.Error(err))

			return nil, fmt.Errorf("create agent memory: %w", err)
		}

		return entity, nil
	})
}

// Update rewrites what the memory says and is about, under its version. The
// columns that record where it came from are left alone: an edit changes the
// rule, not its history.
func (r *repository) Update(ctx context.Context, entity *agent.Memory) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		ov := entity.Version
		entity.Version++

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model(entity).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.MemoryScopeTenantUpdate(uq, pagination.TenantInfo{
					OrgID: entity.OrganizationID,
					BuID:  entity.BusinessUnitID,
				}).Where(cols.ID.Eq(), entity.ID).
					Where(cols.Version.Eq(), ov)
			}).
			Set(cols.Kind.Set(), entity.Kind).
			Set(cols.Content.Set(), entity.Content).
			Set(cols.SubjectType.Set(), entity.SubjectType).
			Set(cols.SubjectID.Set(), entity.SubjectID).
			Set(cols.SubjectLabel.Set(), entity.SubjectLabel).
			Set(cols.ToolName.Set(), entity.ToolName).
			Set(cols.ExpiresAt.Set(), entity.ExpiresAt).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.Set(), entity.Version).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("update agent memory: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("update agent memory rows: %w", err)
		}
		if rows == 0 {
			return nil, dberror.CreateVersionMismatchError("AgentMemory", entity.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		entity := new(agent.Memory)
		cols := buncolgen.MemoryColumns
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.MemoryScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID)
			}).
			Scan(ctx)
		if err != nil {
			return nil, dberror.HandleNotFoundError(err, "AgentMemory")
		}

		return entity, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentMemoryConnectionRequest,
) (*pagination.CursorListResult[*agent.Memory], error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*pagination.CursorListResult[*agent.Memory], error) {
		log := r.l.With(zap.String("operation", "ListConnection"))

		dba := r.db.DBForContext(ctx)
		var totalCount *int
		if req.Cursor.IncludeTotalCount {
			total, err := dba.
				NewSelect().
				Model((*agent.Memory)(nil)).
				Apply(func(sq *bun.SelectQuery) *bun.SelectQuery {
					sq = querybuilder.ApplyFiltersWithoutSort(
						sq,
						buncolgen.MemoryTable.Alias,
						req.Filter,
						(*agent.Memory)(nil),
					)

					return sq.Apply(buncolgen.MemoryApplyTenant(req.Filter.TenantInfo))
				}).
				Count(ctx)
			if err != nil {
				log.Error("failed to count agent memories", zap.Error(err))

				return nil, err
			}
			totalCount = &total
		}

		result, err := dbhelper.CursorList(
			ctx,
			dbhelper.CursorListParams[*agent.Memory]{
				Filter:     req.Filter,
				Cursor:     req.Cursor,
				TotalCount: totalCount,
				Query: func(entities *[]*agent.Memory) *bun.SelectQuery {
					q := dba.NewSelect().Model(entities)
					if len(req.Columns) > 0 {
						q = q.Column(req.Columns...)
					}

					return q
				},
				Apply: func(sq *bun.SelectQuery) (*bun.SelectQuery, error) {
					return querybuilder.ApplyCursorFilters(
						sq,
						buncolgen.MemoryTable.Alias,
						req.Filter,
						req.Cursor,
						(*agent.Memory)(nil),
					)
				},
			})
		if err != nil {
			log.Error("failed to list agent memories", zap.Error(err))

			return nil, err
		}

		return result, nil
	})
}

// ListActive reads what a prompt carries. The scope clause is a disjunction
// of what was asked for: the organization-wide rows, the rows about any of
// the subjects, the rows about any of the tools. Asking for none of them
// reads nothing, which is what a caller with no scope means.
func (r *repository) ListActive(
	ctx context.Context,
	req repositories.ListActiveAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		if !req.OrganizationWide && len(req.Subjects) == 0 && len(req.ToolNames) == 0 {
			return []*agent.Memory{}, nil
		}

		cols := buncolgen.MemoryColumns
		limit := boundedCandidateLimit(req.Limit)
		rows := make([]*agent.Memory, 0, min(limit, candidatePrealloc))

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = activeOnly(buncolgen.MemoryScopeTenant(sq, req.TenantInfo), req.Now)
				sq = forAgent(sq, req.AgentDefinitionID, req.Reader)

				return sq.WhereGroup(" AND ", func(scope *bun.SelectQuery) *bun.SelectQuery {
					if req.OrganizationWide {
						scope = scope.WhereGroup(" OR ", func(wide *bun.SelectQuery) *bun.SelectQuery {
							return wide.Where(cols.SubjectType.IsNull()).Where(cols.ToolName.IsNull())
						})
					}
					for _, subject := range req.Subjects {
						scope = scope.WhereGroup(" OR ", func(one *bun.SelectQuery) *bun.SelectQuery {
							return one.Where(cols.SubjectType.Eq(), subject.Type).
								Where(cols.SubjectID.Eq(), subject.ID)
						})
					}
					if len(req.ToolNames) > 0 {
						scope = scope.WhereOr(cols.ToolName.In(), bun.In(req.ToolNames))
					}

					return scope
				})
			}).
			OrderExpr(candidateOrder(), agent.MemoryKindInstruction).
			OrderExpr(kindOrder()).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderAsc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list active agent memories", zap.Error(err))

			return nil, fmt.Errorf("list active agent memories: %w", err)
		}

		return rows, nil
	})
}

// Search is the recall tool's read: the memories whose words match the
// query, best first, and whatever narrowing it was given. A query with no
// operators also matches every word as a prefix, and when that finds nothing
// the words are tried one at a time, so a question finds the memories that
// share most of its words.
func (r *repository) Search(
	ctx context.Context,
	req repositories.SearchAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	text := newTextQuery(req.Query)
	if text.empty() {
		return r.search(ctx, &req, nil)
	}
	if text.unmatchable() {
		return []*agent.Memory{}, nil
	}

	rows, err := r.search(ctx, &req, text.primary())
	if err != nil || len(rows) > 0 {
		return rows, err
	}

	fallback := text.fallback()
	if fallback == nil {
		return rows, nil
	}

	return r.search(ctx, &req, fallback)
}

func (r *repository) search(
	ctx context.Context,
	req *repositories.SearchAgentMemoriesRequest,
	match *textMatch,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		limit := boundedRecallLimit(req.Limit)
		rows := make([]*agent.Memory, 0, limit)

		query := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = forAgent(
					activeOnly(buncolgen.MemoryScopeTenant(sq, req.TenantInfo), req.Now),
					req.AgentDefinitionID,
					req.Reader,
				)
				if match != nil {
					sq = sq.Where(cols.SearchVector.Expr("{} @@ "+match.tsquery), match.args...)
				}
				if len(req.IDs) > 0 {
					sq = sq.Where(cols.ID.In(), bun.List(req.IDs))
				}
				if req.Kind != "" {
					sq = sq.Where(cols.Kind.Eq(), req.Kind)
				}
				if req.Subject != nil {
					sq = sq.Where(cols.SubjectType.Eq(), req.Subject.Type).
						Where(cols.SubjectID.Eq(), req.Subject.ID)
				}
				if tool := strings.TrimSpace(req.ToolName); tool != "" {
					sq = sq.Where(cols.ToolName.Eq(), tool)
				}

				return sq
			})
		if match != nil {
			query = query.OrderExpr(
				cols.SearchVector.Expr("ts_rank_cd({}, "+match.tsquery+") DESC"),
				match.args...,
			)
		}

		err := query.
			OrderExpr(kindOrder()).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderAsc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to search agent memories", zap.Error(err))

			return nil, fmt.Errorf("search agent memories: %w", err)
		}

		return rows, nil
	})
}

// FindActive returns the active memory that already says this in this
// scope, or nil. Content is compared case-insensitively after trimming,
// because the same sentence recorded twice with different spacing is the
// same memory.
func (r *repository) FindActive(
	ctx context.Context,
	req repositories.FindActiveAgentMemoryRequest,
) (*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		entity := new(agent.Memory)

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = activeOnly(buncolgen.MemoryScopeTenant(sq, req.TenantInfo), req.Now).
					Where(cols.Content.Expr("LOWER(TRIM({})) = ?"),
						strings.ToLower(strings.TrimSpace(req.Content)))
				sq = sameReaders(sq, &req)
				if !req.Tainted {
					sq = sq.Where(cols.Tainted.IsFalse())
				}
				if req.Subject != nil {
					sq = sq.Where(cols.SubjectType.Eq(), req.Subject.Type).
						Where(cols.SubjectID.Eq(), req.Subject.ID)
				} else {
					sq = sq.Where(cols.SubjectType.IsNull())
				}
				if tool := strings.TrimSpace(req.ToolName); tool != "" {
					sq = sq.Where(cols.ToolName.Eq(), tool)
				} else {
					sq = sq.Where(cols.ToolName.IsNull())
				}

				return sq
			}).
			OrderExpr(cols.Tainted.OrderAsc()).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			Limit(1).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}

			return nil, fmt.Errorf("find agent memory: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) CountActive(
	ctx context.Context,
	req repositories.CountActiveAgentMemoriesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agent.Memory)(nil)).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return activeOnly(buncolgen.MemoryScopeTenant(sq, req.TenantInfo), req.Now)
			}).
			Count(ctx)
		if err != nil {
			r.l.Error("failed to count active agent memories", zap.Error(err))

			return 0, fmt.Errorf("count active agent memories: %w", err)
		}

		return count, nil
	})
}

func (r *repository) SetStatus(
	ctx context.Context,
	req repositories.SetAgentMemoryStatusRequest,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.Memory)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.MemoryScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Status.NotEq(), req.Status).
					Where(cols.Status.NotIn(), bun.List(suggestionStatuses()))
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.UpdatedAt.Set(), req.At).
			Set(cols.Version.Inc(1))

		if req.Status == agent.MemoryStatusRetired {
			query = query.
				Set(cols.RetiredAt.Set(), req.At).
				Set(cols.RetiredByUserID.Set(), nullableID(req.ByUserID))
		} else {
			query = query.
				Set(cols.RetiredAt.Set(), nil).
				Set(cols.RetiredByUserID.Set(), nil)
		}

		res, err := query.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("set agent memory status: %w", err)
		}
		if err = dberror.CheckRowsAffected(res, "AgentMemory", req.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(
			ctx,
			repositories.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
		)
	})
}

func (r *repository) MarkUsed(
	ctx context.Context,
	req repositories.MarkAgentMemoriesUsedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.IDs) == 0 {
			return nil
		}

		cols := buncolgen.MemoryColumns
		if _, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.Memory)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.MemoryScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.In(), bun.In(req.IDs))
			}).
			Set(cols.UseCount.Inc(1)).
			Set(cols.LastUsedAt.Set(), req.At).
			Exec(ctx); err != nil {
			return fmt.Errorf("mark agent memories used: %w", err)
		}

		return nil
	})
}

func (r *repository) ListSuggestionContext(
	ctx context.Context,
	req repositories.ListAgentMemorySuggestionContextRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		limit := req.Limit
		if limit <= 0 || limit > maxSuggestionContextLimit {
			limit = maxSuggestionContextLimit
		}
		rows := make([]*agent.Memory, 0, min(limit, candidatePrealloc))

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(&rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.MemoryScopeTenant(sq, req.TenantInfo).
					WhereGroup(" AND ", func(scope *bun.SelectQuery) *bun.SelectQuery {
						return scope.
							WhereGroup(" OR ", func(active *bun.SelectQuery) *bun.SelectQuery {
								return activeOnly(active, req.Now).
									Where(cols.Scope.In(), bun.List(agentScopes())).
									WhereGroup(" AND ", func(owner *bun.SelectQuery) *bun.SelectQuery {
										return owner.Where(cols.AgentDefinitionID.IsNull()).
											WhereOr(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
									})
							}).
							WhereGroup(" OR ", func(pending *bun.SelectQuery) *bun.SelectQuery {
								return pending.Where(cols.Status.Eq(), agent.MemoryStatusSuggested).
									Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
							}).
							WhereGroup(" OR ", func(dismissed *bun.SelectQuery) *bun.SelectQuery {
								return dismissed.Where(cols.Status.Eq(), agent.MemoryStatusDismissed).
									Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID).
									Where(cols.RetiredAt.Gte(), req.DismissedSince)
							})
					})
			}).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			r.l.Error("failed to list agent memory suggestion context", zap.Error(err))

			return nil, fmt.Errorf("list agent memory suggestion context: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ResolveSuggestion(
	ctx context.Context,
	req repositories.ResolveAgentMemorySuggestionRequest,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		query := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.Memory)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				from := []agent.MemoryStatus{agent.MemoryStatusSuggested}
				if req.Reconsidered {
					from = append(from, agent.MemoryStatusDismissed)
				}

				return buncolgen.MemoryScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Status.In(), bun.List(from)).
					Where(cols.Version.Eq(), req.Version)
			}).
			Set(cols.Status.Set(), req.Status).
			Set(cols.UpdatedAt.Set(), req.At).
			Set(cols.Version.Inc(1))

		switch req.Status {
		case agent.MemoryStatusActive:
			scope := req.Scope
			if !scope.IsValid() {
				scope = agent.MemoryScopeAgent
			}
			owner, role := pulid.Nil, pulid.Nil
			switch scope {
			case agent.MemoryScopeUser:
				owner = req.OwnerUserID
			case agent.MemoryScopeRole:
				role = req.RoleID
			}
			query = query.
				Set(cols.Content.Set(), req.Content).
				Set(cols.Kind.Set(), req.Kind).
				Set(cols.Scope.Set(), scope).
				Set(cols.OwnerUserID.Set(), nullableID(owner)).
				Set(cols.RoleID.Set(), nullableID(role)).
				Set(cols.CreatedByUserID.Set(), nullableID(req.ByUserID)).
				Set(cols.RetiredAt.Set(), nil).
				Set(cols.RetiredByUserID.Set(), nil)
		case agent.MemoryStatusDismissed:
			query = query.
				Set(cols.RetiredAt.Set(), req.At).
				Set(cols.RetiredByUserID.Set(), nullableID(req.ByUserID))
		default:
			return nil, fmt.Errorf("resolve agent memory suggestion: %q is not a resolution", req.Status)
		}

		res, err := query.Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("resolve agent memory suggestion: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("resolve agent memory suggestion rows: %w", err)
		}
		if rows == 0 {
			return nil, dberror.CreateVersionMismatchError("AgentMemory", req.ID.String())
		}

		return r.GetByID(
			ctx,
			repositories.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
		)
	})
}

// forAgent keeps the memories a prompt for one agent may carry: those kept
// for the whole organization, those kept for this agent alone, and those kept
// for the person the prompt is for, themselves or one of their roles. A
// prompt for no agent in particular carries none of the agent's, and one for
// nobody in particular none of anybody's.
func forAgent(sq *bun.SelectQuery, agentID pulid.ID, reader agent.MemoryReader) *bun.SelectQuery {
	cols := buncolgen.MemoryColumns

	return sq.WhereGroup(" AND ", func(owner *bun.SelectQuery) *bun.SelectQuery {
		owner = owner.Where(cols.Scope.Eq(), agent.MemoryScopeOrganization)
		if agentID.IsNotNil() {
			owner = owner.WhereGroup(" OR ", func(own *bun.SelectQuery) *bun.SelectQuery {
				return own.Where(cols.Scope.Eq(), agent.MemoryScopeAgent).
					Where(cols.AgentDefinitionID.Eq(), agentID)
			})
		}

		return forPerson(owner, reader)
	})
}

// forPerson adds, as alternatives, the memories kept for the reader and for
// their roles.
func forPerson(sq *bun.SelectQuery, reader agent.MemoryReader) *bun.SelectQuery {
	cols := buncolgen.MemoryColumns
	if reader.UserID.IsNotNil() {
		sq = sq.WhereGroup(" OR ", func(own *bun.SelectQuery) *bun.SelectQuery {
			return own.Where(cols.Scope.Eq(), agent.MemoryScopeUser).
				Where(cols.OwnerUserID.Eq(), reader.UserID)
		})
	}
	if len(reader.RoleIDs) > 0 {
		sq = sq.WhereGroup(" OR ", func(role *bun.SelectQuery) *bun.SelectQuery {
			return role.Where(cols.Scope.Eq(), agent.MemoryScopeRole).
				Where(cols.RoleID.In(), bun.List(reader.RoleIDs))
		})
	}

	return sq
}

// sameReaders keeps the rows read by exactly the readers the new memory
// would be: the same agent, the same person or the same role.
func sameReaders(sq *bun.SelectQuery, req *repositories.FindActiveAgentMemoryRequest) *bun.SelectQuery {
	cols := buncolgen.MemoryColumns
	switch req.Scope {
	case agent.MemoryScopeAgent:
		return sq.Where(cols.Scope.Eq(), agent.MemoryScopeAgent).
			Where(cols.AgentDefinitionID.Eq(), req.AgentDefinitionID)
	case agent.MemoryScopeUser:
		return sq.Where(cols.Scope.Eq(), agent.MemoryScopeUser).
			Where(cols.OwnerUserID.Eq(), req.OwnerUserID)
	case agent.MemoryScopeRole:
		return sq.Where(cols.Scope.Eq(), agent.MemoryScopeRole).
			Where(cols.RoleID.Eq(), req.RoleID)
	default:
		return sq.Where(cols.Scope.Eq(), agent.MemoryScopeOrganization)
	}
}

// agentScopes are the scopes about which agents read a memory rather than
// which people. What one person keeps for themselves or their team is theirs,
// and is never handed to the job that drafts suggestions for everyone.
func agentScopes() []agent.MemoryScope {
	return []agent.MemoryScope{agent.MemoryScopeOrganization, agent.MemoryScopeAgent}
}

func suggestionStatuses() []agent.MemoryStatus {
	return []agent.MemoryStatus{agent.MemoryStatusSuggested, agent.MemoryStatusDismissed}
}

func activeOnly(sq *bun.SelectQuery, now int64) *bun.SelectQuery {
	cols := buncolgen.MemoryColumns

	return sq.Where(cols.Status.Eq(), agent.MemoryStatusActive).
		WhereGroup(" AND ", func(expiry *bun.SelectQuery) *bun.SelectQuery {
			return expiry.Where(cols.ExpiresAt.IsNull()).WhereOr(cols.ExpiresAt.Gt(), now)
		})
}

// kindOrder puts instructions before corrections before facts, the order a
// prompt reads them in.
func kindOrder() string {
	return "CASE " + buncolgen.MemoryColumns.Kind.Qualified() +
		" WHEN 'Instruction' THEN 0 WHEN 'Correction' THEN 1 ELSE 2 END ASC"
}

func candidateOrder() string {
	return buncolgen.Expr(
		"CASE WHEN {0} IS NOT NULL THEN 0 WHEN {1} IS NULL AND {2} = ? THEN 1 "+
			"WHEN {1} IS NOT NULL THEN 2 ELSE 3 END ASC",
		buncolgen.MemoryColumns.SubjectType,
		buncolgen.MemoryColumns.ToolName,
		buncolgen.MemoryColumns.Kind,
	)
}

func boundedCandidateLimit(limit int) int {
	if limit <= 0 || limit > agent.MaxMemoryCandidates {
		return agent.MaxMemoryCandidates
	}

	return limit
}

func boundedRecallLimit(limit int) int {
	if limit <= 0 {
		return agent.DefaultMemoryRecallLimit
	}

	return min(limit, agent.MaxMemoryRecallLimit)
}

func nullableID(id pulid.ID) any {
	if id.IsNil() {
		return nil
	}

	return id
}
