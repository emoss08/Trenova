package agentmemoryrepository

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/infrastructure/postgres"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent/agentdb"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/keelkinds"
	"github.com/emoss08/trenova/pkg/listquery"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	maxSuggestionContextLimit = 500
	memoryEntity              = "AgentMemory"
)

var memoryErrors = dberror.Mapping{Entity: memoryEntity}

type Params struct {
	fx.In

	DB     *postgres.Connection
	Logger *zap.Logger
}

type repository struct {
	db *postgres.Connection
	l  *zap.Logger
}

func New(p Params) agentports.AgentMemoryRepository {
	return &repository{
		db: p.DB,
		l:  p.Logger.Named("postgres.agentmemory-repository"),
	}
}

func (r *repository) Create(ctx context.Context, entity *agent.Memory) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		if err := agentdb.Memories.Create(ctx, r.db.Keel(ctx), entity); err != nil {
			r.l.Error("failed to create agent memory", zap.Error(err))

			return nil, fmt.Errorf("create agent memory: %w", err)
		}

		if entity.Status == agent.MemoryStatusActive && entity.Replaces() {
			if err := r.retireSuperseded(ctx, retireSupersededParams{
				id:       *entity.SupersedesID,
				byUserID: pulid.ConvertFromPtr(entity.CreatedByUserID),
				at:       entity.CreatedAt,
			}); err != nil {
				return nil, err
			}
		}

		return entity, nil
	})
}

type retireSupersededParams struct {
	id       pulid.ID
	byUserID pulid.ID
	at       int64
}

func (r *repository) retireSuperseded(ctx context.Context, p retireSupersededParams) error {
	m := agentdb.Memories
	if _, err := m.Where(m.ID.Eq(p.id), m.Status.In(replaceableStatuses()...)).Update(
		ctx,
		r.db.Keel(ctx),
		m.Status.To(agent.MemoryStatusRetired),
		m.RetiredAt.To(p.at),
		setID(m.RetiredByUserID, p.byUserID),
		m.UpdatedAt.To(p.at),
	); err != nil {
		return fmt.Errorf("retire the agent memory a new one replaces: %w", err)
	}

	return nil
}

// Update rewrites what the memory says and is about, under its version. The
// columns that record where it came from are left alone: an edit changes the
// rule, not its history.
func (r *repository) Update(ctx context.Context, entity *agent.Memory) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		m := agentdb.Memories
		err := m.Save(
			ctx,
			r.db.Keel(ctx),
			entity,
			m.Kind,
			m.Content,
			m.SubjectType,
			m.SubjectID,
			m.SubjectLabel,
			m.ToolName,
			m.ExpiresAt,
		)
		if err != nil {
			if dberror.IsVersionConflict(err) {
				return nil, memoryErrors.Map(err, entity.ID.String())
			}

			return nil, fmt.Errorf("update agent memory: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) GetByID(
	ctx context.Context,
	req agentports.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		entity, err := agentdb.Memories.Get(ctx, r.db.Keel(ctx), req.ID)
		if err != nil {
			return nil, memoryErrors.Map(err, req.ID.String())
		}

		return entity, nil
	})
}

func (r *repository) ListByIDs(
	ctx context.Context,
	req agentports.ListAgentMemoriesByIDsRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		if len(req.IDs) == 0 {
			return []*agent.Memory{}, nil
		}

		m := agentdb.Memories
		entities, err := m.Where(m.ID.In(req.IDs...)).
			Limit(len(req.IDs)).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent memories by ids: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListReplacements(
	ctx context.Context,
	req agentports.ListAgentMemoryReplacementsRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		if len(req.ReplacedIDs) == 0 {
			return []*agent.Memory{}, nil
		}

		m := agentdb.Memories
		entities, err := m.Where(
			m.SupersedesID.In(req.ReplacedIDs...),
			m.Status.In(agent.ReplacingMemoryStatuses()...),
		).
			Order(m.CreatedAt.Desc(), m.ID.Desc()).
			All(ctx, r.db.Keel(ctx))
		if err != nil {
			return nil, fmt.Errorf("list agent memory replacements: %w", err)
		}

		return entities, nil
	})
}

