package agentqualityjobs

import (
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agentquality"
	"github.com/emoss08/trenova/internal/core/services/agentqualityservice"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentflow"
	"github.com/emoss08/trenova/internal/core/temporaljobs/agentjobs"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

var stepRetry = &temporal.RetryPolicy{
	InitialInterval:    time.Second,
	BackoffCoefficient: 2.0,
	MaximumInterval:    30 * time.Second,
	MaximumAttempts:    3,
}

var (
	planOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy:         stepRetry,
	}
	stepOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 2 * time.Minute,
		RetryPolicy:         stepRetry,
	}
	finalizeOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy:         stepRetry,
	}
	judgeOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 3 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	reconcileOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy:         stepRetry,
	}
)

func evaluationPriority(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflow.PriorityEvaluation,
		FairnessKey: orgID.String(),
	}
}

func withPriority(options workflow.ActivityOptions, orgID pulid.ID) workflow.ActivityOptions {
	options.Priority = evaluationPriority(orgID)

	return options
}

func SweepWorkflowID(orgID pulid.ID) string {
	return sweepWorkflowIDPrefix + orgID.String()
}

func childOptions(workflowID string, orgID pulid.ID) workflow.ChildWorkflowOptions {
	return workflow.ChildWorkflowOptions{
		WorkflowID:            workflowID,
		TaskQueue:             temporaltype.TaskQueueAgentHeavy.String(),
		WorkflowIDReusePolicy: enums.WORKFLOW_ID_REUSE_POLICY_REJECT_DUPLICATE,
		ParentClosePolicy:     enums.PARENT_CLOSE_POLICY_ABANDON,
		Priority:              evaluationPriority(orgID),
	}
}

func AgentQualitySweepWorkflow(ctx workflow.Context, payload *SweepPayload) (*SweepResult, error) {
	var a *Activities
	logger := workflow.GetLogger(ctx)
	tenant := payload.tenantInfo()

	result := payload.Result
	if result == nil {
		result = &SweepResult{}
	}

	plan := payload.Plan
	if plan == nil {
		planCtx := workflow.WithActivityOptions(ctx, withPriority(planOptions, tenant.OrgID))
		var planned agentqualityservice.SweepPlan
		if err := workflow.ExecuteActivity(planCtx, a.PlanSweepActivity,
			&agentqualityservice.SweepRequest{
				TenantInfo:        tenant,
				AgentDefinitionID: payload.AgentDefinitionID,
			},
		).Get(planCtx, &planned); err != nil {
			return nil, err
		}
		plan = &planned
		result.Planned = len(plan.Agents)
	}
	if !plan.Enabled {
		return result, nil
	}

	sweepID := workflow.GetInfo(ctx).WorkflowExecution.ID
	stepCtx := workflow.WithActivityOptions(ctx, withPriority(stepOptions, tenant.OrgID))
	handled := 0
	for idx := payload.Cursor; idx < len(plan.Agents); idx++ {
		if handled >= agentsPerExecution {
			next := *payload
			next.Plan = plan
			next.Cursor = idx
			next.Result = result

			return nil, workflow.NewContinueAsNewError(ctx, AgentQualitySweepWorkflowName, &next)
		}
		handled++

		planned := plan.Agents[idx]
		sweepKey := sweepID + "/" + planned.AgentDefinitionID.String()
		if planned.Skip {
			if err := workflow.ExecuteActivity(stepCtx, a.RecordSkippedSuiteActivity,
				&agentqualityservice.RecordSkippedRequest{
					TenantInfo: tenant,
					Agent:      planned,
					SweepKey:   sweepKey,
					Trigger:    agentquality.SuiteRunTriggerScheduled,
				},
			).Get(stepCtx, nil); err != nil {
				logger.Warn("could not record an agent the sweep skipped",
					"agentDefinitionId", planned.AgentDefinitionID.String(), "error", err.Error())
				result.Failed++

				continue
			}
			result.Skipped++

			continue
		}

		var opened agentqualityservice.OpenedSuite
		if err := workflow.ExecuteActivity(stepCtx, a.OpenSuiteActivity,
			&agentqualityservice.OpenSuiteRequest{
				TenantInfo: tenant,
				Agent:      planned,
				SweepKey:   sweepKey,
				Trigger:    agentquality.SuiteRunTriggerScheduled,
				Settings:   plan.Settings,
			},
		).Get(stepCtx, &opened); err != nil {
			logger.Warn("could not open an agent's suite run",
				"agentDefinitionId", planned.AgentDefinitionID.String(), "error", err.Error())
			result.Failed++

			continue
		}
		if opened.Status.Terminal() {
			result.Ran++

			continue
		}

		childCtx := workflow.WithChildOptions(ctx, childOptions(opened.WorkflowID, tenant.OrgID))
		if err := workflow.ExecuteChildWorkflow(childCtx, AgentSuiteRunWorkflowName,
			&SuiteRunPayload{
				OrganizationID:    tenant.OrgID,
				BusinessUnitID:    tenant.BuID,
				SuiteRunID:        opened.SuiteRunID,
				AgentDefinitionID: planned.AgentDefinitionID,
				SampleSeed:        opened.SampleSeed,
				Settings:          plan.Settings,
				DayStart:          plan.DayStart,
				MonthStart:        plan.MonthStart,
			},
		).Get(childCtx, nil); err != nil {
			logger.Warn("an agent's suite run did not finish",
				"suiteRunId", opened.SuiteRunID.String(), "error", err.Error())
			result.Failed++

			continue
		}
		result.Ran++
	}

	return result, nil
}

