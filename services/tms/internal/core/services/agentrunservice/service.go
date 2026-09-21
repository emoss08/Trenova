package agentrunservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/client"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	SystemKeyBillingException   = "billing_exception"
	SystemKeyDispatchAssignment = "dispatch_assignment"

	definitionPromptVersion = "agent-definition/v2"
	inlinePromptVersion     = "inline-v1"
	provisionalHash         = "pending"
	workflowIDPrefix        = "agent-run-"
)

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentRunRepository
	Definitions  repositories.AgentDefinitionRepository
	Workflows    services.WorkflowStarter
	Validator    *Validator
	AuditService services.AuditService
	Budgets      services.AgentBudgetService     `optional:"true"`
	Activity     services.AgentActivityPublisher `optional:"true"`
}

type Service struct {
	l           *zap.Logger
	repo        repositories.AgentRunRepository
	definitions repositories.AgentDefinitionRepository
	validator   *Validator
	workflows   services.WorkflowStarter
	audit       services.AuditService
	budgets     services.AgentBudgetService
	activity    services.AgentActivityPublisher
}

func New(p Params) services.AgentRunService {
	return &Service{
		l:           p.Logger.Named("service.agentrun"),
		repo:        p.Repo,
		definitions: p.Definitions,
		validator:   p.Validator,
		workflows:   p.Workflows,
		audit:       p.AuditService,
		budgets:     p.Budgets,
		activity:    p.Activity,
	}
}
func (s *Service) StartForDefinition(
	ctx context.Context,
	req *services.StartAgentRunForDefinitionRequest,
	actor *services.RequestActor,
) (*agent.AgentRun, error) {
	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError("The workflow engine is not available")
	}

	definition, err := s.resolveDefinition(ctx, req)
	if err != nil {
		return nil, err
	}
	if !definition.Enabled {
		return nil, errortypes.NewBusinessError(
			"Agent {0} is disabled and cannot run", definition.Name,
		)
	}

	if err = s.assertWithinBudget(ctx, definition); err != nil {
		return nil, err
	}

	subjectType := req.SubjectType
	subjectID := req.SubjectID
	if subjectType == "" {
		subjectType = agent.SubjectOrganization
		subjectID = req.TenantInfo.OrgID
	}
	trigger := req.Trigger
	if trigger == "" {
		trigger = runTrigger(actor)
	}

	run := &agent.AgentRun{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentType:         agentTypeFor(definition),
		AgentDefinitionID: definition.ID,
		SubjectType:       subjectType,
		SubjectID:         subjectID,
		Status:            agent.RunStatusPending,
		Trigger:           trigger,
		PromptVersion:     definitionPromptVersion,
		InputContextHash:  provisionalHash,
	}

	if multiErr := s.validator.ValidateCreate(ctx, run); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, run)
	if err != nil {
		return nil, err
	}

	workflowID := workflowIDFor(definition, created, req.Slot)
	payload := &agentjobs.AgentRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         actor.UserIDOrNil(),
			Timestamp:      timeutils.NowUnix(),
		},
		RunID:        created.ID,
		DefinitionID: definition.ID,
		Trigger:      trigger,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		EventKind:    req.EventKind,
	}

	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.TaskQueueAgent.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}, agentjobs.AgentRunWorkflowName, payload); err != nil {
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

	s.logStart(updated, actor, fmt.Sprintf("Run of agent %s started (%s)", definition.Name, trigger))
	if s.activity != nil {
		s.activity.RunChanged(ctx, updated, actor.AuditActorOrSystem(), services.ActivityCreated)
	}

	return updated, nil
}

// assertWithinBudget refuses a run past the agent's monthly cost or daily
// run cap. The refusal names the cap, so the scheduler's log and the person
// pressing Run both learn what to change.
func (s *Service) assertWithinBudget(
	ctx context.Context,
	definition *agentdefinition.Definition,
) error {
	if s.budgets == nil {
		return nil
	}

	refusal, err := s.budgets.CheckRun(ctx, definition)
	if err != nil {
		return err
	}
	if refusal.Refused() {
		return errortypes.NewBusinessError(refusal.Message(definition.Name))
	}

	return nil
}

func (s *Service) resolveDefinition(
	ctx context.Context,
	req *services.StartAgentRunForDefinitionRequest,
) (*agentdefinition.Definition, error) {
	if req.DefinitionID.IsNotNil() {
		return s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
			ID:         req.DefinitionID,
			TenantInfo: req.TenantInfo,
		})
	}
	if req.SystemKey != "" {
		return s.definitions.GetBySystemKey(ctx, repositories.GetAgentDefinitionBySystemKeyRequest{
			SystemKey:  req.SystemKey,
			TenantInfo: req.TenantInfo,
		})
	}

	return nil, errortypes.NewValidationError(
		"agentDefinitionId",
		errortypes.ErrRequired,
		"An agent definition id or system key is required",
	)
}
func (s *Service) StartInline(
	ctx context.Context,
	req *services.StartInlineAgentRunRequest,
	actor *services.RequestActor,
) (*agent.AgentRun, error) {
	promptVersion := req.PromptVersion
	if promptVersion == "" {
		promptVersion = inlinePromptVersion
	}

	run := &agent.AgentRun{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentType:         req.AgentType,
		AgentDefinitionID: req.AgentDefinitionID,
		SubjectType:       req.SubjectType,
		SubjectID:         req.SubjectID,
		Status:            agent.RunStatusAwaitingDecision,
		Trigger:           inlineTrigger(req.Trigger, actor),
		PromptVersion:     promptVersion,
		InputContextHash:  provisionalHash,
		StartedAt:         timeutils.NowUnix(),
	}

	if multiErr := s.validator.ValidateCreate(ctx, run); multiErr != nil {
		return nil, multiErr
	}

	created, err := s.repo.Create(ctx, run)
	if err != nil {
		return nil, err
	}

	comment := req.Summary
	if comment == "" {
		comment = "Inline agent run started"
	}
	s.logStart(created, actor, comment)

	return created, nil
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentRunConnectionRequest,
) (*pagination.CursorListResult[*agent.AgentRun], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentRunByIDRequest,
) (*agent.AgentRun, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) logStart(run *agent.AgentRun, actor *services.RequestActor, comment string) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentRun,
		ResourceID:     run.GetID().String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(run),
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent run audit", zap.Error(err))
	}
}

func workflowIDFor(definition *agentdefinition.Definition, run *agent.AgentRun, slot int64) string {
	if slot > 0 {
		return fmt.Sprintf("%s%s-%d", workflowIDPrefix, definition.ID, slot)
	}

	return workflowIDPrefix + run.ID.String()
}

func agentTypeFor(definition *agentdefinition.Definition) agent.Type {
	switch definition.SystemKey {
	case SystemKeyBillingException:
		return agent.TypeBillingException
	case SystemKeyDispatchAssignment:
		return agent.TypeDispatchAssignment
	default:
		return agent.TypeGeneral
	}
}

func inlineTrigger(requested agent.RunTrigger, actor *services.RequestActor) agent.RunTrigger {
	if requested != "" {
		return requested
	}

	return runTrigger(actor)
}

func runTrigger(actor *services.RequestActor) agent.RunTrigger {
	if actor != nil && actor.IsUser() {
		return agent.RunTriggerManual
	}

	return agent.RunTriggerEvent
}
