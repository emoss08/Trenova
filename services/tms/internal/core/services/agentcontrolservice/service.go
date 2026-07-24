package agentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
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
	Repo         repositories.AgentControlRepository
	AuditService services.AuditService
}

type Service struct {
	l     *zap.Logger
	repo  repositories.AgentControlRepository
	audit services.AuditService
}

func New(p Params) services.AgentControlService {
	return &Service{
		l:     p.Logger.Named("service.agentcontrol"),
		repo:  p.Repo,
		audit: p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	return s.repo.GetOrCreate(ctx, tenantInfo)
}

func (s *Service) Update(
	ctx context.Context,
	req *services.UpdateAgentControlRequest,
	actor *services.RequestActor,
) (*tenant.AgentControl, error) {
	control, err := s.repo.GetOrCreate(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	previous := *control
	control.ShadowMode = req.ShadowMode
	control.BillingAgentEnabled = req.BillingAgentEnabled
	control.DecisionTimeoutSeconds = req.DecisionTimeoutSeconds

	me := errortypes.NewMultiError()
	control.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	updated, err := s.repo.Update(ctx, control)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActor()
	if err = s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentControl,
		ResourceID:     updated.GetID().String(),
		Operation:      permission.OpUpdate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updated),
		PreviousState:  jsonutils.MustToJSON(&previous),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Agent control updated")); err != nil {
		s.l.Error("failed to log agent control audit", zap.Error(err))
	}

	return updated, nil
}
