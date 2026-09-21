package agentcontrolservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
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

const SystemKeyBillingException = "billing_exception"

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentControlRepository
	Definitions  repositories.AgentDefinitionRepository
	AuditService services.AuditService
}

type Service struct {
	l           *zap.Logger
	repo        repositories.AgentControlRepository
	definitions repositories.AgentDefinitionRepository
	audit       services.AuditService
}

func New(p Params) services.AgentControlService {
	return &Service{
		l:           p.Logger.Named("service.agentcontrol"),
		repo:        p.Repo,
		definitions: p.Definitions,
		audit:       p.AuditService,
	}
}

func (s *Service) Get(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*tenant.AgentControl, error) {
	control, err := s.repo.GetOrCreate(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}

	definition, err := s.billingDefinition(ctx, tenantInfo)
	if err != nil {
		return nil, err
	}
	applyLegacyFields(control, definition)

	return control, nil
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
	if req.EarnedAutonomy != nil {
		control.EarnedAutonomy = *req.EarnedAutonomy
	}
	if req.PromotionThreshold != nil {
		control.PromotionThreshold = *req.PromotionThreshold
	}

	me := errortypes.NewMultiError()
	control.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	updated, err := s.repo.Update(ctx, control)
	if err != nil {
		return nil, err
	}

	definition, err := s.applyLegacyUpdate(ctx, req)
	if err != nil {
		return nil, err
	}
	applyLegacyFields(updated, definition)

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

func (s *Service) billingDefinition(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (*agentdefinition.Definition, error) {
	definition, err := s.definitions.GetBySystemKey(
		ctx,
		repositories.GetAgentDefinitionBySystemKeyRequest{
			SystemKey:  SystemKeyBillingException,
			TenantInfo: tenantInfo,
		},
	)
	if err != nil {
		if errortypes.IsNotFoundError(err) {
			return nil, nil
		}

		return nil, err
	}

	return definition, nil
}

func (s *Service) applyLegacyUpdate(
	ctx context.Context,
	req *services.UpdateAgentControlRequest,
) (*agentdefinition.Definition, error) {
	definition, err := s.billingDefinition(ctx, req.TenantInfo)
	if err != nil || definition == nil {
		return definition, err
	}
	if req.BillingAgentEnabled == nil && req.DecisionTimeoutSeconds == nil {
		return definition, nil
	}

	if req.BillingAgentEnabled != nil {
		definition.Enabled = *req.BillingAgentEnabled
	}
	if req.DecisionTimeoutSeconds != nil {
		definition.DecisionTimeoutSeconds = *req.DecisionTimeoutSeconds
	}

	me := errortypes.NewMultiError()
	definition.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	return s.definitions.Update(ctx, definition)
}

func applyLegacyFields(control *tenant.AgentControl, definition *agentdefinition.Definition) {
	if definition == nil {
		control.BillingAgentEnabled = false
		control.DecisionTimeoutSeconds = agentdefinition.DefaultDecisionTimeoutSeconds
		return
	}

	control.BillingAgentEnabled = definition.Enabled
	control.DecisionTimeoutSeconds = definition.DecisionTimeoutSeconds
}
