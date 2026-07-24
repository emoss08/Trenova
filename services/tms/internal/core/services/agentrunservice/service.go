package agentrunservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	agentPromptVersion = "billing-exception-v1"
	provisionalHash    = "pending"
	workflowIDPrefix   = "billing-exception-agent-"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentRunRepository
	Control      services.AgentControlService
	Workflows    services.WorkflowStarter
	AuditService services.AuditService
}

type Service struct {
	l         *zap.Logger
	repo      repositories.AgentRunRepository
	control   services.AgentControlService
	workflows services.WorkflowStarter
	audit     services.AuditService
}

func New(p Params) services.AgentRunService {
	return &Service{
		l:         p.Logger.Named("service.agentrun"),
		repo:      p.Repo,
		control:   p.Control,
		workflows: p.Workflows,
		audit:     p.AuditService,
	}
}

func (s *Service) Start(
	ctx context.Context,
	req *services.StartAgentRunRequest,
	actor *services.RequestActor,
) (*agent.AgentRun, error) {
	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError("The workflow engine is not available")
	}

	control, err := s.control.Get(ctx, req.TenantInfo)
	if err != nil {
		return nil, err
	}

	run := &agent.AgentRun{
		OrganizationID:   req.TenantInfo.OrgID,
		BusinessUnitID:   req.TenantInfo.BuID,
		AgentType:        req.AgentType,
		SubjectType:      req.SubjectType,
		SubjectID:        req.SubjectID,
		Status:           agent.RunStatusPending,
		PromptVersion:    agentPromptVersion,
		InputContextHash: provisionalHash,
	}

	me := errortypes.NewMultiError()
	run.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	created, err := s.repo.Create(ctx, run)
	if err != nil {
		return nil, err
	}

	workflowID := workflowIDPrefix + created.ID.String()
	payload := &agentjobs.AgentRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         actor.UserIDOrNil(),
			Timestamp:      timeutils.NowUnix(),
		},
		RunID:                  created.ID,
		SubjectType:            req.SubjectType,
		SubjectID:              req.SubjectID,
		PromptVersion:          agentPromptVersion,
		ShadowMode:             control.ShadowMode,
		DecisionTimeoutSeconds: control.DecisionTimeoutSeconds,
	}

	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: temporaltype.TaskQueueBilling.String(),
	}, agentjobs.BillingExceptionAgentWorkflowName, payload); err != nil {
		created.Status = agent.RunStatusFailed
		created.ErrorMessage = err.Error()
		if _, updateErr := s.repo.Update(ctx, created); updateErr != nil {
			s.l.Error("failed to mark agent run failed", zap.Error(updateErr))
		}
		return nil, err
	}

	created.WorkflowID = workflowID
	updated, err := s.repo.Update(ctx, created)
	if err != nil {
		return nil, err
	}

	auditActor := actor.AuditActorOrSystem()
	if err = s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentRun,
		ResourceID:     updated.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(updated),
		OrganizationID: updated.OrganizationID,
		BusinessUnitID: updated.BusinessUnitID,
	}, auditservice.WithComment("Billing exception agent run started")); err != nil {
		s.l.Error("failed to log agent run audit", zap.Error(err))
	}

	return updated, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return s.repo.GetByID(ctx, req)
}
