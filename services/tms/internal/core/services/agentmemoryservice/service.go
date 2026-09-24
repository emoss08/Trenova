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
	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
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

// SubjectLabeler names the record a memory is about, so the prompt can say
// "Acme Freight (customer)" rather than an id.
type SubjectLabeler interface {
	Label(
		ctx context.Context,
		tenant pagination.TenantInfo,
		ref repositories.MemorySubjectRef,
	) (string, error)
}

type runReader interface {
	GetByID(ctx context.Context, req repositories.GetAgentRunByIDRequest) (*agent.AgentRun, error)
}

type actionLogger interface {
	LogAction(params *services.LogActionParams, opts ...services.LogOption) error
}

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentMemoryRepository
	Runs         repositories.AgentRunRepository
	Labeler      SubjectLabeler
	AuditService services.AuditService
	// Subjects reads which records a shipment, a document or a message
	// names. Without it a turn reads the memories of the records it is
	// directly about and no others.
	Subjects repositories.AgentMemorySubjectRepository `optional:"true"`
	// Ranker orders the memories a prompt may carry. Without one they are
	// ordered by how recently they were recorded and how often they are used.
	Ranker  services.MemoryRanker         `optional:"true"`
	Indexer services.RetrievalIndexer     `optional:"true"`
	Vectors services.MemoryVectorSearcher `optional:"true"`
}

type Service struct {
	l        *zap.Logger
	repo     repositories.AgentMemoryRepository
	runs     runReader
	labeler  SubjectLabeler
	audit    actionLogger
	subjects SubjectResolver
	ranker   services.MemoryRanker
	indexer  services.RetrievalIndexer
	vectors  services.MemoryVectorSearcher
}

func New(p Params) services.AgentMemoryService {
	ranker := p.Ranker
	if ranker == nil {
		ranker = NewRecencyRanker()
	}

	return &Service{
		l:        p.Logger.Named("service.agentmemory"),
		repo:     p.Repo,
		runs:     p.Runs,
		labeler:  p.Labeler,
		audit:    p.AuditService,
		subjects: NewSubjectResolver(p.Subjects),
		ranker:   ranker,
		indexer:  p.Indexer,
		vectors:  p.Vectors,
	}
}

func (s *Service) Remember(
	ctx context.Context,
	req *services.RememberRequest,
	actor *services.RequestActor,
) (*agent.Memory, error) {
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"An actor is required",
		)
	}

	kind := req.Kind
	if kind == "" {
		kind = agent.MemoryKindFact
	}

	entity := &agent.Memory{
		OrganizationID: req.TenantInfo.OrgID,
		BusinessUnitID: req.TenantInfo.BuID,
		Kind:           kind,
		Source:         agent.MemorySourceUser,
		Status:         agent.MemoryStatusActive,
		SubjectType:    req.SubjectType,
		ToolName:       strings.TrimSpace(req.ToolName),
		Content:        strings.TrimSpace(req.Content),
		ExpiresAt:      req.ExpiresAt,
	}
	if req.SubjectID.IsNotNil() {
		id := req.SubjectID
		entity.SubjectID = &id
	}
	if actor.IsUser() && actor.UserID.IsNotNil() {
		id := actor.UserID
		entity.CreatedByUserID = &id
	}

	// A memory recorded from inside a run is the agent's, whoever approved
	// the tool call: the run says which agent, and that is what the list
	// shows as its author.
	if req.RunID.IsNotNil() {
		entity.Source = agent.MemorySourceAgent
		runID := req.RunID
		entity.SourceRunID = &runID
		if run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
			ID:         req.RunID,
			TenantInfo: &req.TenantInfo,
		}); err != nil {
			s.l.Warn(
				"agent memory: run lookup failed",
				zap.String("run", req.RunID.String()),
				zap.Error(err),
			)
		} else if run.AgentDefinitionID.IsNotNil() {
			definitionID := run.AgentDefinitionID
			entity.AgentDefinitionID = &definitionID
		}
	}
	// A memory written by a run that had read outside content keeps that,
	// so a later run that reads it back is tainted by it.
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

	if err := s.label(ctx, req.TenantInfo, entity); err != nil {
		return nil, err
	}

	me := errortypes.NewMultiError()
	entity.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	// The same sentence about the same record is one memory. Returning the
	// existing row is what lets an agent say "noted" twice without the
	// prompt carrying it twice.
	existing, err := s.repo.FindActive(ctx, sameMemoryRequest(req.TenantInfo, entity))
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.log(created, actor, permission.OpCreate, "Agent memory recorded")
	s.queueForRetrieval(ctx, created)

	return created, nil
}

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateAgentMemoryRequest,
	actor *services.RequestActor,
) (*agent.Memory, error) {
	if !actor.IsUser() {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrForbidden,
			"Only a person can change what an agent remembers",
		)
	}

	entity, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
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

