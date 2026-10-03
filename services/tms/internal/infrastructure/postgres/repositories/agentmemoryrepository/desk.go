package agentmemoryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/buncolgen"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
)

const (
	defaultDeskPage = 50
	maxDeskPage     = 100
	maxDeskIDs      = 200
)

// deskRow is a memory with what the Desk shows beside it, read in the same
// query: the conversation it was saved from and the role it is kept for.
type deskRow struct {
	agent.Memory `bun:",extend"`

	SourceTitle string `bun:"source_title,scanonly"`
	RoleName    string `bun:"role_name,scanonly"`
}

func (r deskRow) row() *repositories.DeskMemoryRow {
	memory := r.Memory

	return &repositories.DeskMemoryRow{
		Memory:      &memory,
		SourceTitle: r.SourceTitle,
		RoleName:    r.RoleName,
	}
}

// deskSelect reads memories with their conversation's title and their role's
// name. The conversation is joined within the tenant, so a title is only ever
// read from the memory's own organization.
func (r *repository) deskSelect(ctx context.Context, rows *[]deskRow) *bun.SelectQuery {
	cols := buncolgen.MemoryColumns

	return r.db.DBForContext(ctx).
		NewSelect().
		Model(rows).
		ColumnExpr(buncolgen.MemoryTable.Alias + ".*").
		ColumnExpr("athr.title AS source_title").
		ColumnExpr("dmr.name AS role_name").
		Join("LEFT JOIN assistant_threads AS athr ON athr.id = " + cols.SourceThreadID.Qualified() +
			" AND athr.organization_id = " + cols.OrganizationID.Qualified() +
			" AND athr.business_unit_id = " + cols.BusinessUnitID.Qualified()).
		Join("LEFT JOIN roles AS dmr ON dmr.id = " + cols.RoleID.Qualified())
}

// deskVisible keeps what a person keeps: memories of the organization, their
// own, and their roles', active or paused. What is kept for one agent is
// administered in AI Control, and a suggestion waits in its conversation.
func deskVisible(sq *bun.SelectQuery, filter *repositories.DeskMemoryFilter) *bun.SelectQuery {
	cols := buncolgen.MemoryColumns
	sq = buncolgen.MemoryScopeTenant(sq, filter.TenantInfo).
		Where(cols.Status.In(), bun.List([]agent.MemoryStatus{
			agent.MemoryStatusActive,
			agent.MemoryStatusPaused,
		})).
		WhereGroup(" AND ", func(who *bun.SelectQuery) *bun.SelectQuery {
			return forPerson(who.Where(cols.Scope.Eq(), agent.MemoryScopeOrganization), filter.Reader)
		})

	if text := newTextQuery(filter.Query); !text.empty() {
		if text.unmatchable() {
			return sq.Where("FALSE")
		}
		match := text.primary()
		sq = sq.Where(cols.SearchVector.Expr("{} @@ "+match.tsquery), match.args...)
	}

	return sq
}

func (r *repository) ListDesk(
	ctx context.Context,
	req repositories.ListDeskMemoriesRequest,
) ([]*repositories.DeskMemoryRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*repositories.DeskMemoryRow, error) {
		cols := buncolgen.MemoryColumns
		limit := req.Limit
		if limit <= 0 {
			limit = defaultDeskPage
		}
		limit = min(limit, maxDeskPage)

		rows := make([]deskRow, 0, limit)
		err := r.deskSelect(ctx, &rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				sq = deskVisible(sq, &req.Filter)
				if req.Filter.Scope != "" {
					sq = sq.Where(cols.Scope.Eq(), req.Filter.Scope)
				}
				if req.Filter.Scope == agent.MemoryScopeRole && req.Filter.RoleID.IsNotNil() {
					sq = sq.Where(cols.RoleID.Eq(), req.Filter.RoleID)
				}
				if after := req.After; after != nil {
					sq = sq.Where(
						"("+cols.CreatedAt.Qualified()+", "+cols.ID.Qualified()+") < (?, ?)",
						after.CreatedAt, after.ID,
					)
				}

				return sq
			}).
			OrderExpr(cols.CreatedAt.OrderDesc()).
			OrderExpr(cols.ID.OrderDesc()).
			Limit(limit).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list desk memories: %w", err)
		}

		out := make([]*repositories.DeskMemoryRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, row.row())
		}

		return out, nil
	})
}

