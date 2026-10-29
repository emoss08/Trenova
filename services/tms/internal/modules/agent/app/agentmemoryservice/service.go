// Package agentmemoryservice keeps what an organization has told its agents
// between runs.
//
// A memory is recorded three ways: a person writes one in AI Control, an
// agent records one through its remember tool when told something worth
// keeping, or a decision on a proposal teaches one, because a person changing
// or refusing what an agent proposed is the clearest correction there is and
// used to be thrown away. Every prompt that asks for memory reads the active
// ones back, so a memory is bounded, retired rather than deleted, and never
// recorded twice.
package agentmemoryservice

import (
	"cmp"
	"context"
	"fmt"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemoryrank"
	"github.com/emoss08/trenova/internal/modules/agent/app/agentmemorysubjects"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdecision"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentmemory"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentmemoryports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/retrievalports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/rankfusion"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	// maxValueChars bounds a proposed or corrected value in a correction, so
	// a long note in one parameter does not become the whole memory.
	maxValueChars = 80

	recallCandidateFactor = 4
)

type runReader interface {
	GetByID(
		ctx context.Context,
		req agentruntimeports.GetAgentRunByIDRequest,
	) (*agent.AgentRun, error)
}

type actionLogger interface {
	LogAction(params *auditports.LogActionParams, opts ...auditports.LogOption) error
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         agentmemoryports.AgentMemoryRepository
	Runs         agentruntimeports.AgentRunRepository
	Labeler      agentmemorysubjects.SubjectLabeler
	AuditService auditports.AuditService
	// Subjects reads which records a shipment, a document or a message
	// names. Without it a turn reads the memories of the records it is
	// directly about and no others.
	Subjects agentmemoryports.AgentMemorySubjectRepository `optional:"true"`
	// Ranker orders the memories a prompt may carry. Without one they are
	// ordered by how recently they were recorded and how often they are used.
	Ranker  agentmemoryports.MemoryRanker       `optional:"true"`
	Indexer retrievalports.RetrievalIndexer     `optional:"true"`
	Vectors retrievalports.MemoryVectorSearcher `optional:"true"`
	// Roles says which roles a person holds, so a memory kept for a role
	// reaches its holders. Without it only the organization's and a person's
	// own memories are read.
	Roles permissionports.RoleRepository `optional:"true"`
	// Security records a review that clears a memory's taint as a critical
	// change: it lets the turns that read the memory write on their own again.
	Security permissionports.SecurityAuditor `optional:"true"`
	// Permissions says who may manage shared memory. Without it nobody does:
	// every memory for a team, the organization or every agent is offered.
	Permissions permissionports.PermissionEngine `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	repo        agentmemoryports.AgentMemoryRepository
	runs        runReader
	labeler     agentmemorysubjects.SubjectLabeler
	audit       actionLogger
	subjects    agentmemorysubjects.SubjectResolver
	ranker      agentmemoryports.MemoryRanker
	indexer     retrievalports.RetrievalIndexer
	vectors     retrievalports.MemoryVectorSearcher
	roles       roleReader
	security    permissionports.SecurityAuditor
	permissions permissionports.PermissionEngine
}

//nolint:gocritic // fx requires an fx.In parameter struct to be passed by value.
func New(p Params) agentmemoryports.AgentMemoryService {
	ranker := p.Ranker
	if ranker == nil {
		ranker = agentmemoryrank.NewRecencyRanker()
	}

	return &Service{
		l:           p.Logger.Named("service.agentmemory"),
		repo:        p.Repo,
		runs:        p.Runs,
		labeler:     p.Labeler,
		audit:       p.AuditService,
		subjects:    agentmemorysubjects.NewSubjectResolver(p.Subjects),
		ranker:      ranker,
		indexer:     p.Indexer,
		vectors:     p.Vectors,
		roles:       rolesOrNil(p.Roles),
		security:    p.Security,
		permissions: p.Permissions,
	}
}

func (s *Service) MayManageShared(
	ctx context.Context,
	tenant pagination.TenantInfo,
	userID pulid.ID,
) bool {
	if s.permissions == nil || userID.IsNil() {
		return false
	}

	actor := permissionports.UserActor(
		pagination.TenantInfo{OrgID: tenant.OrgID, BuID: tenant.BuID, UserID: userID},
	)
	result, err := s.permissions.Check(
		ctx,
		actor.PermissionCheck(permission.ResourceAgentMemory, permission.OpManage),
	)
	if err != nil {
		s.l.Warn(
			"agent memory: the person's manage permission could not be read; treating shared memory as offered",
			zap.String("user", userID.String()),
			zap.Error(err),
		)

		return false
	}

	return result != nil && result.Allowed
}

// manages reports whether the actor keeps shared memory: a person holding
// agent-memory manage. An agent principal never does, so what a run records
// for the organization is offered for a person to approve.
func (s *Service) manages(
	ctx context.Context,
	tenant pagination.TenantInfo,
	actor *permissionports.RequestActor,
) bool {
	if actor == nil || !actor.IsUser() {
		return false
	}

	return s.MayManageShared(ctx, tenant, actor.UserID)
}

// errSharedMemoryManaged is the refusal for keeping or changing a memory other
// people's conversations read without the permission to.
func errSharedMemoryManaged() error {
	return errortypes.NewAuthorizationError(
		"Only an administrator keeps, changes, pauses or forgets a memory shared with a " +
			"team, the organization or every agent: it needs permission to manage agent " +
			"memory",
	)
}

func (s *Service) Remember(
	ctx context.Context,
	req *agentmemoryports.RememberRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	return s.remember(ctx, req, actor, s.manages(ctx, req.TenantInfo, actor))
}

// remember keeps or offers the memory for an actor whose right to manage
// shared memory is already known: the Desk reads it with the person, every
// other caller asks the permission engine.
func (s *Service) remember(
	ctx context.Context,
	req *agentmemoryports.RememberRequest,
	actor *permissionports.RequestActor,
	manages bool,
) (*agentmemory.Memory, error) {
	plan, err := s.planRemember(ctx, req, actor, manages)
	if err != nil {
		return nil, err
	}
	if plan.Existing != nil {
		refreshed, refreshErr := s.refresh(ctx, req.TenantInfo, plan.Existing)
		if refreshErr != nil {
			return nil, refreshErr
		}
		if plan.Replaced != nil && !plan.Held && !refreshed.Status.IsSuggestion() {
			s.retireReplaced(ctx, plan.Replaced, refreshed, actor)
		}

		return refreshed, nil
	}

	created, err := s.repo.Create(ctx, plan.Memory)
	if err != nil {
		return nil, err
	}

	s.log(created, actor, permission.OpCreate, "Agent memory recorded")
	s.queueForRetrieval(ctx, created)
	if created.Status == agentmemory.MemoryStatusActive && created.Replaces() {
		s.recordReplaced(ctx, created, actor)
	}

	return created, nil
}

// refresh keeps the memory that already says what was asked to be saved, in
// place of a duplicate beside it. Being told it again is a use: the memory is
// counted as used now, so a memory people keep restating never goes stale,
// and it is marked Refreshed so the caller can say nothing new was recorded.
func (s *Service) refresh(
	ctx context.Context,
	tenant pagination.TenantInfo,
	existing *agentmemory.Memory,
) (*agentmemory.Memory, error) {
	existing.Refreshed = true
	if existing.Status.IsSuggestion() {
		return existing, nil
	}

	now := timeutils.NowUnix()
	if err := s.repo.MarkUsed(ctx, agentmemoryports.MarkAgentMemoriesUsedRequest{
		TenantInfo: tenant,
		IDs:        []pulid.ID{existing.ID},
		At:         now,
	}); err != nil {
		return nil, err
	}

	existing.UseCount++
	existing.LastUsedAt = &now

	return existing, nil
}

func (s *Service) PreviewRemember(
	ctx context.Context,
	req *agentmemoryports.RememberRequest,
	actor *permissionports.RequestActor,
) (*agentmemoryports.RememberPlan, error) {
	return s.planRemember(ctx, req, actor, s.manages(ctx, req.TenantInfo, actor))
}

func (s *Service) planRemember(
	ctx context.Context,
	req *agentmemoryports.RememberRequest,
	actor *permissionports.RequestActor,
	manages bool,
) (*agentmemoryports.RememberPlan, error) {
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"An actor is required",
		)
	}

	entity := NewMemory(req, actor)

	if req.RunID.IsNotNil() {
		if run, err := s.runs.GetByID(ctx, agentruntimeports.GetAgentRunByIDRequest{
			ID:         req.RunID,
			TenantInfo: &req.TenantInfo,
		}); err != nil {
			s.l.Warn(
				"agent memory: run lookup failed",
				zap.String("run", req.RunID.String()),
				zap.Error(err),
			)
		} else {
			if run.AgentDefinitionID.IsNotNil() {
				definitionID := run.AgentDefinitionID
				entity.AgentDefinitionID = &definitionID
			}
			// An assistant run answers a conversation, which is where the
			// memory says it came from.
			if run.SubjectType == agent.SubjectAssistantThread && run.SubjectID.IsNotNil() {
				threadID := run.SubjectID
				entity.SourceThreadID = &threadID
			}
		}
	}

	plan := &agentmemoryports.RememberPlan{Memory: entity}
	if req.Replaces.IsNotNil() {
		replaced, held, err := s.planReplacement(ctx, req, entity)
		if err != nil {
			return nil, err
		}
		plan.Replaced = replaced
		plan.Held = held
		if held {
			entity.Status = agentmemory.MemoryStatusSuggested
		}
	}

	// A memory for a team, the organization or every agent reaches other
	// people's conversations, so only a person who manages shared memory
	// keeps one at once; anyone else, and every run with nobody in it,
	// offers it for an administrator to approve. The readers are final here:
	// a replacement has taken the replaced memory's.
	if entity.Scope.Shared() && entity.Status == agentmemory.MemoryStatusActive && !manages {
		entity.Status = agentmemory.MemoryStatusSuggested
	}

	if err := s.label(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	same := sameMemoryRequest(req.TenantInfo, entity)
	same.IncludeSuggested = entity.Status == agentmemory.MemoryStatusSuggested
	existing, err := s.repo.FindActive(ctx, same)
	if err != nil {
		return nil, err
	}
	plan.Existing = existing

	return plan, nil
}

func NewMemory(
	req *agentmemoryports.RememberRequest,
	actor *permissionports.RequestActor,
) *agentmemory.Memory {
	kind := req.Kind
	if kind == "" {
		kind = agentmemory.MemoryKindFact
	}

	entity := &agentmemory.Memory{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Kind:           kind,
		Source:         agentmemory.MemorySourceUser,
		Status:         agentmemory.MemoryStatusActive,
		SubjectType:    req.SubjectType,
		ToolName:       strings.TrimSpace(req.ToolName),
		Content:        strings.TrimSpace(req.Content),
		ExpiresAt:      req.ExpiresAt,
	}
	if req.SubjectID.IsNotNil() {
		id := req.SubjectID
		entity.SubjectID = &id
	}
	if actor != nil && actor.IsUser() && actor.UserID.IsNotNil() {
		id := actor.UserID
		entity.CreatedByUserID = &id
	}
	if req.PersonUserID.IsNotNil() {
		id := req.PersonUserID
		entity.CreatedByUserID = &id
	}
	if req.RunID.IsNotNil() {
		entity.Source = agentmemory.MemorySourceAgent
		runID := req.RunID
		entity.SourceRunID = &runID
	}
	if req.ProposalID.IsNotNil() {
		proposalID := req.ProposalID
		entity.SourceProposalID = &proposalID
	}
	if req.Taint.Tainted() {
		entity.Tainted = true
		if req.RunID.IsNotNil() {
			runID := req.RunID
			entity.TaintRunID = &runID
		}
	}
	if req.Scope.IsValid() {
		entity.SetAudience(req.Scope, req.OwnerUserID, req.RoleID)
	}
	if req.Suggest {
		entity.Status = agentmemory.MemoryStatusSuggested
	}
	if req.Source.IsValid() {
		entity.Source = req.Source
	}
	entity.AgentDefinitionID = pulid.PtrOrNil(req.AgentDefinitionID)
	entity.SourceThreadID = pulid.PtrOrNil(req.ThreadID)
	entity.ReflectionID = pulid.PtrOrNil(req.ReflectionID)
	entity.SupersedesID = pulid.PtrOrNil(req.Replaces)
	entity.Evidence = req.Evidence

	return entity
}

func (s *Service) Update(
	ctx context.Context,
	req *agentmemoryports.UpdateAgentMemoryRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a person can change what an agent remembers",
		)
	}

	entity, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if entity.Scope.Shared() && !s.manages(ctx, req.TenantInfo, actor) {
		return nil, errSharedMemoryManaged()
	}

	previous := jsonutils.MustToJSON(entity)
	entity.Kind = req.Kind
	entity.Content = strings.TrimSpace(req.Content)
	entity.SubjectType = req.SubjectType
	entity.SubjectID = nil
	entity.SubjectLabel = ""
	if req.SubjectID.IsNotNil() {
		id := req.SubjectID
		entity.SubjectID = &id
	}
	entity.ToolName = strings.TrimSpace(req.ToolName)
	entity.ExpiresAt = req.ExpiresAt
	entity.Version = req.Version

	if err = s.label(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	updated, err := s.repo.Update(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logChange(updated, previous, actor, permission.OpUpdate, "Agent memory changed")
	s.queueForRetrieval(ctx, updated)

	return updated, nil
}

// mayChangeStatus says who resumes, pauses or retires a memory: a shared one
// takes a person who manages shared memory; a person's own takes that person,
// or such a manager.
func mayChangeStatus(
	current *agentmemory.Memory,
	actor *permissionports.RequestActor,
	manages bool,
) error {
	if current.Scope.Shared() {
		if manages {
			return nil
		}

		return errSharedMemoryManaged()
	}
	if actor.IsUser() && current.OwnerUserID != nil && *current.OwnerUserID == actor.UserID {
		return nil
	}
	if manages {
		return nil
	}

	return errortypes.NewAuthorizationError(
		"Only the person a memory is kept for, or someone who manages agent memory, " +
			"pauses, forgets or restores it",
	)
}

func (s *Service) SetStatus(
	ctx context.Context,
	req *agentmemoryports.SetAgentMemoryStatusRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	return s.setStatus(ctx, req, actor, s.manages(ctx, req.TenantInfo, actor))
}

func (s *Service) PreviewSetStatus(
	ctx context.Context,
	req *agentmemoryports.SetAgentMemoryStatusRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	return s.statusTarget(ctx, req, actor, s.manages(ctx, req.TenantInfo, actor))
}

// statusTarget is the memory a status change would act on, once the change
// and the actor's right to make it have been checked; a preview reads it and
// the write goes on from it.
func (s *Service) statusTarget(
	ctx context.Context,
	req *agentmemoryports.SetAgentMemoryStatusRequest,
	actor *permissionports.RequestActor,
	manages bool,
) (*agentmemory.Memory, error) {
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"An actor is required",
		)
	}
	if err := checkStatusTarget(req.Status); err != nil {
		return nil, err
	}

	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = mayChangeStatus(current, actor, manages); err != nil {
		return nil, err
	}

	return current, nil
}

func (s *Service) setStatus(
	ctx context.Context,
	req *agentmemoryports.SetAgentMemoryStatusRequest,
	actor *permissionports.RequestActor,
	manages bool,
) (*agentmemory.Memory, error) {
	current, err := s.statusTarget(ctx, req, actor, manages)
	if err != nil {
		return nil, err
	}

	change := StatusChange{
		Status:   req.Status,
		ByUserID: StatusActor(actor),
		At:       timeutils.NowUnix(),
	}
	planned := *current
	if err = PlanStatus(&planned, change); err != nil {
		return nil, err
	}

	updated, err := s.repo.SetStatus(ctx, agentmemoryports.SetAgentMemoryStatusQuery{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Status:     change.Status,
		ByUserID:   change.ByUserID,
		At:         change.At,
	})
	if err != nil {
		return nil, err
	}

	comment := statusComment(current.Status, req.Status)
	s.log(updated, actor, permission.OpUpdate, comment)
	s.queueForRetrieval(ctx, updated)

	return updated, nil
}

func (s *Service) Review(
	ctx context.Context,
	req agentmemoryports.ReviewAgentMemoryRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a person can review what an agent remembered",
		)
	}

	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if err = reviewable(current); err != nil {
		return nil, err
	}

	reviewed, err := s.repo.Review(ctx, agentmemoryports.ReviewAgentMemoryQuery{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		ByUserID:   actor.UserID,
		At:         timeutils.NowUnix(),
		Version:    req.Version,
	})
	if err != nil {
		return nil, err
	}

	s.recordReview(ctx, current, reviewed, actor)
	s.queueForRetrieval(ctx, reviewed)

	return reviewed, nil
}

// reviewable refuses a review that would change nothing: a memory no run
// tainted, one already reviewed, or one that is no longer kept.
func reviewable(memory *agentmemory.Memory) error {
	switch {
	case !memory.Tainted:
		return errortypes.NewValidationError(
			"tainted",
			errortypes.ErrInvalid,
			"This memory was not written after outside content, so there is nothing to review",
		)
	case memory.Reviewed():
		return errortypes.NewBusinessError("This memory has already been reviewed")
	case memory.Status != agentmemory.MemoryStatusActive && memory.Status != agentmemory.MemoryStatusPaused:
		return errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only an active or paused memory can be reviewed; approve a suggested one instead",
		)
	default:
		return nil
	}
}

func (s *Service) recordReview(
	ctx context.Context,
	before, after *agentmemory.Memory,
	actor *permissionports.RequestActor,
) {
	const comment = "Agent memory reviewed: it no longer taints the turns that read it"
	if s.security == nil {
		s.logChange(after, jsonutils.MustToJSON(before), actor, permission.OpUpdate, comment)

		return
	}

	s.security.RecordChange(ctx, &permissionports.SecurityChange{
		Resource:       permission.ResourceAgentMemory,
		ResourceID:     after.ID.String(),
		Operation:      permission.OpUpdate,
		Actor:          actor.AuditActor(),
		OrganizationID: after.OrganizationID,
		BusinessUnitID: after.BusinessUnitID,
		Before:         jsonutils.MustToJSON(before),
		After:          jsonutils.MustToJSON(after),
		Comment:        comment,
		Metadata: map[string]any{
			"taintRunId": pulid.ConvertFromPtr(before.TaintRunID).String(),
			"kind":       string(after.Kind),
			"scope":      string(after.Scope),
		},
	})
}

func (s *Service) ApproveSuggestion(
	ctx context.Context,
	req *agentmemoryports.ApproveAgentMemorySuggestionRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a person can approve a suggested memory",
		)
	}

	current, err := s.suggestion(ctx, req.ID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	kind := req.Kind
	if kind == "" {
		kind = current.Kind
	}
	candidate := *current
	candidate.Kind = kind
	candidate.Content = strings.TrimSpace(req.Content)
	candidate.Status = agentmemory.MemoryStatusActive
	candidate.Scope = approvedScope(req.Scope, current)

	me := errortypes.NewMultiError()
	candidate.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	approved, err := s.repo.ResolveSuggestion(
		ctx,
		agentmemoryports.ResolveAgentMemorySuggestionRequest{
			ID:          req.ID,
			TenantInfo:  req.TenantInfo,
			Status:      agentmemory.MemoryStatusActive,
			Kind:        candidate.Kind,
			Content:     candidate.Content,
			Scope:       candidate.Scope,
			OwnerUserID: pulid.ConvertFromPtr(current.OwnerUserID),
			RoleID:      pulid.ConvertFromPtr(current.RoleID),
			ByUserID:    actor.UserID,
			At:          timeutils.NowUnix(),
			Version:     req.Version,
		},
	)
	if err != nil {
		return nil, err
	}

	if current.Tainted {
		s.recordReview(ctx, current, approved, actor)
	} else {
		s.logChange(
			approved,
			jsonutils.MustToJSON(current),
			actor,
			permission.OpUpdate,
			"Suggested agent memory approved",
		)
	}
	s.queueForRetrieval(ctx, approved)
	s.recordReplaced(ctx, approved, actor)

	return approved, nil
}

// approvedScope is who an approved suggestion is read by. One offered by an
// agent or a person is kept for the readers it was offered to; one drawn from
// ratings of an agent's work is kept for that agent alone. An administrator
// may widen either to the whole organization.
func approvedScope(
	requested agentmemory.MemoryScope,
	suggestion *agentmemory.Memory,
) agentmemory.MemoryScope {
	if requested == agentmemory.MemoryScopeOrganization {
		return agentmemory.MemoryScopeOrganization
	}
	offered := suggestion.Source.OfferedByAgent() ||
		suggestion.Source == agentmemory.MemorySourceUser
	if offered && suggestion.Scope.IsValid() {
		return suggestion.Scope
	}
	if suggestion.AgentDefinitionID == nil || suggestion.AgentDefinitionID.IsNil() {
		return agentmemory.MemoryScopeOrganization
	}

	return agentmemory.MemoryScopeAgent
}

func (s *Service) DismissSuggestion(
	ctx context.Context,
	req agentmemoryports.DismissAgentMemorySuggestionRequest,
	actor *permissionports.RequestActor,
) (*agentmemory.Memory, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a person can dismiss a suggested memory",
		)
	}

	current, err := s.suggestion(ctx, req.ID, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	dismissed, err := s.repo.ResolveSuggestion(
		ctx,
		agentmemoryports.ResolveAgentMemorySuggestionRequest{
			ID:         req.ID,
			TenantInfo: req.TenantInfo,
			Status:     agentmemory.MemoryStatusDismissed,
			ByUserID:   actor.UserID,
			At:         timeutils.NowUnix(),
			Version:    req.Version,
		},
	)
	if err != nil {
		return nil, err
	}

	s.logChange(
		dismissed,
		jsonutils.MustToJSON(current),
		actor,
		permission.OpUpdate,
		"Suggested agent memory dismissed",
	)

	return dismissed, nil
}

func (s *Service) suggestion(
	ctx context.Context,
	id pulid.ID,
	tenant pagination.TenantInfo,
) (*agentmemory.Memory, error) {
	current, err := s.repo.GetByID(ctx, agentmemoryports.GetAgentMemoryByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if current.Status != agentmemory.MemoryStatusSuggested {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Only a suggested memory can be approved or dismissed",
		)
	}

	return current, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req agentmemoryports.GetAgentMemoryByIDRequest,
) (*agentmemory.Memory, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentmemoryports.ListAgentMemoryConnectionRequest,
) (*pagination.CursorListResult[*agentmemory.Memory], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Recall(
	ctx context.Context,
	req *agentmemoryports.RecallAgentMemoriesRequest,
) ([]agentmemoryports.RecalledMemory, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = agentmemory.DefaultMemoryRecallLimit
	}
	limit = min(limit, agentmemory.MaxMemoryRecallLimit)

	reader := s.promptReader(ctx, req.TenantInfo, req.ReaderUserID)

	search := agentmemoryports.SearchAgentMemoriesRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Reader:            reader,
		Now:               timeutils.NowUnix(),
		Query:             strings.TrimSpace(req.Query),
		Terms:             agentmemoryrank.RecallTerms(req.Query),
		IDs:               req.IDs,
		Kind:              req.Kind,
		ToolName:          req.ToolName,
		Limit:             limit,
	}
	if req.SubjectType != "" && req.SubjectID.IsNotNil() {
		search.Subject = &agentmemoryports.MemorySubjectRef{
			Type: req.SubjectType,
			ID:   req.SubjectID,
		}
	}

	found, err := s.repo.Search(ctx, search)
	if err != nil {
		return nil, err
	}
	if search.Query == "" {
		return recalledAs(found, ""), nil
	}

	similar, ok := s.similar(ctx, req, &search)
	if !ok {
		return recalledAs(found, agentmemory.MemoryMatchWords), nil
	}

	return s.fuseRecall(ctx, &search, found, similar)
}

func recalledAs(
	memories []*agentmemory.Memory,
	match agentmemory.MemoryMatch,
) []agentmemoryports.RecalledMemory {
	recalled := make([]agentmemoryports.RecalledMemory, 0, len(memories))
	for _, memory := range memories {
		recalled = append(recalled, agentmemoryports.RecalledMemory{Memory: memory, Match: match})
	}

	return recalled
}

func (s *Service) similar(
	ctx context.Context,
	req *agentmemoryports.RecallAgentMemoriesRequest,
	search *agentmemoryports.SearchAgentMemoriesRequest,
) (retrievalports.SimilarMemories, bool) {
	if s.vectors == nil {
		return retrievalports.SimilarMemories{}, false
	}

	similar, err := s.vectors.SimilarMemories(ctx, &retrievalports.SimilarMemoriesRequest{
		TenantInfo:  search.TenantInfo,
		Text:        search.Query,
		Limit:       search.Limit * recallCandidateFactor,
		Attribution: req.Attribution,
	})
	if err != nil {
		s.l.Warn("agent memory: recalling by meaning failed; recalling by words",
			zap.String("organization", search.TenantInfo.OrgID.String()),
			zap.Error(err),
		)
		return retrievalports.SimilarMemories{}, false
	}

	return similar, similar.Semantics.Used
}

func (s *Service) fuseRecall(
	ctx context.Context,
	search *agentmemoryports.SearchAgentMemoriesRequest,
	found []*agentmemory.Memory,
	similar retrievalports.SimilarMemories,
) ([]agentmemoryports.RecalledMemory, error) {
	byID := make(map[pulid.ID]*agentmemory.Memory, len(found)+len(similar.Memories))
	keywordIDs := make([]pulid.ID, 0, len(found))
	for _, memory := range found {
		byID[memory.ID] = memory
		keywordIDs = append(keywordIDs, memory.ID)
	}

	missing := make([]pulid.ID, 0, len(similar.Memories))
	for _, hit := range similar.Memories {
		if _, ok := byID[hit.MemoryID]; !ok && hit.Similarity >= similar.Floor {
			missing = append(missing, hit.MemoryID)
		}
	}
	if len(missing) > 0 {
		narrowed := *search
		narrowed.Query = ""
		narrowed.IDs = missing
		narrowed.Limit = len(missing)
		loaded, err := s.repo.Search(ctx, narrowed)
		if err != nil {
			return nil, err
		}
		for _, memory := range loaded {
			byID[memory.ID] = memory
		}
	}

	vectorIDs := make([]pulid.ID, 0, len(similar.Memories))
	for _, hit := range similar.Memories {
		if _, ok := byID[hit.MemoryID]; ok &&
			(hit.Similarity >= similar.Floor || slices.Contains(keywordIDs, hit.MemoryID)) {
			vectorIDs = append(vectorIDs, hit.MemoryID)
		}
	}

	return FuseRecall(byID, keywordIDs, vectorIDs, search.Limit), nil
}

func FuseRecall(
	byID map[pulid.ID]*agentmemory.Memory,
	keywordIDs, vectorIDs []pulid.ID,
	limit int,
) []agentmemoryports.RecalledMemory {
	scores := rankfusion.Reciprocal(rankfusion.DefaultK, keywordIDs, vectorIDs)
	order := make([]pulid.ID, 0, len(scores))
	order = append(order, keywordIDs...)
	for _, id := range vectorIDs {
		if !slices.Contains(order, id) {
			order = append(order, id)
		}
	}
	slices.SortStableFunc(order, func(a, b pulid.ID) int {
		return cmp.Compare(scores[b], scores[a])
	})

	recalled := make([]agentmemoryports.RecalledMemory, 0, min(len(order), limit))
	for _, id := range order {
		memory, ok := byID[id]
		if !ok {
			continue
		}
		recalled = append(recalled, agentmemoryports.RecalledMemory{
			Memory: memory,
			Match:  recallMatch(slices.Contains(keywordIDs, id), slices.Contains(vectorIDs, id)),
		})
		if len(recalled) == limit {
			break
		}
	}

	return recalled
}

func recallMatch(words, meaning bool) agentmemory.MemoryMatch {
	switch {
	case words && meaning:
		return agentmemory.MemoryMatchBoth
	case meaning:
		return agentmemory.MemoryMatchMeaning
	default:
		return agentmemory.MemoryMatchWords
	}
}

func (s *Service) ForContext(
	ctx context.Context,
	req *agentmemoryports.MemoryContextRequest,
) (*agentmemoryports.MemoryContext, error) {
	now := timeutils.NowUnix()

	subjects, err := s.subjects.Resolve(ctx, req.TenantInfo, req.Records)
	if err != nil {
		s.l.Warn("agent memory: the records a turn is about could not all be read",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)
	}

	reader := s.promptReader(ctx, req.TenantInfo, req.ReaderUserID)

	memories, err := s.repo.ListActive(ctx, agentmemoryports.ListActiveAgentMemoriesRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Reader:            reader,
		Now:               now,
		OrganizationWide:  true,
		Subjects:          agentmemorysubjects.SubjectRefs(subjects),
		ToolNames:         req.ToolNames,
		Limit:             agentmemory.MaxMemoryCandidates,
	})
	if err != nil {
		return nil, err
	}

	ranker := s.ranker
	if ranker == nil {
		ranker = agentmemoryrank.NewRecencyRanker()
	}
	ranked, err := ranker.RankMemories(ctx, &agentmemoryports.RankMemoriesRequest{
		TenantInfo: req.TenantInfo,
		Now:        now,
		Memories:   memories,
		Query:      req.Query,
	})
	if err != nil {
		s.l.Warn("agent memory: ranking failed; ordering by recency and use",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)
		ranked = agentmemoryports.RankedMemories{
			Memories: agentmemoryrank.RankByRecencyAndUse(memories, now),
		}
	}

	return &agentmemoryports.MemoryContext{
		Memories:  ranked.Memories,
		Subjects:  subjects,
		Relevance: agentmemoryrank.Relevance(req.Focus, req.Text, ranked),
	}, nil
}

func (s *Service) RecordUse(
	ctx context.Context,
	req agentmemoryports.RecordMemoryUseRequest,
) error {
	if len(req.IDs) == 0 {
		return nil
	}

	return s.repo.MarkUsed(ctx, agentmemoryports.MarkAgentMemoriesUsedRequest{
		TenantInfo: req.TenantInfo,
		IDs:        req.IDs,
		At:         timeutils.NowUnix(),
	})
}

func (s *Service) Usage(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*agentmemoryports.AgentMemoryUsage, error) {
	count, err := s.repo.CountActive(ctx, agentmemoryports.CountActiveAgentMemoriesRequest{
		TenantInfo: tenant,
		Now:        timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	return &agentmemoryports.AgentMemoryUsage{
		ActiveCount:   count,
		ActiveSoftCap: agentmemory.MemoryActiveSoftCap,
		WarnAt:        agentmemory.MemoryActiveWarnAt,
	}, nil
}

// RecordCorrection keeps what a decision taught. A change to the proposed
// parameters is always worth keeping, with the reason when a person typed
// one; a rejection is worth keeping only when a person said why, because
// "rejected" alone teaches the next run nothing it can act on.
func (s *Service) RecordCorrection(
	ctx context.Context,
	proposal *agent.AgentProposal,
	decision *agentdecision.AgentDecision,
) (*agentmemory.Memory, error) {
	if proposal == nil || decision == nil {
		return nil, nil
	}

	content := correctionContent(proposal, decision)
	if content == "" {
		return nil, nil
	}

	entity := &agentmemory.Memory{
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Kind:           agentmemory.MemoryKindCorrection,
		Source:         agentmemory.MemorySourceDecision,
		Status:         agentmemory.MemoryStatusActive,
		ToolName:       proposal.ToolName,
		Content:        content,
	}
	runID := proposal.RunID
	entity.SourceRunID = &runID
	proposalID := proposal.ID
	entity.SourceProposalID = &proposalID
	if decision.DecidedByUserID.IsNotNil() {
		userID := decision.DecidedByUserID
		entity.CreatedByUserID = &userID
	}

	tenant := pagination.TenantInfo{OrgID: proposal.OrganizationID, BuID: proposal.BusinessUnitID}
	existing, err := s.repo.FindActive(ctx, sameMemoryRequest(tenant, entity))
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}
	s.queueForRetrieval(ctx, created)

	return created, nil
}

// label resolves the subject's name so a prompt line and a list row can say
// which customer or driver the memory is about. A subject that cannot be
// found is refused: a memory about a record that does not exist would be
// read by nobody and mislead anyone who saw it.
func (s *Service) label(
	ctx context.Context,
	tenant pagination.TenantInfo,
	entity *agentmemory.Memory,
) error {
	if entity.SubjectType == "" || entity.SubjectID == nil || s.labeler == nil {
		return nil
	}

	label, err := s.labeler.Label(ctx, tenant, agentmemoryports.MemorySubjectRef{
		Type: entity.SubjectType,
		ID:   *entity.SubjectID,
	})
	if err != nil {
		return errortypes.NewValidationError(
			"subjectId",
			errortypes.ErrInvalid,
			fmt.Sprintf(
				"No %s with that id is visible to you",
				strings.ToLower(string(entity.SubjectType)),
			),
		)
	}
	entity.SubjectLabel = stringutils.Ellipsize(label, 200)

	return nil
}

func (s *Service) log(
	entity *agentmemory.Memory,
	actor *permissionports.RequestActor,
	op permission.Operation,
	comment string,
) {
	s.logChange(entity, nil, actor, op, comment)
}

func (s *Service) logChange(
	entity *agentmemory.Memory,
	previous map[string]any,
	actor *permissionports.RequestActor,
	op permission.Operation,
	comment string,
) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentMemory,
		ResourceID:     entity.ID.String(),
		Operation:      op,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(entity),
		PreviousState:  previous,
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent memory audit", zap.Error(err))
	}
}

func sameMemoryRequest(
	tenant pagination.TenantInfo,
	entity *agentmemory.Memory,
) agentmemoryports.FindActiveAgentMemoryRequest {
	scope := entity.Scope
	if scope == "" {
		scope = agentmemory.MemoryScopeOrganization
	}
	agentID := pulid.Nil
	if entity.AgentDefinitionID != nil {
		agentID = *entity.AgentDefinitionID
	}
	owner, role := pulid.Nil, pulid.Nil
	if entity.OwnerUserID != nil {
		owner = *entity.OwnerUserID
	}
	if entity.RoleID != nil {
		role = *entity.RoleID
	}

	return agentmemoryports.FindActiveAgentMemoryRequest{
		OwnerUserID:       owner,
		RoleID:            role,
		TenantInfo:        tenant,
		Now:               timeutils.NowUnix(),
		Content:           entity.Content,
		Subject:           subjectRefOf(entity),
		ToolName:          entity.ToolName,
		Scope:             scope,
		AgentDefinitionID: agentID,
		Tainted:           entity.Tainted,
	}
}

func subjectRefOf(entity *agentmemory.Memory) *agentmemoryports.MemorySubjectRef {
	if entity.SubjectType == "" || entity.SubjectID == nil {
		return nil
	}

	return &agentmemoryports.MemorySubjectRef{Type: entity.SubjectType, ID: *entity.SubjectID}
}

var (
	// reasonCodePattern matches a machine reason such as
	// approved_from_activity: one token of lower-case letters, digits and
	// underscores. A sentence a person typed has a space or a capital in it.
	reasonCodePattern = regexp.MustCompile(`^[a-z0-9_]+$`)
	// planSuffixPattern is what the plan service appends to a step's reason.
	planSuffixPattern = regexp.MustCompile(`\s*\(plan [^)]+\)$`)
)

// HumanReason returns what a person wrote, or nothing when the reason is a
// code the client filled in for them.
func HumanReason(reason string) string {
	reason = strings.TrimSpace(planSuffixPattern.ReplaceAllString(reason, ""))
	if reason == "" || reasonCodePattern.MatchString(reason) {
		return ""
	}

	return reason
}

// correctionContent writes the lesson in one or two sentences, or nothing
// when the decision carries none.
func correctionContent(
	proposal *agent.AgentProposal,
	decision *agentdecision.AgentDecision,
) string {
	reason := HumanReason(decision.ReasonCode)

	switch agenttooltrust.OutcomeOfDecision(decision.Decision, decision.Modifications) {
	case agenttooltrust.TrustOutcomeModified:
		changes := describeChanges(proposal.ToolParams, decision.Modifications)
		if len(changes) == 0 {
			return ""
		}
		content := fmt.Sprintf("When %s was proposed, a person changed %s.",
			proposal.ToolName, joinNatural(changes))
		if reason != "" {
			content += " Reason: " + reason
		}

		return stringutils.Ellipsize(content, agentmemory.MaxMemoryContentChars)
	case agenttooltrust.TrustOutcomeRejected:
		if reason == "" {
			return ""
		}

		return stringutils.Ellipsize(
			fmt.Sprintf("A person rejected %s. Reason: %s", proposal.ToolName, reason),
			agentmemory.MaxMemoryContentChars,
		)
	case agenttooltrust.TrustOutcomeApproved, agenttooltrust.TrustOutcomeExecutionFailed:
		return ""
	default:
		return ""
	}
}

// describeChanges lists each parameter the person changed, in key order so
// the same decision always reads the same and is found by the duplicate
// check.
func describeChanges(proposed, modifications map[string]any) []string {
	keys := make([]string, 0, len(modifications))
	for key := range modifications {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	changes := make([]string, 0, len(keys))
	for _, key := range keys {
		next := modifications[key]
		previous, had := proposed[key]
		if had && reflect.DeepEqual(previous, next) {
			continue
		}
		if !had {
			changes = append(changes, fmt.Sprintf("%s to %s", key, describeValue(next)))
			continue
		}
		changes = append(changes, fmt.Sprintf("%s from %s to %s",
			key, describeValue(previous), describeValue(next)))
	}

	return changes
}

func describeValue(value any) string {
	switch v := value.(type) {
	case nil:
		return "nothing"
	case string:
		if strings.TrimSpace(v) == "" {
			return "nothing"
		}

		return stringutils.Ellipsize(v, maxValueChars)
	case bool, int, int32, int64, float32, float64:
		return fmt.Sprint(v)
	default:
		encoded, err := sonic.MarshalString(v)
		if err != nil {
			return fmt.Sprint(v)
		}

		return stringutils.Ellipsize(encoded, maxValueChars)
	}
}

func joinNatural(items []string) string {
	switch len(items) {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:len(items)-1], ", ") + " and " + items[len(items)-1]
	}
}
