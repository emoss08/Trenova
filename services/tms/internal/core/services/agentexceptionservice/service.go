package agentexceptionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentExceptionRepository
	AuditService services.AuditService
}

type Service struct {
	l     *zap.Logger
	repo  repositories.AgentExceptionRepository
	audit services.AuditService
}

func New(p Params) services.AgentExceptionService {
	return &Service{
		l:     p.Logger.Named("service.agentexception"),
		repo:  p.Repo,
		audit: p.AuditService,
	}
}

func (s *Service) Flag(
	ctx context.Context,
	req *services.FlagAgentExceptionRequest,
	actor *services.RequestActor,
) (*agent.AgentException, error) {
	entity := &agent.AgentException{
		OrganizationID:  req.TenantInfo.OrgID,
		BusinessUnitID:  req.TenantInfo.BuID,
		RunID:           req.RunID,
		Category:        req.Category,
		Severity:        req.Severity,
		SubjectType:     req.SubjectType,
		SubjectID:       req.SubjectID,
		AttemptSummary:  req.AttemptSummary,
		Evidence:        req.Evidence,
		BlastRadius:     req.BlastRadius,
		ResolutionState: agent.ResolutionStateOpen,
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

	s.logAction(actor, permission.OpCreate, created, nil, "Agent exception raised")

	return created, nil
}

func (s *Service) List(
	ctx context.Context,
	req *repositories.ListAgentExceptionRequest,
) (*pagination.ListResult[*agent.AgentException], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentExceptionByIDRequest,
) (*agent.AgentException, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Resolve(
	ctx context.Context,
	req *services.ResolveAgentExceptionRequest,
	actor *services.RequestActor,
) (*agent.AgentException, error) {
	if !req.ResolutionState.IsValid() {
		return nil, errortypes.NewValidationError(
			"resolutionState",
			errortypes.ErrInvalid,
			"Invalid resolution state",
		)
	}

	updated, err := s.repo.UpdateResolution(ctx, repositories.UpdateAgentExceptionResolutionRequest{
		ID:              req.ID,
		ResolutionState: req.ResolutionState,
		ResolutionNotes: req.ResolutionNotes,
		TenantInfo:      req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	s.logAction(actor, permission.OpUpdate, updated, nil, "Agent exception resolved")

	return updated, nil
}

func (s *Service) logAction(
	actor *services.RequestActor,
	op permission.Operation,
	entity *agent.AgentException,
	previous *agent.AgentException,
	comment string,
) {
	auditActor := actor.AuditActorOrSystem()

	params := &services.LogActionParams{
		Resource:       permission.ResourceAgentException,
		ResourceID:     entity.GetID().String(),
		Operation:      op,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(entity),
		OrganizationID: entity.OrganizationID,
		BusinessUnitID: entity.BusinessUnitID,
	}

	if previous != nil {
		params.PreviousState = jsonutils.MustToJSON(previous)
	}

	if err := s.audit.LogAction(params, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent exception audit", zap.Error(err))
	}
}
