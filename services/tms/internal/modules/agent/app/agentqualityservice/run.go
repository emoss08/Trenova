package agentqualityservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityengine"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentqualityports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/workspace/app/auditservice"
	"github.com/emoss08/trenova/internal/modules/workspace/ports/auditports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/jsonutils"
	"go.uber.org/zap"
)

func (s *Service) RunSuite(
	ctx context.Context,
	req *agentqualityports.RunAgentSuiteRequest,
	actor *permissionports.RequestActor,
) (*agentqualitysuite.SuiteRun, error) {
	if s.starter == nil {
		return nil, errortypes.NewBusinessError("The workflow engine is not available")
	}

	plan, err := s.suites.PlanSweep(ctx, &agentqualityengine.SweepRequest{
		TenantInfo:        req.TenantInfo,
		AgentDefinitionID: req.AgentDefinitionID,
		Forced:            true,
	})
	if err != nil {
		return nil, err
	}
	if len(plan.Agents) == 0 {
		return nil, errortypes.NewNotFoundError("Agent not found within your organization")
	}

	planned := plan.Agents[0]
	if planned.ActiveCases == 0 {
		return nil, errortypes.NewBusinessError(
			"This agent has no active evaluation cases to run; activate a case first",
		)
	}

	opened, err := s.suites.OpenSuite(ctx, &agentqualityengine.OpenSuiteRequest{
		TenantInfo:        req.TenantInfo,
		Agent:             planned,
		Trigger:           agentqualitysuite.SuiteRunTriggerManual,
		RequestedByUserID: actor.UserIDOrNil(),
		Settings:          plan.Settings,
	})
	if err != nil {
		return nil, err
	}

	run, err := s.suiteRuns.GetByID(ctx, agentqualityports.GetAgentSuiteRunRequest{
		ID:         opened.SuiteRunID,
		TenantInfo: req.TenantInfo,
	})
	if err != nil {
		return nil, err
	}

	if _, err = s.starter.StartSuiteRun(ctx, &agentqualityports.AgentSuiteRunStart{
		TenantInfo:        req.TenantInfo,
		UserID:            actor.UserIDOrNil(),
		SuiteRunID:        run.ID,
		AgentDefinitionID: run.AgentDefinitionID,
		SampleSeed:        run.SampleSeed,
		Settings:          plan.Settings,
		DayStart:          plan.DayStart,
		MonthStart:        plan.MonthStart,
	}); err != nil {
		if failErr := s.suites.FailSuite(ctx, &agentqualityengine.FailSuiteRequest{
			TenantInfo: req.TenantInfo,
			SuiteRunID: run.ID,
			Error:      "The suite could not be started: " + err.Error(),
		}); failErr != nil {
			s.l.Error("could not close a suite run that failed to start", zap.Error(failErr))
		}

		return nil, err
	}

	s.logRun(run, actor)

	return run, nil
}

func (s *Service) logRun(run *agentqualitysuite.SuiteRun, actor *permissionports.RequestActor) {
	auditActor := actor.AuditActorOrSystem()
	if err := s.audit.LogAction(&auditports.LogActionParams{
		Resource:       permission.ResourceAgentEvalSuite,
		ResourceID:     run.ID.String(),
		Operation:      permission.OpCreate,
		UserID:         auditActor.UserID,
		PrincipalType:  auditActor.PrincipalType,
		PrincipalID:    auditActor.PrincipalID,
		APIKeyID:       auditActor.APIKeyID,
		CurrentState:   jsonutils.MustToJSON(run),
		OrganizationID: run.OrganizationID,
		BusinessUnitID: run.BusinessUnitID,
	}, auditservice.WithComment("Agent suite run started by hand")); err != nil {
		s.l.Error("failed to log agent suite run audit", zap.Error(err))
	}
}