func (r *repository) CountDesk(
	ctx context.Context,
	filter repositories.DeskMemoryFilter,
) ([]repositories.DeskMemoryCount, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]repositories.DeskMemoryCount, error) {
		cols := buncolgen.MemoryColumns
		var rows []struct {
			Scope  agent.MemoryScope `bun:"scope"`
			RoleID *pulid.ID         `bun:"role_id"`
			Count  int               `bun:"count"`
		}

		err := r.db.DBForContext(ctx).
			NewSelect().
			Model((*agent.Memory)(nil)).
			ColumnExpr(cols.Scope.Qualified()+" AS scope").
			ColumnExpr(cols.RoleID.Qualified()+" AS role_id").
			ColumnExpr("COUNT(*) AS count").
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return deskVisible(sq, &filter)
			}).
			GroupExpr(cols.Scope.Qualified()).
			GroupExpr(cols.RoleID.Qualified()).
			Scan(ctx, &rows)
		if err != nil {
			return nil, fmt.Errorf("count desk memories: %w", err)
		}

		counts := make([]repositories.DeskMemoryCount, 0, len(rows))
		for _, row := range rows {
			count := repositories.DeskMemoryCount{Scope: row.Scope, Count: row.Count}
			if row.RoleID != nil {
				count.RoleID = *row.RoleID
			}
			counts = append(counts, count)
		}

		return counts, nil
	})
}

func (r *repository) GetDesk(
	ctx context.Context,
	req repositories.GetDeskMemoriesRequest,
) ([]*repositories.DeskMemoryRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*repositories.DeskMemoryRow, error) {
		if len(req.IDs) == 0 {
			return []*repositories.DeskMemoryRow{}, nil
		}
		ids := req.IDs
		if len(ids) > maxDeskIDs {
			ids = ids[:maxDeskIDs]
		}

		cols := buncolgen.MemoryColumns
		rows := make([]deskRow, 0, len(ids))
		err := r.deskSelect(ctx, &rows).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.MemoryScopeTenant(sq, req.TenantInfo).
					Where(cols.ID.In(), bun.List(ids))
			}).
			Scan(ctx)
		if err != nil {
			return nil, fmt.Errorf("get desk memories: %w", err)
		}

		out := make([]*repositories.DeskMemoryRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, row.row())
		}

		return out, nil
	})
}

// Revise rewrites what a memory says and who reads it, under its version.
// Like Update it leaves where the memory came from alone.
func (r *repository) Revise(
	ctx context.Context,
	req repositories.ReviseAgentMemoryRequest,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		cols := buncolgen.MemoryColumns
		owner, role := pulid.Nil, pulid.Nil
		switch req.Scope {
		case agent.MemoryScopeUser:
			owner = req.OwnerUserID
		case agent.MemoryScopeRole:
			role = req.RoleID
		}

		res, err := r.db.DBForContext(ctx).
			NewUpdate().
			Model((*agent.Memory)(nil)).
			WhereGroup(" AND ", func(uq *bun.UpdateQuery) *bun.UpdateQuery {
				return buncolgen.MemoryScopeTenantUpdate(uq, req.TenantInfo).
					Where(cols.ID.Eq(), req.ID).
					Where(cols.Version.Eq(), req.Version)
			}).
			Set(cols.Content.Set(), req.Content).
			Set(cols.Scope.Set(), req.Scope).
			Set(cols.OwnerUserID.Set(), nullableID(owner)).
			Set(cols.RoleID.Set(), nullableID(role)).
			Set(cols.UpdatedAt.Set(), timeutils.NowUnix()).
			Set(cols.Version.Inc(1)).
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("revise agent memory: %w", err)
		}

		rows, err := res.RowsAffected()
		if err != nil {
			return nil, fmt.Errorf("revise agent memory rows: %w", err)
		}
		if rows == 0 {
			return nil, dberror.CreateVersionMismatchError("AgentMemory", req.ID.String())
		}

		return r.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
	})
}

func (r *repository) GetPreference(
	ctx context.Context,
	req repositories.GetAgentMemoryPreferenceRequest,
) (*agent.MemoryPreference, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.MemoryPreference, error) {
		entity := new(agent.MemoryPreference)
		err := r.db.DBForContext(ctx).
			NewSelect().
			Model(entity).
			WhereGroup(" AND ", func(sq *bun.SelectQuery) *bun.SelectQuery {
				return buncolgen.MemoryPreferenceScopeTenant(sq, req.TenantInfo).
					Where(buncolgen.MemoryPreferenceColumns.UserID.Eq(), req.UserID)
			}).
			Scan(ctx)
		if err != nil {
			if dberror.IsNotFoundError(err) {
				return nil, nil
			}

			return nil, fmt.Errorf("get agent memory preference: %w", err)
		}

		return entity, nil
	})
}

// SavePreference writes the person's choice, creating their row the first
// time. Two tabs saving at once both land; the later one wins, which is what
// the person last chose.
func (r *repository) SavePreference(
	ctx context.Context,
	entity *agent.MemoryPreference,
) (*agent.MemoryPreference, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.MemoryPreference, error) {
		cols := buncolgen.MemoryPreferenceColumns
		_, err := r.db.DBForContext(ctx).
			NewInsert().
			Model(entity).
			On("CONFLICT (organization_id, business_unit_id, user_id) DO UPDATE").
			Set(cols.SavingMode.SetExcluded()).
			Set(cols.UpdatedAt.SetExcluded()).
			Set(cols.Version.IncConflict(1)).
			Returning("*").
			Exec(ctx)
		if err != nil {
			return nil, fmt.Errorf("save agent memory preference: %w", err)
		}

		return entity, nil
	})
}
