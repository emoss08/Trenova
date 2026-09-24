package agentevalcaseservice

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/aifeedback"
	"github.com/emoss08/trenova/internal/core/domain/conversation"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const promptVersion = "agent-definition/v2"

type caseStore interface {
	Create(ctx context.Context, entity *agentquality.EvalCase) (*agentquality.EvalCase, error)
	Update(ctx context.Context, entity *agentquality.EvalCase) (*agentquality.EvalCase, error)
	GetByID(
		ctx context.Context,
		req repositories.GetAgentEvalCaseByIDRequest,
	) (*agentquality.EvalCase, error)
	GetByContent(
		ctx context.Context,
		req repositories.GetAgentEvalCaseByContentRequest,
	) (*agentquality.EvalCase, error)
	GetByProposal(
		ctx context.Context,
		req repositories.GetAgentEvalCaseByProposalRequest,
	) (*agentquality.EvalCase, error)
	ListConnection(
		ctx context.Context,
		req *repositories.ListAgentEvalCaseConnectionRequest,
	) (*pagination.CursorListResult[*agentquality.EvalCase], error)
	ListCaptureCandidates(
		ctx context.Context,
		req repositories.ListEvalCaseCaptureCandidatesRequest,
	) ([]repositories.EvalCaseCaptureCandidate, error)
	PurgeExpired(ctx context.Context, req repositories.PurgeExpiredEvalCasesRequest) (int, error)
	PurgeOrphaned(ctx context.Context, req repositories.PurgeOrphanedEvalCasesRequest) (int, error)
}

type proposalReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentProposalByIDRequest,
	) (*agent.AgentProposal, error)
}

type decisionReader interface {
	ListByProposals(
		ctx context.Context,
		req repositories.ListAgentDecisionsByProposalsRequest,
	) ([]*agent.AgentDecision, error)
}

type runReader interface {
	GetByID(ctx context.Context, req repositories.GetAgentRunByIDRequest) (*agent.AgentRun, error)
}

type definitionReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetAgentDefinitionByIDRequest,
	) (*agentdefinition.Definition, error)
}

type conversationReader interface {
	GetThreadOwned(
		ctx context.Context,
		req repositories.GetThreadOwnedRequest,
	) (*conversation.Thread, error)
	ListMessages(
		ctx context.Context,
		req repositories.ListMessagesRequest,
	) ([]conversation.Message, error)
}

type feedbackStore interface {
	ListByIDs(
		ctx context.Context,
		req repositories.ListAIFeedbackByIDsRequest,
	) ([]*aifeedback.Feedback, error)
	LinkEvalCase(ctx context.Context, req repositories.LinkAIFeedbackEvalCaseRequest) error
}

type retentionReader interface {
	List(ctx context.Context) (*pagination.ListResult[*tenant.DataRetention], error)
}

type Params struct {
	fx.In

	Logger        *zap.Logger
	Cases         repositories.AgentEvalCaseRepository
	Proposals     repositories.AgentProposalRepository
	Decisions     repositories.AgentDecisionRepository
	Runs          repositories.AgentRunRepository
	Definitions   repositories.AgentDefinitionRepository
	Conversations repositories.ConversationRepository
	Feedback      repositories.AIFeedbackRepository
	Retention     repositories.DataRetentionRepository
	Registry      *permission.Registry
	QueryTools    services.AgentQueryToolRegistry
	ActionTools   services.AgentToolRegistry
	Subjects      services.AgentSubjectDescriber `optional:"true"`
	AuditService  services.AuditService
}

type Service struct {
	l             *zap.Logger
	cases         caseStore
	proposals     proposalReader
	decisions     decisionReader
	runs          runReader
	definitions   definitionReader
	conversations conversationReader
	feedback      feedbackStore
	retention     retentionReader
	redactor      *Redactor
	subjects      services.AgentSubjectDescriber
	audit         services.AuditService
	now           func() int64
}

