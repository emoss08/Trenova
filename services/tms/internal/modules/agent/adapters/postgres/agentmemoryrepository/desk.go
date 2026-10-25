package agentmemoryrepository

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/domain/conversation/conversationdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission/permissiondb"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	defaultDeskPage = 50
	maxDeskPage     = 100
	maxDeskIDs      = 200
)

type deskCount struct {
	Scope  agent.MemoryScope
	RoleID pulid.ID
	Count  int
}

func deskRow(memory *agent.Memory) *agentports.DeskMemoryRow {
	return &agentports.DeskMemoryRow{
		Memory:      memory,
		SourceTitle: memory.SourceTitle,
		RoleName:    memory.RoleName,
	}
}

func deskRows(memories []*agent.Memory) []*agentports.DeskMemoryRow {
	out := make([]*agentports.DeskMemoryRow, 0, len(memories))
	for _, memory := range memories {
		out = append(out, deskRow(memory))
	}

	return out
}

// deskSelect reads memories with their conversation's title and their role's
// name. Both are joined within the tenant, so a title is only ever read from
// the memory's own organization.
func deskSelect() *keel.Query[agent.Memory] {
	m := agentdb.Memories
	thread := conversationdb.Threads.As("desk_thread")
	role := permissiondb.Roles.As("desk_role")

	return m.LeftJoin(thread, thread.ID.EqCol(m.SourceThreadID)).
		LeftJoin(role, role.ID.EqCol(m.RoleID)).
		Compute(m.SourceTitle, thread.Title).
		Compute(m.RoleName, role.Name)
}

// deskVisible keeps what a person keeps: memories of the organization, their
// own, and their roles', active or paused, and the memories they offered for
// their team or the organization while an administrator has yet to decide.
// What is kept for one agent is administered in AI Control, and an agent's
// suggestion waits in its conversation.
func deskVisible(filter *agentports.DeskMemoryFilter) keel.Pred {
	m := agentdb.Memories
	kept := keel.And(
		m.Status.In(agent.MemoryStatusActive, agent.MemoryStatusPaused),
		keel.Or(m.Scope.Eq(agent.MemoryScopeOrganization), forPerson(filter.Reader)),
	)
	var offered keel.Pred
	if filter.Reader.UserID.IsNotNil() {
		offered = keel.And(
			m.Status.Eq(agent.MemoryStatusSuggested),
			m.Source.Eq(agent.MemorySourceUser),
			m.CreatedByUserID.Eq(filter.Reader.UserID),
			m.Scope.In(agent.MemoryScopeRole, agent.MemoryScopeOrganization),
		)
	}
	visible := keel.Or(kept, offered)

	text := newTextQuery(filter.Query, nil)
	switch {
	case text.empty():
		return visible
	case text.unmatchable():
		return keel.Raw("FALSE")
	default:
		return keel.And(visible, text.primary().matches(m.SearchVector))
	}
}

func listDeskQuery(req *agentports.ListDeskMemoriesQuery) *keel.Query[agent.Memory] {
	m := agentdb.Memories
	limit := req.Limit
	if limit <= 0 {
		limit = defaultDeskPage
	}
	limit = min(limit, maxDeskPage)

	var after keel.Pred
	if req.After != nil {
		after = keel.DynRow(m.CreatedAt, m.ID).Lt(keel.DynRow(req.After.CreatedAt, req.After.ID.String()))
	}

	return deskSelect().
		Where(
			deskVisible(&req.Filter),
			keel.When(req.Filter.Scope != "", m.Scope.Eq(req.Filter.Scope)),
			keel.When(
				req.Filter.Scope == agent.MemoryScopeRole && req.Filter.RoleID.IsNotNil(),
				m.RoleID.Eq(req.Filter.RoleID),
			),
			after,
		).
		Order(m.CreatedAt.Desc(), m.ID.Desc()).
		Limit(limit)
}