func AgentSuiteRunWorkflow(
	ctx workflow.Context,
	payload *SuiteRunPayload,
) (*SuiteRunResult, error) {
	var a *Activities
	logger := workflow.GetLogger(ctx)
	tenant := payload.tenantInfo()
	stepCtx := workflow.WithActivityOptions(ctx, withPriority(stepOptions, tenant.OrgID))

	stop := payload.StopReason
	cursor := payload.Cursor
	replayed := 0
	for stop == "" {
		room := casesPerExecution - replayed
		if room <= 0 {
			next := *payload
			next.Cursor = cursor
			next.Replayed = payload.Replayed + replayed

			return nil, workflow.NewContinueAsNewError(ctx, AgentSuiteRunWorkflowName, &next)
		}

		var cases []agentqualityservice.SuiteCase
		if err := workflow.ExecuteActivity(stepCtx, a.ListSuiteCasesActivity,
			&agentqualityservice.ListSuiteCasesRequest{
				TenantInfo:   tenant,
				SuiteRunID:   payload.SuiteRunID,
				AfterOrdinal: cursor,
				Limit:        room,
			},
		).Get(stepCtx, &cases); err != nil {
			return nil, failSuite(ctx, a, payload, err)
		}
		if len(cases) == 0 {
			break
		}

		for _, suiteCase := range cases {
			cursor = suiteCase.Ordinal
			if suiteCase.Status.Terminal() {
				continue
			}

			budget, err := checkBudget(stepCtx, a, payload)
			if err != nil {
				return nil, failSuite(ctx, a, payload, err)
			}
			if budget.Stop {
				stop = budget.Reason

				break
			}

			replayed++
			childCtx := workflow.WithChildOptions(ctx, childOptions(
				agentquality.EvaluationWorkflowID(suiteCase.EvaluationID),
				tenant.OrgID,
			))
			if err = workflow.ExecuteChildWorkflow(childCtx, agentjobs.AgentEvaluationWorkflowName,
				&agentjobs.AgentEvaluationPayload{
					BasePayload: temporaltype.BasePayload{
						OrganizationID: tenant.OrgID,
						BusinessUnitID: tenant.BuID,
						UserID:         payload.UserID,
						Timestamp:      payload.DayStart,
					},
					EvaluationID: suiteCase.EvaluationID,
				},
			).Get(childCtx, nil); err != nil {
				logger.Warn("a case's replay did not finish; it is recorded as failed",
					"evaluationId", suiteCase.EvaluationID.String(), "error", err.Error())
			}
		}

		if len(cases) < room {
			break
		}
	}

	var scored agentqualityservice.ScoredSuite
	if err := workflow.ExecuteActivity(stepCtx, a.ScoreSuiteActivity,
		&agentqualityservice.ScoreSuiteRequest{
			TenantInfo: tenant,
			SuiteRunID: payload.SuiteRunID,
			SampleSeed: payload.SampleSeed,
			Settings:   payload.Settings,
		},
	).Get(stepCtx, &scored); err != nil {
		return nil, failSuite(ctx, a, payload, err)
	}

	judged := 0
	if stop == "" && payload.Settings.JudgeEnabled {
		judgeCtx := workflow.WithActivityOptions(ctx, withPriority(judgeOptions, tenant.OrgID))
		for _, evaluationID := range scored.JudgeCases {
			budget, err := checkBudget(stepCtx, a, payload)
			if err != nil {
				return nil, failSuite(ctx, a, payload, err)
			}
			if budget.Stop {
				stop = budget.Reason

				break
			}

			var verdict agentqualityservice.JudgedCase
			if err = workflow.ExecuteActivity(judgeCtx, a.JudgeCaseActivity,
				&agentqualityservice.JudgeCaseRequest{
					TenantInfo:   tenant,
					EvaluationID: evaluationID,
				},
			).Get(judgeCtx, &verdict); err != nil {
				logger.Warn("the judge could not read a case; it keeps its checks' score",
					"evaluationId", evaluationID.String(), "error", err.Error())

				continue
			}
			if verdict.Judged {
				judged++
			}
		}
	}

	finalizeCtx := workflow.WithActivityOptions(ctx, withPriority(finalizeOptions, tenant.OrgID))
	var finalized agentqualityservice.FinalizedSuite
	if err := workflow.ExecuteActivity(finalizeCtx, a.FinalizeSuiteActivity,
		&agentqualityservice.FinalizeSuiteRequest{
			TenantInfo: tenant,
			SuiteRunID: payload.SuiteRunID,
			Settings:   payload.Settings,
			StopReason: stop,
		},
	).Get(finalizeCtx, &finalized); err != nil {
		return nil, failSuite(ctx, a, payload, err)
	}

	return &SuiteRunResult{
		Replayed:     payload.Replayed + replayed,
		Judged:       judged,
		Status:       finalized.Status,
		QualityScore: finalized.QualityScore,
		Regression:   finalized.Regression,
	}, nil
}

