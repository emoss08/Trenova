package agentexceptionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/watchtower"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/services/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         agentports.AgentExceptionRepository
	AuditService auditports.AuditService
	// Watchtower puts an exception on the feed and takes it off when it is
	// resolved.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	repo       agentports.AgentExceptionRepository
	audit      auditports.AuditService
	watchtower watchtowerports.WatchtowerProjector
}

func New(p Params) agentports.AgentExceptionService {
	return &Service{
		l:          p.Logger.Named("service.agentexception"),
		repo:       p.Repo,
		audit:      p.AuditService,
		watchtower: p.Watchtower,
	}
}

func (s *Service) Flag(
	ctx context.Context,
	req *agentports.FlagAgentExceptionRequest,
	actor *permissionports.RequestActor,
) (*agent.AgentException, error) {
	entity, err := PlanException(req)
	if err != nil {
		return nil, err
	}

	created, err := s.repo.Create(ctx, entity)
	if err != nil {
		return nil, err
	}

	s.logAction(actor, permission.OpCreate, created, nil, "Agent exception raised")
	if s.watchtower != nil {
		s.watchtower.Upsert(ctx, watchtowersources.DescribeException(created))
	}

	return created, nil
}

func (s *Service) List(
	ctx context.Context,
	req *agentports.ListAgentExceptionRequest,
) (*pagination.ListResult[*agent.AgentException], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentports.ListAgentExceptionConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentException], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req agentports.GetAgentExceptionByIDRequest,
) (*agent.AgentException, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Resolve(
	ctx context.Context,
	req *agentports.ResolveAgentExceptionRequest,
	actor *permissionports.RequestActor,
) (*agent.AgentException, error) {
	if !req.ResolutionState.IsValid() {
		return nil, errortypes.NewValidationError(
			"resolutionState",
			errortypes.ErrInvalid,
			"Invalid resolution state",
		)
	}

	updated, err := s.repo.UpdateResolution(ctx, agentports.UpdateAgentExceptionResolutionRequest{
		ID:              req.ID,
		ResolutionState: req.ResolutionState,
		ResolutionNotes: req.ResolutionNotes,
		TenantInfo:      req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	s.logAction(actor, permission.OpUpdate, updated, nil, "Agent exception resolved")
	if s.watchtower != nil && updated.ResolutionState != agent.ResolutionStateOpen {
		s.watchtower.Resolve(
			ctx,
			req.TenantInfo,
			watchtower.SourceAgentException,
			updated.ID.String(),
		)
	}

	return updated, nil
}

func (s *Service) logAction(
	actor *permissionports.RequestActor,
	op permission.Operation,
	entity *agent.AgentException,
	previous *agent.AgentException,
	comment string,
) {
	auditActor := actor.AuditActorOrSystem()

	params := &auditports.LogActionParams{
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