func New(p Params) services.AgentEvalCaseService {
	return &Service{
		l:             p.Logger.Named("service.agentevalcase"),
		cases:         p.Cases,
		proposals:     p.Proposals,
		decisions:     p.Decisions,
		runs:          p.Runs,
		definitions:   p.Definitions,
		conversations: p.Conversations,
		feedback:      p.Feedback,
		retention:     p.Retention,
		redactor:      NewRedactor(p.Registry, p.QueryTools, p.ActionTools),
		subjects:      p.Subjects,
		audit:         p.AuditService,
		now:           timeutils.NowUnix,
	}
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentEvalCaseByIDRequest,
) (*agentquality.EvalCase, error) {
	return s.cases.GetByID(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentEvalCaseConnectionRequest,
) (*pagination.CursorListResult[*agentquality.EvalCase], error) {
	return s.cases.ListConnection(ctx, req)
}

func (s *Service) CreateCurated(
	ctx context.Context,
	req *services.CreateCuratedEvalCaseRequest,
	actor *services.RequestActor,
) (*services.EvalCaseCapture, error) {
	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         req.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	trigger := req.Trigger
	if trigger == "" {
		trigger = agent.RunTriggerChat
	}
	heldTools := req.HeldTools
	if len(heldTools) == 0 {
		heldTools = definition.EffectiveToolNames()
	}

	evalCase := &agentquality.EvalCase{
		OrganizationID:      req.TenantInfo.OrgID,
		BusinessUnitID:      req.TenantInfo.BuID,
		AgentDefinitionID:   definition.ID,
		Title:               strings.TrimSpace(req.Title),
		Source:              agentquality.CaseSourceCurated,
		Status:              agentquality.CaseStatusCandidate,
		Trigger:             trigger,
		Input:               req.Input,
		PageContext:         req.PageContext,
		Mentions:            req.Mentions,
		SubjectType:         req.SubjectType,
		SubjectID:           req.SubjectID,
		HeldTools:           heldTools,
		Expected:            req.Expected,
		Rubric:              strings.TrimSpace(req.Rubric),
		CapturedFingerprint: agentquality.FingerprintOf(definition, promptVersion),
		ExpiresAt:           req.ExpiresAt,
	}

	return s.save(ctx, evalCase, actor)
}

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateEvalCaseRequest,
	actor *services.RequestActor,
) (*agentquality.EvalCase, error) {
	evalCase, err := s.cases.GetByID(ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if evalCase.Version != req.Version {
		return nil, dberror.CreateVersionMismatchError("AgentEvalCase", evalCase.ID.String())
	}

	previous := *evalCase
	if req.Title != nil {
		evalCase.Title = strings.TrimSpace(*req.Title)
	}
	if req.Input != nil {
		evalCase.Input = *req.Input
	}
	if req.HeldTools != nil {
		evalCase.HeldTools = req.HeldTools
	}
	if req.Expected != nil {
		evalCase.Expected = *req.Expected
	}
	if req.Rubric != nil {
		evalCase.Rubric = strings.TrimSpace(*req.Rubric)
	}
	switch {
	case req.ClearExpiry:
		evalCase.ExpiresAt = nil
	case req.ExpiresAt != nil:
		evalCase.ExpiresAt = req.ExpiresAt
	}

	if err = s.prepare(evalCase); err != nil {
		return nil, err
	}

	updated, err := s.cases.Update(ctx, evalCase)
	if err != nil {
		return nil, duplicateOr(err)
	}

	s.log(updated, &previous, actor, permission.OpUpdate, "Evaluation case updated")

	return updated, nil
}

func (s *Service) SetStatus(
	ctx context.Context,
	req *services.SetEvalCaseStatusRequest,
	actor *services.RequestActor,
) (*agentquality.EvalCase, error) {
	if !req.Status.IsValid() {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			"Status is not recognised",
		)
	}

	evalCase, err := s.cases.GetByID(ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         req.ID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if evalCase.Status == req.Status {
		return evalCase, nil
	}
	if !evalCase.Status.CanBecome(req.Status) {
		return nil, errortypes.NewValidationError(
			"status",
			errortypes.ErrInvalid,
			fmt.Sprintf("A %s case cannot become %s", evalCase.Status, req.Status),
		)
	}

	previous := *evalCase
	evalCase.Status = req.Status
	if err = s.prepare(evalCase); err != nil {
		return nil, err
	}

	updated, err := s.cases.Update(ctx, evalCase)
	if err != nil {
		return nil, err
	}

	s.log(updated, &previous, actor, permission.OpUpdate,
		fmt.Sprintf("Evaluation case moved from %s to %s", previous.Status, updated.Status))

	return updated, nil
}

func (s *Service) prepare(evalCase *agentquality.EvalCase) error {
	evalCase.Input = strings.TrimSpace(evalCase.Input)
	evalCase.Expected.Normalize()
	evalCase.HeldTools = distinctTools(evalCase.HeldTools)

	hash, err := evalCase.ComputeContentHash()
	if err != nil {
		return fmt.Errorf("hash evaluation case: %w", err)
	}
	evalCase.ContentHash = hash

	multiErr := errortypes.NewMultiError()
	evalCase.Validate(multiErr)
	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

func (s *Service) save(
	ctx context.Context,
	evalCase *agentquality.EvalCase,
	actor *services.RequestActor,
) (*services.EvalCaseCapture, error) {
	if actor != nil && actor.IsUser() && actor.UserID.IsNotNil() {
		userID := actor.UserID
		evalCase.CreatedByUserID = &userID
	}
	if err := s.prepare(evalCase); err != nil {
		return nil, err
	}

	tenantInfo := pagination.TenantInfo{
		OrgID: evalCase.OrganizationID,
		BuID:  evalCase.BusinessUnitID,
	}
	if existing, found, err := s.byContent(ctx, evalCase, tenantInfo); err != nil || found {
		return &services.EvalCaseCapture{Case: existing, Duplicate: found}, err
	}

	created, err := s.cases.Create(ctx, evalCase)
	if err != nil {
		if !dberror.IsUniqueConstraintViolation(err) {
			return nil, err
		}
		existing, found, lookupErr := s.byContent(ctx, evalCase, tenantInfo)
		if lookupErr != nil {
			return nil, lookupErr
		}
		if !found && evalCase.SourceProposalID != nil {
			existing, lookupErr = s.cases.GetByProposal(
				ctx,
				repositories.GetAgentEvalCaseByProposalRequest{
					ProposalID: *evalCase.SourceProposalID,
					TenantInfo: tenantInfo,
				},
			)
			if lookupErr != nil {
				return nil, lookupErr
			}
			found = true
		}
		if !found {
			return nil, err
		}

		return &services.EvalCaseCapture{Case: existing, Duplicate: true}, nil
	}

	s.log(created, nil, actor, permission.OpCreate,
		fmt.Sprintf("Evaluation case captured from %s", created.Source))

	return &services.EvalCaseCapture{Case: created}, nil
}

func (s *Service) byContent(
	ctx context.Context,
	evalCase *agentquality.EvalCase,
	tenantInfo pagination.TenantInfo,
) (*agentquality.EvalCase, bool, error) {
	existing, err := s.cases.GetByContent(ctx, repositories.GetAgentEvalCaseByContentRequest{
		AgentDefinitionID: evalCase.AgentDefinitionID,
		ContentHash:       evalCase.ContentHash,
		TenantInfo:        tenantInfo,
	})
	switch {
	case err == nil:
		return existing, true, nil
	case isNotFound(err):
		return nil, false, nil
	default:
		return nil, false, err
	}
}

func (s *Service) log(
	current *agentquality.EvalCase,
	previous *agentquality.EvalCase,
	actor *services.RequestActor,
	operation permission.Operation,
	comment string,
) {
	if s.audit == nil {
		return
	}

	auditActor := actor.AuditActorOrSystem()
	params := &services.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     current.ID.String(),
		Operation:      operation,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(auditSummary(current)),
		OrganizationID: current.OrganizationID,
		BusinessUnitID: current.BusinessUnitID,
	}
	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(auditSummary(previous))
	}
	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log evaluation case audit", zap.Error(err))
	}
}