func (r *repository) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentMemoryConnectionRequest,
) (*pagination.CursorListResult[*agent.Memory], error) {
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) (*pagination.CursorListResult[*agent.Memory], error) {
			result, err := listquery.Connection(
				ctx,
				r.db.Keel(ctx),
				agentdb.Memories.Query(),
				&listquery.Request[agent.Memory]{
					Filter:  req.Filter,
					Cursor:  req.Cursor,
					Columns: req.Columns,
				},
			)
			if err != nil {
				r.l.Error("failed to list agent memories", zap.Error(err))

				return nil, err
			}

			return result, nil
		},
	)
}

// ListActive reads what a prompt carries. The scope clause is a disjunction
// of what was asked for: the organization-wide rows, the rows about any of
// the subjects, the rows about any of the tools. Asking for none of them
// reads nothing, which is what a caller with no scope means.
func (r *repository) ListActive(
	ctx context.Context,
	req agentports.ListActiveAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		if !req.OrganizationWide && len(req.Subjects) == 0 && len(req.ToolNames) == 0 {
			return []*agent.Memory{}, nil
		}

		m := agentdb.Memories
		scope := make([]keel.Pred, 0, len(req.Subjects)+2)
		if req.OrganizationWide {
			scope = append(scope, keel.And(m.SubjectType.IsNull(), m.ToolName.IsNull()))
		}
		for _, subject := range req.Subjects {
			scope = append(scope, keel.And(
				m.SubjectType.Eq(subject.Type),
				m.SubjectID.Eq(subject.ID),
			))
		}
		if len(req.ToolNames) > 0 {
			scope = append(scope, m.ToolName.In(req.ToolNames...))
		}

		rows, err := m.Where(
			activeOnly(req.Now),
			forAgent(req.AgentDefinitionID, req.Reader),
			keel.Or(scope...),
		).
			Order(candidateOrder(), kindOrder(), m.CreatedAt.Desc(), m.ID.Asc()).
			Limit(boundedCandidateLimit(req.Limit)).
			List(ctx, r.db.Keel(ctx))
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
// any two of its meaningful words are tried, so a question finds the
// memories that share most of its words.
func (r *repository) Search(
	ctx context.Context,
	req agentports.SearchAgentMemoriesRequest,
) ([]*agent.Memory, error) {
	text := newTextQuery(req.Query, req.Terms)
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

func searchQuery(
	req *agentports.SearchAgentMemoriesRequest,
	match *textMatch,
) *keel.Query[agent.Memory] {
	m := agentdb.Memories
	var subject keel.Pred
	if req.Subject != nil {
		subject = keel.And(m.SubjectType.Eq(req.Subject.Type), m.SubjectID.Eq(req.Subject.ID))
	}
	tool := strings.TrimSpace(req.ToolName)

	q := m.Where(
		activeOnly(req.Now),
		forAgent(req.AgentDefinitionID, req.Reader),
		match.matches(m.SearchVector),
		keel.When(len(req.IDs) > 0, m.ID.In(req.IDs...)),
		keel.When(req.Kind != "", m.Kind.Eq(req.Kind)),
		subject,
		keel.When(tool != "", m.ToolName.Eq(tool)),
	)
	if match != nil {
		q = q.Order(keel.Desc(match.rank(m.SearchVector)))
	}

	return q.Order(kindOrder(), m.CreatedAt.Desc(), m.ID.Asc()).Limit(boundedRecallLimit(req.Limit))
}

func (r *repository) search(
	ctx context.Context,
	req *agentports.SearchAgentMemoriesRequest,
	match *textMatch,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		rows, err := searchQuery(req, match).List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to search agent memories", zap.Error(err))

			return nil, fmt.Errorf("search agent memories: %w", err)
		}

		return rows, nil
	})
}

