package agentrunservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
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

	// The id is known before the record is written, because the workflow
	// that carries the run starts first and is told which run it is.
	run.ID = pulid.MustNew("ar_")
	run.WorkflowID = workflowIDFor(definition, run, req.Slot)

	payload := &agentjobs.AgentRunPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: req.TenantInfo.OrgID,
			BusinessUnitID: req.TenantInfo.BuID,
			UserID:         actor.UserIDOrNil(),
			Timestamp:      timeutils.NowUnix(),
		},
		RunID:        run.ID,
		DefinitionID: definition.ID,
		Trigger:      trigger,
		SubjectType:  subjectType,
		SubjectID:    subjectID,
		EventKind:    req.EventKind,
	}

	// The workflow starts before the run is recorded, because its id is what
	// keeps a subject to one open run: a second event about a subject whose
	// run is still open is refused by Temporal, where two requests cannot
	// both pass the check. Recording first would leave a record behind for
	// every duplicate refused. The run's first activity waits out the moment
	// between the start and the record.
	if _, err = s.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       run.WorkflowID,
		TaskQueue:                temporaltype.TaskQueueAgentBackground.String(),
		WorkflowIDReusePolicy:    reusePolicyFor(run, req.Slot),
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_FAIL,
		// Without this the SDK hands back the open run as if it had just
		// started, and the duplicate would be recorded against it.
		WorkflowExecutionErrorWhenAlreadyStarted: true,
		StaticSummary:                            definition.Name,
		Priority: temporal.Priority{
			PriorityKey: priorityFor(trigger),
			FairnessKey: req.TenantInfo.OrgID.String(),
		},
	}, agentjobs.AgentRunWorkflowName, payload); err != nil {
		var started *serviceerror.WorkflowExecutionAlreadyStarted
		if errors.As(err, &started) {
			return nil, services.ErrAgentRunAlreadyOpen
		}

		return nil, err
	}

	updated, err := s.repo.Create(ctx, run)
	if err != nil {
		// Nothing can run without its record. The workflow is stopped so it
		// does not fail on a record that will never exist.
		if cErr := s.workflows.CancelWorkflow(ctx, run.WorkflowID, ""); cErr != nil {
			s.l.Error("a run's workflow started without its record and could not be stopped",
				zap.String("workflow", run.WorkflowID),
				zap.Error(cErr),
			)
		}

		return nil, err
	}

	s.logStart(
		updated,
		actor,
		fmt.Sprintf("Run of agent %s started (%s)", definition.Name, trigger),
	)
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

// workflowIDFor names a run's workflow by what must not run twice at once:
// the schedule slot a scheduled run fills, or the subject an event run is
// about. Any other run is its own.
func workflowIDFor(definition *agentdefinition.Definition, run *agent.AgentRun, slot int64) string {
	switch {
	case slot > 0:
		return fmt.Sprintf("%s%s-%d", workflowIDPrefix, definition.ID, slot)
	case run.Trigger == agent.RunTriggerEvent && run.SubjectID.IsNotNil():
		return fmt.Sprintf("%s%s-subject-%s", workflowIDPrefix, definition.ID, run.SubjectID)
	default:
		return workflowIDPrefix + run.ID.String()
	}
}

// reusePolicyFor says whether a run's workflow id may be used again once the
// run is over. A slot is filled once, ever; a subject may have a new run
// whenever its last one has finished.
func reusePolicyFor(run *agent.AgentRun, slot int64) enums.WorkflowIdReusePolicy {
	if slot == 0 && run.Trigger == agent.RunTriggerEvent && run.SubjectID.IsNotNil() {
		return enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE
	}

	return enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE
}

// priorityFor orders a run against the queue's other work: one a person
// started is waited on, one a schedule or an event started is not.
func priorityFor(trigger agent.RunTrigger) int {
	switch trigger {
	case agent.RunTriggerScheduled, agent.RunTriggerContinuous, agent.RunTriggerEvent:
		return agentflow.PriorityBackground
	default:
		return agentflow.PriorityOneShot
	}
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