func (s *Service) SetStatus(
	ctx context.Context,
	req services.SetAgentMemoryStatusRequest,
	actor *services.RequestActor,
) (*agent.Memory, error) {
	if actor == nil {
		return nil, errortypes.NewValidationError(
			"actor",
			errortypes.ErrRequired,
			"An actor is required",
		)
	}
	if !req.Status.IsValid() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf("%q is not a memory status", req.Status),
		)
	}
	if req.Status.IsSuggestion() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A memory is suggested only by feedback; approve or dismiss the suggestion instead",
		)
	}

	current, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if current.Status.IsSuggestion() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"A suggested memory is approved or dismissed, not retired or restored",
		)
	}

	byUser := pulid.Nil
	if actor.IsUser() {
		byUser = actor.UserID
	}

	updated, err := s.repo.SetStatus(ctx, repositories.SetAgentMemoryStatusRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Status:     req.Status,
		ByUserID:   byUser,
		At:         timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	comment := "Agent memory retired"
	if req.Status == agent.MemoryStatusActive {
		comment = "Agent memory restored"
	}
	s.log(updated, actor, permission.OpUpdate, comment)
	s.queueForRetrieval(ctx, updated)

	return updated, nil
}

func (s *Service) ApproveSuggestion(
	ctx context.Context,
	req *services.ApproveAgentMemorySuggestionRequest,
	actor *services.RequestActor,
) (*agent.Memory, error) {
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
	candidate.Status = agent.MemoryStatusActive
	candidate.Scope = approvedScope(req.Scope, current)

	me := errortypes.NewMultiError()
	candidate.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	approved, err := s.repo.ResolveSuggestion(ctx, repositories.ResolveAgentMemorySuggestionRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Status:     agent.MemoryStatusActive,
		Kind:       candidate.Kind,
		Content:    candidate.Content,
		Scope:      candidate.Scope,
		ByUserID:   actor.UserID,
		At:         timeutils.NowUnix(),
		Version:    req.Version,
	})
	if err != nil {
		return nil, err
	}

	s.logChange(
		approved,
		jsonutils.MustToJSON(current),
		actor,
		permission.OpUpdate,
		"Suggested agent memory approved",
	)
	s.queueForRetrieval(ctx, approved)

	return approved, nil
}

// approvedScope is who an approved suggestion is read by. A suggestion is
// drawn from ratings of one agent's work, so by default it is kept for that
// agent alone; an administrator may widen it to the whole organization.
func approvedScope(requested agent.MemoryScope, suggestion *agent.Memory) agent.MemoryScope {
	if requested == agent.MemoryScopeOrganization {
		return agent.MemoryScopeOrganization
	}
	if suggestion.AgentDefinitionID == nil || suggestion.AgentDefinitionID.IsNil() {
		return agent.MemoryScopeOrganization
	}

	return agent.MemoryScopeAgent
}

func (s *Service) DismissSuggestion(
	ctx context.Context,
	req services.DismissAgentMemorySuggestionRequest,
	actor *services.RequestActor,
) (*agent.Memory, error) {
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

	dismissed, err := s.repo.ResolveSuggestion(ctx, repositories.ResolveAgentMemorySuggestionRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
		Status:     agent.MemoryStatusDismissed,
		ByUserID:   actor.UserID,
		At:         timeutils.NowUnix(),
		Version:    req.Version,
	})
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
) (*agent.Memory, error) {
	current, err := s.repo.GetByID(ctx, repositories.GetAgentMemoryByIDRequest{
		ID:         id,
		TenantInfo: tenant,
	})
	if err != nil {
		return nil, err
	}
	if current.Status != agent.MemoryStatusSuggested {
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
	req repositories.GetAgentMemoryByIDRequest,
) (*agent.Memory, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentMemoryConnectionRequest,
) (*pagination.CursorListResult[*agent.Memory], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) Recall(
	ctx context.Context,
	req services.RecallAgentMemoriesRequest,
) ([]services.RecalledMemory, error) {
	limit := req.Limit
	if limit <= 0 {
		limit = agent.DefaultMemoryRecallLimit
	}
	limit = min(limit, agent.MaxMemoryRecallLimit)

	search := repositories.SearchAgentMemoriesRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Now:               timeutils.NowUnix(),
		Query:             strings.TrimSpace(req.Query),
		IDs:               req.IDs,
		Kind:              req.Kind,
		ToolName:          req.ToolName,
		Limit:             limit,
	}
	if req.SubjectType != "" && req.SubjectID.IsNotNil() {
		search.Subject = &repositories.MemorySubjectRef{Type: req.SubjectType, ID: req.SubjectID}
	}

	found, err := s.repo.Search(ctx, search)
	if err != nil {
		return nil, err
	}
	if search.Query == "" {
		return recalledAs(found, ""), nil
	}

	similar, ok := s.similar(ctx, req, search)
	if !ok {
		return recalledAs(found, agent.MemoryMatchWords), nil
	}

	return s.fuseRecall(ctx, search, found, similar)
}