// FindActive returns the active memory that already says this in this
// scope, or nil. Content is compared by its words alone, ignoring case,
// spacing and punctuation, because "Quote in dollars." and "quote in
// dollars" are the same memory, and an agent restating one should refresh it
// rather than record a second. Both sides go through the same SQL, so the
// database's own idea of a letter decides, never a Go copy of it.
func (r *repository) FindActive(
	ctx context.Context,
	req agentports.FindActiveAgentMemoryRequest,
) (*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		m := agentdb.Memories
		subject := m.SubjectType.IsNull()
		if req.Subject != nil {
			subject = keel.And(m.SubjectType.Eq(req.Subject.Type), m.SubjectID.Eq(req.Subject.ID))
		}
		tool := m.ToolName.IsNull()
		if name := strings.TrimSpace(req.ToolName); name != "" {
			tool = m.ToolName.Eq(name)
		}

		entity, err := m.Where(
			keptOnly(req.Now, req.IncludeSuggested),
			keel.Raw(
				"BTRIM(REGEXP_REPLACE(LOWER(?), '[^[:alnum:]]+', ' ', 'g')) = "+
					"BTRIM(REGEXP_REPLACE(LOWER(?), '[^[:alnum:]]+', ' ', 'g'))",
				m.Content,
				req.Content,
			),
			sameReaders(&req),
			keel.When(!req.Tainted, keel.Or(m.Tainted.IsFalse(), m.ReviewedAt.IsNotNull())),
			subject,
			tool,
		).
			Order(taintsLast(), activeFirst(), m.CreatedAt.Desc()).
			First(ctx, r.db.Keel(ctx))
		if dberror.IsNotFoundError(err) {
			return nil, nil
		}
		if err != nil {
			return nil, fmt.Errorf("find agent memory: %w", err)
		}

		return entity, nil
	})
}

func (r *repository) CountActive(
	ctx context.Context,
	req agentports.CountActiveAgentMemoriesRequest,
) (int, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) (int, error) {
		count, err := agentdb.Memories.Where(activeOnly(req.Now)).Count(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to count active agent memories", zap.Error(err))

			return 0, fmt.Errorf("count active agent memories: %w", err)
		}

		return int(count), nil
	})
}

func (r *repository) SetStatus(
	ctx context.Context,
	req agentports.SetAgentMemoryStatusQuery,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		m := agentdb.Memories
		sets := []keel.Assignment{m.Status.To(req.Status), m.UpdatedAt.To(req.At)}
		if req.Status == agent.MemoryStatusRetired {
			sets = append(sets, m.RetiredAt.To(req.At), setID(m.RetiredByUserID, req.ByUserID))
		} else {
			sets = append(sets, m.RetiredAt.ToNull(), m.RetiredByUserID.ToNull())
		}

		n, err := m.Where(
			m.ID.Eq(req.ID),
			m.Status.Ne(req.Status),
			m.Status.NotIn(suggestionStatuses()...),
		).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("set agent memory status: %w", err)
		}
		if err = dberror.CheckAffected(n, memoryEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(
			ctx,
			agentports.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
		)
	})
}

func (r *repository) Review(
	ctx context.Context,
	req agentports.ReviewAgentMemoryQuery,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		m := agentdb.Memories
		n, err := m.Where(
			m.ID.Eq(req.ID),
			m.Version.Eq(req.Version),
			m.Tainted.IsTrue(),
			m.ReviewedAt.IsNull(),
			m.Status.In(agent.MemoryStatusActive, agent.MemoryStatusPaused),
		).Update(
			ctx,
			r.db.Keel(ctx),
			setID(m.ReviewedByUserID, req.ByUserID),
			m.ReviewedAt.To(req.At),
			m.UpdatedAt.To(req.At),
		)
		if err != nil {
			return nil, fmt.Errorf("review agent memory: %w", err)
		}
		if err = dberror.CheckAffected(n, memoryEntity, req.ID.String()); err != nil {
			return nil, err
		}

		return r.GetByID(
			ctx,
			agentports.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
		)
	})
}

func (r *repository) MarkUsed(
	ctx context.Context,
	req agentports.MarkAgentMemoriesUsedRequest,
) error {
	return dbtx.WriteErr(ctx, r.db, func(ctx context.Context) error {
		if len(req.IDs) == 0 {
			return nil
		}

		m := agentdb.Memories
		if _, err := m.Where(m.ID.In(req.IDs...)).Update(
			ctx,
			r.db.Keel(ctx),
			m.UseCount.Add(1),
			m.LastUsedAt.To(req.At),
			m.UpdatedAt.ToExpr(m.UpdatedAt),
			m.Version.ToExpr(m.Version),
		); err != nil {
			return fmt.Errorf("mark agent memories used: %w", err)
		}

		return nil
	})
}