func (r *repository) ListDesk(
	ctx context.Context,
	req agentports.ListDeskMemoriesQuery,
) ([]*agentports.DeskMemoryRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentports.DeskMemoryRow, error) {
		memories, err := listDeskQuery(&req).List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list desk memories: %w", err)
		}

		return deskRows(memories), nil
	})
}

func (r *repository) CountDesk(
	ctx context.Context,
	filter agentports.DeskMemoryFilter,
) ([]agentports.DeskMemoryCountRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]agentports.DeskMemoryCountRow, error) {
		m := agentdb.Memories
		rows, err := keel.Into[deskCount](ctx, r.db.Keel(ctx), m.
			Group(m.Scope, m.RoleID).
			Pick(m.Scope, m.RoleID, keel.Count()).
			Where(deskVisible(&filter)))
		if err != nil {
			return nil, fmt.Errorf("count desk memories: %w", err)
		}

		counts := make([]agentports.DeskMemoryCountRow, 0, len(rows))
		for _, row := range rows {
			counts = append(counts, agentports.DeskMemoryCountRow{
				Scope:  row.Scope,
				RoleID: row.RoleID,
				Count:  row.Count,
			})
		}

		return counts, nil
	})
}

func (r *repository) GetDesk(
	ctx context.Context,
	req agentports.GetDeskMemoriesRequest,
) ([]*agentports.DeskMemoryRow, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agentports.DeskMemoryRow, error) {
		if len(req.IDs) == 0 {
			return []*agentports.DeskMemoryRow{}, nil
		}
		ids := req.IDs
		if len(ids) > maxDeskIDs {
			ids = ids[:maxDeskIDs]
		}

		m := agentdb.Memories
		memories, err := deskSelect().
			Where(m.ID.In(ids...)).
			Limit(len(ids)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("get desk memories: %w", err)
		}

		return deskRows(memories), nil
	})
}

// Revise rewrites what a memory says and who reads it, under its version.
// Like Update it leaves where the memory came from alone.
func (r *repository) Revise(
	ctx context.Context,
	req agentports.ReviseAgentMemoryRequest,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		owner, role := pulid.Nil, pulid.Nil
		switch req.Scope {
		case agent.MemoryScopeUser:
			owner = req.OwnerUserID
		case agent.MemoryScopeRole:
			role = req.RoleID
		}

		m := agentdb.Memories
		n, err := m.Where(m.ID.Eq(req.ID), m.Version.Eq(req.Version)).Update(
			ctx,
			r.db.Keel(ctx),
			m.Content.To(req.Content),
			m.Scope.To(req.Scope),
			setID(m.OwnerUserID, owner),
			setID(m.RoleID, role),
		)
		if err != nil {
			return nil, fmt.Errorf("revise agent memory: %w", err)
		}
		if n == 0 {
			return nil, dberror.CreateVersionMismatchError(memoryEntity, req.ID.String())
		}

		return r.GetByID(ctx, agentports.GetAgentMemoryByIDRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
		})
	})
}

func (r *repository) GetPreference(
	ctx context.Context,
	req agentports.GetAgentMemoryPreferenceRequest,
) (*agent.MemoryPreference, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.MemoryPreference, error) {
		p := agentdb.MemoryPreferences
		entity, err := p.Where(p.UserID.Eq(req.UserID)).One(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
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
		p := agentdb.MemoryPreferences
		inserted, err := p.Upsert(
			ctx,
			r.db.Keel(ctx),
			[]*agent.MemoryPreference{entity},
			keel.OnConflict(p.OrganizationID, p.BusinessUnitID, p.UserID).Update(p.SavingMode),
		)
		if err != nil {
			return nil, fmt.Errorf("save agent memory preference: %w", err)
		}
		if len(inserted) > 0 && inserted[0] {
			return entity, nil
		}

		saved, err := p.Where(p.UserID.Eq(entity.UserID)).One(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("read the saved agent memory preference: %w", err)
		}

		return saved, nil
	})
}
