package agentexceptionservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/watchtowersources"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentexception"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentexceptionports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
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
	Repo         agentexceptionports.AgentExceptionRepository
	AuditService auditports.AuditService
	// Watchtower puts an exception on the feed and takes it off when it is
	// resolved.
	Watchtower watchtowerports.WatchtowerProjector `optional:"true"`
}

type Service struct {
	l          *zap.Logger
	repo       agentexceptionports.AgentExceptionRepository
	audit      auditports.AuditService
	watchtower watchtowerports.WatchtowerProjector
}

func New(p Params) agentexceptionports.AgentExceptionService {
	return &Service{
		l:          p.Logger.Named("service.agentexception"),
		repo:       p.Repo,
		audit:      p.AuditService,
		watchtower: p.Watchtower,
	}
}

func (s *Service) Flag(
	ctx context.Context,
	req *agentexceptionports.FlagAgentExceptionRequest,
	actor *permissionports.RequestActor,
) (*agentexception.AgentException, error) {
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
	req *agentexceptionports.ListAgentExceptionRequest,
) (*pagination.ListResult[*agentexception.AgentException], error) {
	return s.repo.List(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *agentexceptionports.ListAgentExceptionConnectionRequest,
) (*pagination.CursorListResult[*agentexception.AgentException], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req agentexceptionports.GetAgentExceptionByIDRequest,
) (*agentexception.AgentException, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) Resolve(
	ctx context.Context,
	req *agentexceptionports.ResolveAgentExceptionRequest,
	actor *permissionports.RequestActor,
) (*agentexception.AgentException, error) {
	if !req.ResolutionState.IsValid() {
		return nil, errortypes.NewValidationError(
			"resolutionState",
			errortypes.ErrInvalid,
			"Invalid resolution state",
		)
	}

	updated, err := s.repo.UpdateResolution(
		ctx,
		&agentexceptionports.UpdateAgentExceptionResolutionRequest{
			ID:              req.ID,
			ResolutionState: req.ResolutionState,
			ResolutionNotes: req.ResolutionNotes,
			TenantInfo:      req.TenantInfo,
		},
	)
	if err != nil {
		return nil, err
	}

	s.logAction(actor, permission.OpUpdate, updated, nil, "Agent exception resolved")
	if s.watchtower != nil && updated.ResolutionState != agentexception.ResolutionStateOpen {
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
	entity *agentexception.AgentException,
	previous *agentexception.AgentException,
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