type caseAudit struct {
	ID                pulid.ID                `json:"id"`
	AgentDefinitionID pulid.ID                `json:"agentDefinitionId"`
	Title             string                  `json:"title"`
	Source            agentquality.CaseSource `json:"source"`
	Status            agentquality.CaseStatus `json:"status"`
	Expected          agentquality.Expected   `json:"expected"`
	HeldTools         []string                `json:"heldTools"`
	Rubric            string                  `json:"rubric"`
	ExpiresAt         *int64                  `json:"expiresAt"`
	Version           int64                   `json:"version"`
}

func auditSummary(evalCase *agentquality.EvalCase) caseAudit {
	return caseAudit{
		ID:                evalCase.ID,
		AgentDefinitionID: evalCase.AgentDefinitionID,
		Title:             evalCase.Title,
		Source:            evalCase.Source,
		Status:            evalCase.Status,
		Expected:          evalCase.Expected,
		HeldTools:         evalCase.HeldTools,
		Rubric:            evalCase.Rubric,
		ExpiresAt:         evalCase.ExpiresAt,
		Version:           evalCase.Version,
	}
}

func duplicateOr(err error) error {
	if dberror.IsUniqueConstraintViolation(err) {
		return errortypes.NewValidationError(
			"input",
			errortypes.ErrDuplicate,
			"Another case of this agent already asks exactly this",
		)
	}

	return err
}

func distinctTools(tools []string) []string {
	out := make([]string, 0, len(tools))
	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		name := strings.TrimSpace(tool)
		if name == "" {
			continue
		}
		if _, dup := seen[name]; dup {
			continue
		}
		seen[name] = struct{}{}
		out = append(out, name)
	}

	return out
}

func isNotFound(err error) bool {
	return errortypes.IsNotFoundError(err) || dberror.IsNotFoundError(err)
}