func (r *repository) ListSuggestionContext(
	ctx context.Context,
	req agentports.ListAgentMemorySuggestionContextRequest,
) ([]*agent.Memory, error) {
	return dbtx.Read(ctx, r.db, func(ctx context.Context) ([]*agent.Memory, error) {
		limit := req.Limit
		if limit <= 0 || limit > maxSuggestionContextLimit {
			limit = maxSuggestionContextLimit
		}

		m := agentdb.Memories
		rows, err := m.Where(keel.Or(
			keel.And(
				activeOnly(req.Now),
				m.Scope.In(agentScopes()...),
				keel.Or(
					m.AgentDefinitionID.IsNull(),
					m.AgentDefinitionID.Eq(req.AgentDefinitionID),
				),
			),
			keel.And(
				m.Status.Eq(agent.MemoryStatusSuggested),
				m.AgentDefinitionID.Eq(req.AgentDefinitionID),
			),
			keel.And(
				m.Status.Eq(agent.MemoryStatusDismissed),
				m.AgentDefinitionID.Eq(req.AgentDefinitionID),
				m.RetiredAt.Gte(req.DismissedSince),
			),
		)).
			Order(m.CreatedAt.Desc()).
			Limit(limit).
			List(ctx, r.db.Keel(ctx))
		if err != nil {
			r.l.Error("failed to list agent memory suggestion context", zap.Error(err))

			return nil, fmt.Errorf("list agent memory suggestion context: %w", err)
		}

		return rows, nil
	})
}

func (r *repository) ResolveSuggestion(
	ctx context.Context,
	req agentports.ResolveAgentMemorySuggestionRequest,
) (*agent.Memory, error) {
	return dbtx.Write(ctx, r.db, func(ctx context.Context) (*agent.Memory, error) {
		m := agentdb.Memories
		from := []agent.MemoryStatus{agent.MemoryStatusSuggested}
		if req.Reconsidered {
			from = append(from, agent.MemoryStatusDismissed)
		}

		sets := []keel.Assignment{m.Status.To(req.Status), m.UpdatedAt.To(req.At)}
		switch req.Status {
		case agent.MemoryStatusActive:
			sets = append(sets, activation(&req)...)
		case agent.MemoryStatusDismissed:
			sets = append(sets, m.RetiredAt.To(req.At), setID(m.RetiredByUserID, req.ByUserID))
		default:
			return nil, fmt.Errorf(
				"resolve agent memory suggestion: %q is not a resolution",
				req.Status,
			)
		}

		n, err := m.Where(
			m.ID.Eq(req.ID),
			m.Status.In(from...),
			m.Version.Eq(req.Version),
		).Update(ctx, r.db.Keel(ctx), sets...)
		if err != nil {
			return nil, fmt.Errorf("resolve agent memory suggestion: %w", err)
		}
		if n == 0 {
			return nil, dberror.CreateVersionMismatchError(memoryEntity, req.ID.String())
		}

		resolved, err := r.GetByID(
			ctx,
			agentports.GetAgentMemoryByIDRequest{ID: req.ID, TenantInfo: req.TenantInfo},
		)
		if err != nil {
			return nil, err
		}
		if resolved.Status == agent.MemoryStatusActive && resolved.Replaces() {
			if err = r.retireSuperseded(ctx, retireSupersededParams{
				id:       *resolved.SupersedesID,
				byUserID: req.ByUserID,
				at:       req.At,
			}); err != nil {
				return nil, err
			}
		}

		return resolved, nil
	})
}

func activation(req *agentports.ResolveAgentMemorySuggestionRequest) []keel.Assignment {
	m := agentdb.Memories
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
	case agent.MemoryScopeOrganization, agent.MemoryScopeAgent:
	}
	reviewedAt := m.ReviewedAt.ToNull()
	if req.ByUserID.IsNotNil() {
		reviewedAt = m.ReviewedAt.To(req.At)
	}

	sets := []keel.Assignment{
		m.Content.To(req.Content),
		m.Kind.To(req.Kind),
		m.Scope.To(scope),
		setID(m.OwnerUserID, owner),
		setID(m.RoleID, role),
		setID(m.ReviewedByUserID, req.ByUserID),
		reviewedAt,
		m.RetiredAt.ToNull(),
		m.RetiredByUserID.ToNull(),
	}
	if req.ByUserID.IsNotNil() {
		sets = append(sets, m.CreatedByUserID.ToExpr(
			keel.Raw("COALESCE(?, ?)", m.CreatedByUserID, req.ByUserID.String()),
		))
	}

	return sets
}