func checkBudget(
	ctx workflow.Context,
	a *Activities,
	payload *SuiteRunPayload,
) (agentquality.BudgetDecision, error) {
	var decision agentquality.BudgetDecision
	err := workflow.ExecuteActivity(ctx, a.CheckEvalBudgetActivity,
		&agentqualityservice.CheckBudgetRequest{
			TenantInfo: payload.tenantInfo(),
			DayStart:   payload.DayStart,
			MonthStart: payload.MonthStart,
			Settings:   payload.Settings,
		},
	).Get(ctx, &decision)

	return decision, err
}

func failSuite(
	ctx workflow.Context,
	a *Activities,
	payload *SuiteRunPayload,
	cause error,
) error {
	failCtx, cancel := workflow.NewDisconnectedContext(ctx)
	defer cancel()
	failCtx = workflow.WithActivityOptions(
		failCtx,
		withPriority(stepOptions, payload.OrganizationID),
	)

	if err := workflow.ExecuteActivity(failCtx, a.FailSuiteActivity,
		&agentqualityservice.FailSuiteRequest{
			TenantInfo: payload.tenantInfo(),
			SuiteRunID: payload.SuiteRunID,
			Error:      cause.Error(),
		},
	).Get(failCtx, nil); err != nil {
		workflow.GetLogger(ctx).Error("could not record why a suite run failed",
			"suiteRunId", payload.SuiteRunID.String(), "error", err.Error())
	}

	return cause
}

func ReconcileQualitySchedulesWorkflow(ctx workflow.Context) (*ReconcileResult, error) {
	var a *Activities

	reconcileCtx := workflow.WithActivityOptions(ctx, reconcileOptions)
	var result ReconcileResult
	if err := workflow.ExecuteActivity(reconcileCtx, a.ReconcileQualitySchedulesActivity).
		Get(reconcileCtx, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