func recalledAs(memories []*agent.Memory, match agent.MemoryMatch) []services.RecalledMemory {
	recalled := make([]services.RecalledMemory, 0, len(memories))
	for _, memory := range memories {
		recalled = append(recalled, services.RecalledMemory{Memory: memory, Match: match})
	}

	return recalled
}

func (s *Service) similar(
	ctx context.Context,
	req services.RecallAgentMemoriesRequest,
	search repositories.SearchAgentMemoriesRequest,
) (services.SimilarMemories, bool) {
	if s.vectors == nil {
		return services.SimilarMemories{}, false
	}

	similar, err := s.vectors.SimilarMemories(ctx, services.SimilarMemoriesRequest{
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
		return services.SimilarMemories{}, false
	}

	return similar, similar.Semantics.Used
}

func (s *Service) fuseRecall(
	ctx context.Context,
	search repositories.SearchAgentMemoriesRequest,
	found []*agent.Memory,
	similar services.SimilarMemories,
) ([]services.RecalledMemory, error) {
	byID := make(map[pulid.ID]*agent.Memory, len(found)+len(similar.Memories))
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
		narrowed := search
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
	byID map[pulid.ID]*agent.Memory,
	keywordIDs, vectorIDs []pulid.ID,
	limit int,
) []services.RecalledMemory {
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

	recalled := make([]services.RecalledMemory, 0, min(len(order), limit))
	for _, id := range order {
		memory, ok := byID[id]
		if !ok {
			continue
		}
		recalled = append(recalled, services.RecalledMemory{
			Memory: memory,
			Match:  recallMatch(slices.Contains(keywordIDs, id), slices.Contains(vectorIDs, id)),
		})
		if len(recalled) == limit {
			break
		}
	}

	return recalled
}

func recallMatch(words, meaning bool) agent.MemoryMatch {
	switch {
	case words && meaning:
		return agent.MemoryMatchBoth
	case meaning:
		return agent.MemoryMatchMeaning
	default:
		return agent.MemoryMatchWords
	}
}

func (s *Service) ForContext(
	ctx context.Context,
	req services.MemoryContextRequest,
) (*services.MemoryContext, error) {
	now := timeutils.NowUnix()

	subjects, err := s.subjects.Resolve(ctx, req.TenantInfo, req.Records)
	if err != nil {
		s.l.Warn("agent memory: the records a turn is about could not all be read",
			zap.String("organization", req.TenantInfo.OrgID.String()),
			zap.Error(err),
		)
	}

	memories, err := s.repo.ListActive(ctx, repositories.ListActiveAgentMemoriesRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Now:               now,
		OrganizationWide:  true,
		Subjects:          subjectRefs(subjects),
		ToolNames:         req.ToolNames,
		Limit:             agent.MaxMemoryCandidates,
	})
	if err != nil {
		return nil, err
	}

	ranker := s.ranker
	if ranker == nil {
		ranker = NewRecencyRanker()
	}
	ranked, err := ranker.RankMemories(ctx, services.RankMemoriesRequest{
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
		ranked = RankByRecencyAndUse(memories, now)
	}

	return &services.MemoryContext{Memories: ranked, Subjects: subjects}, nil
}

func (s *Service) RecordUse(ctx context.Context, req services.RecordMemoryUseRequest) error {
	if len(req.IDs) == 0 {
		return nil
	}

	return s.repo.MarkUsed(ctx, repositories.MarkAgentMemoriesUsedRequest{
		TenantInfo: req.TenantInfo,
		IDs:        req.IDs,
		At:         timeutils.NowUnix(),
	})
}

func (s *Service) Usage(
	ctx context.Context,
	tenant pagination.TenantInfo,
) (*services.AgentMemoryUsage, error) {
	count, err := s.repo.CountActive(ctx, repositories.CountActiveAgentMemoriesRequest{
		TenantInfo: tenant,
		Now:        timeutils.NowUnix(),
	})
	if err != nil {
		return nil, err
	}

	return &services.AgentMemoryUsage{
		ActiveCount:   count,
		ActiveSoftCap: agent.MemoryActiveSoftCap,
		WarnAt:        agent.MemoryActiveWarnAt,
	}, nil
}

// RecordCorrection keeps what a decision taught. A change to the proposed
// parameters is always worth keeping, with the reason when a person typed
// one; a rejection is worth keeping only when a person said why, because
// "rejected" alone teaches the next run nothing it can act on.
func (s *Service) RecordCorrection(
	ctx context.Context,
	proposal *agent.AgentProposal,
	decision *agent.AgentDecision,
) (*agent.Memory, error) {
	if proposal == nil || decision == nil {
		return nil, nil
	}

	content := correctionContent(proposal, decision)
	if content == "" {
		return nil, nil
	}

	entity := &agent.Memory{
		OrganizationID: proposal.OrganizationID,
		BusinessUnitID: proposal.BusinessUnitID,
		Kind:           agent.MemoryKindCorrection,
		Source:         agent.MemorySourceDecision,
		Status:         agent.MemoryStatusActive,
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
	entity *agent.Memory,
) error {
	if entity.SubjectType == "" || entity.SubjectID == nil || s.labeler == nil {
		return nil
	}

	label, err := s.labeler.Label(ctx, tenant, repositories.MemorySubjectRef{
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
	entity *agent.Memory,
	actor *services.RequestActor,
	op permission.Operation,
	comment string,
) {
	s.logChange(entity, nil, actor, op, comment)
}

func (s *Service) logChange(
	entity *agent.Memory,
	previous map[string]any,
	actor *services.RequestActor,
	op permission.Operation,
	comment string,
) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&services.LogActionParams{
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
	entity *agent.Memory,
) repositories.FindActiveAgentMemoryRequest {
	scope := entity.Scope
	if scope == "" {
		scope = agent.MemoryScopeOrganization
	}
	agentID := pulid.Nil
	if entity.AgentDefinitionID != nil {
		agentID = *entity.AgentDefinitionID
	}

	return repositories.FindActiveAgentMemoryRequest{
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

func subjectRefOf(entity *agent.Memory) *repositories.MemorySubjectRef {
	if entity.SubjectType == "" || entity.SubjectID == nil {
		return nil
	}

	return &repositories.MemorySubjectRef{Type: entity.SubjectType, ID: *entity.SubjectID}
}

var (
	// reasonCodePattern matches a machine reason such as
	// approved_from_activity: one token of lower-case letters, digits and
	// underscores. A sentence a person typed has a space or a capital in it.
	reasonCodePattern = regexp.MustCompile(`^[a-z0-9_]+$`)
	// planSuffixPattern is what the plan service appends to a step's reason.
	planSuffixPattern = regexp.MustCompile(`\s*\(plan [^)]+\)$`)
)

// humanReason returns what a person wrote, or nothing when the reason is a
// code the client filled in for them.
func humanReason(reason string) string {
	reason = strings.TrimSpace(planSuffixPattern.ReplaceAllString(reason, ""))
	if reason == "" || reasonCodePattern.MatchString(reason) {
		return ""
	}

	return reason
}

// correctionContent writes the lesson in one or two sentences, or nothing
// when the decision carries none.
func correctionContent(proposal *agent.AgentProposal, decision *agent.AgentDecision) string {
	reason := humanReason(decision.ReasonCode)

	switch agent.OutcomeOfDecision(decision.Decision, decision.Modifications) {
	case agent.TrustOutcomeModified:
		changes := describeChanges(proposal.ToolParams, decision.Modifications)
		if len(changes) == 0 {
			return ""
		}
		content := fmt.Sprintf("When %s was proposed, a person changed %s.",
			proposal.ToolName, joinNatural(changes))
		if reason != "" {
			content += " Reason: " + reason
		}

		return stringutils.Ellipsize(content, agent.MaxMemoryContentChars)
	case agent.TrustOutcomeRejected:
		if reason == "" {
			return ""
		}

		return stringutils.Ellipsize(
			fmt.Sprintf("A person rejected %s. Reason: %s", proposal.ToolName, reason),
			agent.MaxMemoryContentChars,
		)
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
