// Package agentevaluationservice replays recorded runs against their agent
// as it is now, so a change to an agent's instructions, tools or model is
// checked against runs that already happened rather than the next live one.
package agentevaluationservice

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/auditservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/internal/infrastructure/observability/aitrace"
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

type Params struct {
	fx.In

	Logger       *zap.Logger
	Repo         repositories.AgentEvaluationRepository
	Runs         repositories.AgentRunRepository
	Definitions  repositories.AgentDefinitionRepository
	Cases        repositories.AgentEvalCaseRepository
	Workflows    services.WorkflowStarter
	AuditService services.AuditService
}

type Service struct {
	l           *zap.Logger
	repo        repositories.AgentEvaluationRepository
	runs        repositories.AgentRunRepository
	definitions repositories.AgentDefinitionRepository
	cases       repositories.AgentEvalCaseRepository
	workflows   services.WorkflowStarter
	audit       services.AuditService
}

func New(p Params) services.AgentEvaluationService {
	return &Service{
		l:           p.Logger.Named("service.agentevaluation"),
		repo:        p.Repo,
		runs:        p.Runs,
		definitions: p.Definitions,
		cases:       p.Cases,
		workflows:   p.Workflows,
		audit:       p.AuditService,
	}
}

// Replay opens an evaluation for a recorded run and hands it to the
// workflow that carries it out.
func (s *Service) Replay(
	ctx context.Context,
	req *services.ReplayAgentRunRequest,
	actor *services.RequestActor,
) (*agent.Evaluation, error) {
	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError("The workflow engine is not available")
	}

	run, err := s.runs.GetByID(ctx, repositories.GetAgentRunByIDRequest{
		ID:         req.RunID,
		TenantInfo: &req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if run.AgentDefinitionID.IsNil() {
		return nil, errortypes.NewBusinessError(
			"This run was not made by a configured agent, so there is nothing to replay it against",
		)
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         run.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	evaluation := &agent.Evaluation{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		SourceRunID:       run.ID,
		Status:            agent.EvaluationStatusPending,
		Trigger:           run.Trigger,
		SubjectType:       run.SubjectType,
		SubjectID:         run.SubjectID,
		DefinitionVersion: definition.Version,
	}

	return s.start(ctx, evaluation, actor,
		fmt.Sprintf("Replay of run %s of agent %s started", run.ID, definition.Name))
}

func (s *Service) ReplayCase(
	ctx context.Context,
	req *services.ReplayAgentEvalCaseRequest,
	actor *services.RequestActor,
) (*agent.Evaluation, error) {
	if !s.workflows.Enabled() {
		return nil, errortypes.NewBusinessError("The workflow engine is not available")
	}

	evalCase, err := s.cases.GetByID(ctx, repositories.GetAgentEvalCaseByIDRequest{
		ID:         req.CaseID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}
	if evalCase.Status == agentquality.CaseStatusRetired {
		return nil, errortypes.NewBusinessError("A retired case is not replayed; restore it first")
	}

	definition, err := s.definitions.GetByID(ctx, repositories.GetAgentDefinitionByIDRequest{
		ID:         evalCase.AgentDefinitionID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	caseID := evalCase.ID
	evaluation := &agent.Evaluation{
		OrganizationID:    req.TenantInfo.OrgID,
		BusinessUnitID:    req.TenantInfo.BuID,
		AgentDefinitionID: definition.ID,
		EvalCaseID:        &caseID,
		Status:            agent.EvaluationStatusPending,
		Trigger:           evalCase.Trigger,
		SubjectType:       evalCase.SubjectType,
		SubjectID:         evalCase.SubjectID,
		DefinitionVersion: definition.Version,
	}

	return s.start(ctx, evaluation, actor, fmt.Sprintf(
		"Replay of evaluation case %s of agent %s started",
		evalCase.ID,
		definition.Name,
	))
}

func (s *Service) start(
	ctx context.Context,
	evaluation *agent.Evaluation,
	actor *services.RequestActor,
	comment string,
) (*agent.Evaluation, error) {
	if actor != nil && actor.IsUser() && actor.UserID.IsNotNil() {
		userID := actor.UserID
		evaluation.RequestedByUserID = &userID
	}

	me := errortypes.NewMultiError()
	evaluation.Validate(me)
	if me.HasErrors() {
		return nil, me
	}

	created, err := s.repo.Create(ctx, evaluation)
	if err != nil {
		return nil, err
	}

	workflowID := "agent-evaluation-" + created.ID.String()
	payload := &agentjobs.AgentEvaluationPayload{
		BasePayload: temporaltype.BasePayload{
			OrganizationID: evaluation.OrganizationID,
			BusinessUnitID: evaluation.BusinessUnitID,
			UserID:         actor.UserIDOrNil(),
			Timestamp:      timeutils.NowUnix(),
		},
		EvaluationID: created.ID,
	}
	anchor := aitrace.AnchorFor(aitrace.AnchorEvaluation, created.ID.String())
	startCtx := aitrace.ContextWithAnchor(ctx, anchor)
	if _, err = s.workflows.StartWorkflow(startCtx, client.StartWorkflowOptions{
		ID:                    workflowID,
		TaskQueue:             temporaltype.TaskQueueAgentHeavy.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
	}, agentjobs.AgentEvaluationWorkflowName, payload); err != nil {
		completedAt := timeutils.NowUnix()
		created.Status = agent.EvaluationStatusFailed
		created.ErrorMessage = err.Error()
		created.CompletedAt = &completedAt
		if _, uErr := s.repo.Update(ctx, created); uErr != nil {
			s.l.Error("failed to mark agent evaluation failed", zap.Error(uErr))
		}

		return nil, err
	}

	created.WorkflowID = workflowID
	updated, err := s.repo.Update(ctx, created)
	if err != nil {
		return nil, err
	}

	s.log(updated, actor, comment)

	return updated, nil
}

func (s *Service) GetByID(
	ctx context.Context,
	req repositories.GetAgentEvaluationByIDRequest,
) (*agent.Evaluation, error) {
	return s.repo.GetByID(ctx, req)
}

func (s *Service) ListConnection(
	ctx context.Context,
	req *repositories.ListAgentEvaluationConnectionRequest,
) (*pagination.CursorListResult[*agent.Evaluation], error) {
	return s.repo.ListConnection(ctx, req)
}

func (s *Service) log(evaluation *agent.Evaluation, actor *services.RequestActor, comment string) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&services.LogActionParams{
		Resource:       permission.ResourceAgentRun,
		ResourceID:     evaluation.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(evaluation),
		OrganizationID: evaluation.OrganizationID,
		BusinessUnitID: evaluation.BusinessUnitID,
	}, auditservice.WithComment(comment)); err != nil {
		s.l.Error("failed to log agent evaluation audit", zap.Error(err))
	}
}