// forAgent keeps the memories a prompt for one agent may carry: those kept
// for the whole organization, those kept for this agent alone, and those kept
// for the person the prompt is for, themselves or one of their roles. A
// prompt for no agent in particular carries none of the agent's, and one for
// nobody in particular none of anybody's.
func forAgent(agentID pulid.ID, reader agent.MemoryReader) keel.Pred {
	m := agentdb.Memories

	return keel.Or(
		m.Scope.Eq(agent.MemoryScopeOrganization),
		keel.When(agentID.IsNotNil(), keel.And(
			m.Scope.Eq(agent.MemoryScopeAgent),
			m.AgentDefinitionID.Eq(agentID),
		)),
		forPerson(reader),
	)
}

// forPerson is, as alternatives, the memories kept for the reader and for
// their roles; nil when the reader is nobody in particular.
func forPerson(reader agent.MemoryReader) keel.Pred {
	m := agentdb.Memories
	if reader.UserID.IsNil() && len(reader.RoleIDs) == 0 {
		return nil
	}

	return keel.Or(
		keel.When(reader.UserID.IsNotNil(), keel.And(
			m.Scope.Eq(agent.MemoryScopeUser),
			m.OwnerUserID.Eq(reader.UserID),
		)),
		keel.When(len(reader.RoleIDs) > 0, keel.And(
			m.Scope.Eq(agent.MemoryScopeRole),
			m.RoleID.In(reader.RoleIDs...),
		)),
	)
}

// sameReaders keeps the rows read by exactly the readers the new memory
// would be: the same agent, the same person or the same role.
func sameReaders(req *agentports.FindActiveAgentMemoryRequest) keel.Pred {
	m := agentdb.Memories
	switch req.Scope {
	case agent.MemoryScopeAgent:
		return keel.And(
			m.Scope.Eq(agent.MemoryScopeAgent),
			m.AgentDefinitionID.Eq(req.AgentDefinitionID),
		)
	case agent.MemoryScopeUser:
		return keel.And(m.Scope.Eq(agent.MemoryScopeUser), m.OwnerUserID.Eq(req.OwnerUserID))
	case agent.MemoryScopeRole:
		return keel.And(m.Scope.Eq(agent.MemoryScopeRole), m.RoleID.Eq(req.RoleID))
	default:
		return m.Scope.Eq(agent.MemoryScopeOrganization)
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

func activeOnly(now int64) keel.Pred {
	return keptOnly(now, false)
}

func keptOnly(now int64, includeSuggested bool) keel.Pred {
	m := agentdb.Memories
	status := m.Status.Eq(agent.MemoryStatusActive)
	if includeSuggested {
		status = m.Status.In(agent.MemoryStatusActive, agent.MemoryStatusSuggested)
	}

	return keel.And(status, keel.Or(m.ExpiresAt.IsNull(), m.ExpiresAt.Gt(now)))
}

func replaceableStatuses() []agent.MemoryStatus {
	return []agent.MemoryStatus{agent.MemoryStatusActive, agent.MemoryStatusPaused}
}

// taintsLast puts a memory that taints the turns reading it after one that
// does not: a clean one, or a tainted one a person has reviewed.
func taintsLast() keel.Order {
	m := agentdb.Memories

	return keel.Asc(keel.Raw(
		"CASE WHEN ? AND ? IS NULL THEN 1 ELSE 0 END",
		m.Tainted,
		m.ReviewedAt,
	))
}

func activeFirst() keel.Order {
	return keel.Asc(keel.Raw(
		"CASE ? WHEN 'Active' THEN 0 ELSE 1 END",
		agentdb.Memories.Status,
	))
}

// kindOrder puts instructions before corrections before procedures before
// facts, the order a prompt reads them in.
func kindOrder() keel.Order {
	return keel.Asc(keel.Raw(
		"CASE ? WHEN 'Instruction' THEN 0 WHEN 'Correction' THEN 1 "+
			"WHEN 'Procedure' THEN 2 ELSE 3 END",
		agentdb.Memories.Kind,
	))
}

func candidateOrder() keel.Order {
	m := agentdb.Memories

	return keel.Asc(keel.Raw(
		"CASE WHEN ? IS NOT NULL THEN 0 WHEN ? IS NULL AND ? = ? THEN 1 "+
			"WHEN ? IS NOT NULL THEN 2 ELSE 3 END",
		m.SubjectType,
		m.ToolName,
		m.Kind,
		string(agent.MemoryKindInstruction),
		m.ToolName,
	))
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

func setID(col keelkinds.PulidIDCol, id pulid.ID) keel.Assignment {
	if id.IsNil() {
		return col.ToNull()
	}

	return col.To(id)
}
