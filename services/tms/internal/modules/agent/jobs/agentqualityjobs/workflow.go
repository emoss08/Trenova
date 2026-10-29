package agentqualityjobs

import (
	"slices"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentqualityengine"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitycontrol"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentqualitysuite"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentflow/agentflowcontract"
	"github.com/emoss08/trenova/internal/modules/agent/jobs/agentjobs/agentcontract"
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
	submitBatchOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy:         &temporal.RetryPolicy{MaximumAttempts: 1},
	}
	collectBatchOptions = workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy:         stepRetry,
	}
)

const (
	judgeBatchFirstPoll = time.Minute
	judgeBatchMaxPoll   = 15 * time.Minute
	judgeBatchWait      = 25 * time.Hour
)

func evaluationPriority(orgID pulid.ID) temporal.Priority {
	return temporal.Priority{
		PriorityKey: agentflowcontract.PriorityEvaluation,
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
		var planned agentqualityengine.SweepPlan
		if err := workflow.ExecuteActivity(planCtx, a.PlanSweepActivity,
			&agentqualityengine.SweepRequest{
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
				&agentqualityengine.RecordSkippedRequest{
					TenantInfo: tenant,
					Agent:      planned,
					SweepKey:   sweepKey,
					Trigger:    agentqualitysuite.SuiteRunTriggerScheduled,
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

		var opened agentqualityengine.OpenedSuite
		if err := workflow.ExecuteActivity(stepCtx, a.OpenSuiteActivity,
			&agentqualityengine.OpenSuiteRequest{
				TenantInfo: tenant,
				Agent:      planned,
				SweepKey:   sweepKey,
				Trigger:    agentqualitysuite.SuiteRunTriggerScheduled,
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
		var child SuiteRunResult
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
				Unattended:        true,
				CommittedUSD:      result.CommittedUSD,
			},
		).Get(childCtx, &child); err != nil {
			logger.Warn("an agent's suite run did not finish",
				"suiteRunId", opened.SuiteRunID.String(), "error", err.Error())
			result.Failed++

			continue
		}
		if child.CommittedUSD.IsPositive() {
			result.CommittedUSD = result.CommittedUSD.Add(child.CommittedUSD)
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

		var cases []agentqualityengine.SuiteCase
		if err := workflow.ExecuteActivity(stepCtx, a.ListSuiteCasesActivity,
			&agentqualityengine.ListSuiteCasesRequest{
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
				agentqualitycontrol.EvaluationWorkflowID(suiteCase.EvaluationID),
				tenant.OrgID,
			))
			if err = workflow.ExecuteChildWorkflow(childCtx, agentcontract.AgentEvaluationWorkflowName,
				&agentcontract.AgentEvaluationPayload{
					BasePayload: temporaltype.BasePayload{
						OrganizationID: tenant.OrgID,
						BusinessUnitID: tenant.BuID,
						UserID:         payload.UserID,
						Timestamp:      payload.DayStart,
					},
					EvaluationID: suiteCase.EvaluationID,
				},
			).
				Get(childCtx, nil); err != nil {
				logger.Warn("a case's replay did not finish; it is recorded as failed",
					"evaluationId", suiteCase.EvaluationID.String(), "error", err.Error())
			}
		}

		if len(cases) < room {
			break
		}
	}

	var scored agentqualityengine.ScoredSuite
	if err := workflow.ExecuteActivity(stepCtx, a.ScoreSuiteActivity,
		&agentqualityengine.ScoreSuiteRequest{
			TenantInfo: tenant,
			SuiteRunID: payload.SuiteRunID,
			SampleSeed: payload.SampleSeed,
			Settings:   payload.Settings,
		},
	).Get(stepCtx, &scored); err != nil {
		return nil, failSuite(ctx, a, payload, err)
	}

	judged := 0
	if stop == "" && payload.Settings.JudgeEnabled && len(scored.JudgeCases) > 0 {
		if payload.Unattended {
			batch := submitJudgeBatch(ctx, a, payload, scored.JudgeCases)
			if batch.Batched() {
				return judgeLater(ctx, a, payload, payload.Replayed+replayed, batch)
			}
			stop = batch.StopReason
		}
		if stop == "" {
			var err error
			if judged, stop, err = judgeEach(ctx, a, payload, scored.JudgeCases); err != nil {
				return nil, failSuite(ctx, a, payload, err)
			}
		}
	}

	return finalizeSuite(ctx, a, payload, stop, payload.Replayed+replayed, judged)
}

func AgentSuiteJudgeWorkflow(
	ctx workflow.Context,
	payload *SuiteJudgePayload,
) (*SuiteRunResult, error) {
	var a *Activities

	return judgeBatchAndFinalize(ctx, a, payload)
}

func submitJudgeBatch(
	ctx workflow.Context,
	a *Activities,
	payload *SuiteRunPayload,
	evaluationIDs []pulid.ID,
) *agentqualityengine.JudgeBatch {
	submitCtx := workflow.WithActivityOptions(
		ctx,
		withPriority(submitBatchOptions, payload.OrganizationID),
	)
	var batch agentqualityengine.JudgeBatch
	if err := workflow.ExecuteActivity(submitCtx, a.SubmitJudgeBatchActivity,
		&agentqualityengine.SubmitJudgeBatchRequest{
			TenantInfo:    payload.tenantInfo(),
			SuiteRunID:    payload.SuiteRunID,
			EvaluationIDs: evaluationIDs,
			DayStart:      payload.DayStart,
			MonthStart:    payload.MonthStart,
			Settings:      payload.Settings,
			CommittedUSD:  payload.CommittedUSD,
		},
	).Get(submitCtx, &batch); err != nil {
		workflow.GetLogger(ctx).
			Warn("the judge batch could not be sent; judging each case on its own",
				"suiteRunId", payload.SuiteRunID.String(), "error", err.Error())

		return &agentqualityengine.JudgeBatch{}
	}

	return &batch
}

func judgeLater(
	ctx workflow.Context,
	a *Activities,
	payload *SuiteRunPayload,
	replayed int,
	batch *agentqualityengine.JudgeBatch,
) (*SuiteRunResult, error) {
	judging := &SuiteJudgePayload{Suite: *payload, Batch: *batch}
	judging.Suite.Replayed = replayed
	judging.Suite.Cursor = 0

	childCtx := workflow.WithChildOptions(ctx, childOptions(
		SuiteJudgeWorkflowID(payload.SuiteRunID),
		payload.OrganizationID,
	))
	if err := workflow.ExecuteChildWorkflow(childCtx, AgentSuiteJudgeWorkflowName, judging).
		GetChildWorkflowExecution().
		Get(childCtx, nil); err != nil {
		workflow.GetLogger(ctx).Warn("the judge batch could not be handed off; waiting for it here",
			"suiteRunId", payload.SuiteRunID.String(), "error", err.Error())

		return judgeBatchAndFinalize(ctx, a, judging)
	}

	return &SuiteRunResult{
		Replayed:     replayed,
		Status:       agentqualitysuite.SuiteRunStatusRunning,
		Judging:      true,
		CommittedUSD: batch.EstimateUSD,
	}, nil
}

func judgeBatchAndFinalize(
	ctx workflow.Context,
	a *Activities,
	judging *SuiteJudgePayload,
) (*SuiteRunResult, error) {
	suite := &judging.Suite
	batch := &judging.Batch

	judged := 0
	pending := batch.Submitted
	if awaitJudgeBatch(ctx, a, suite, batch) {
		collectCtx := workflow.WithActivityOptions(
			ctx,
			withPriority(collectBatchOptions, suite.OrganizationID),
		)
		var collected agentqualityengine.CollectedJudgeBatch
		if err := workflow.ExecuteActivity(collectCtx, a.CollectJudgeBatchActivity,
			&agentqualityengine.CollectJudgeBatchRequest{
				TenantInfo: suite.tenantInfo(),
				SuiteRunID: suite.SuiteRunID,
				Batch:      *batch,
			},
		).Get(collectCtx, &collected); err != nil {
			workflow.GetLogger(ctx).Warn("the judge batch's results could not be read; "+
				"judging each case on its own",
				"suiteRunId", suite.SuiteRunID.String(), "error", err.Error())
		} else {
			judged = collected.Judged
			pending = collected.Retry
		}
	}

	pending = append(slices.Clone(pending), batch.Deferred...)
	stop := ""
	if len(pending) > 0 {
		more, reason, err := judgeEach(ctx, a, suite, pending)
		if err != nil {
			return nil, failSuite(ctx, a, suite, err)
		}
		judged += more
		stop = reason
	}

	return finalizeSuite(ctx, a, suite, stop, suite.Replayed, judged)
}

func awaitJudgeBatch(
	ctx workflow.Context,
	a *Activities,
	suite *SuiteRunPayload,
	batch *agentqualityengine.JudgeBatch,
) bool {
	logger := workflow.GetLogger(ctx)
	pollCtx := workflow.WithActivityOptions(ctx, withPriority(stepOptions, suite.OrganizationID))
	deadline := workflow.Now(ctx).Add(judgeBatchWait)
	wait := judgeBatchFirstPoll
	for {
		if err := workflow.Sleep(ctx, wait); err != nil {
			return false
		}

		var status agentqualityengine.JudgeBatchStatus
		err := workflow.ExecuteActivity(pollCtx, a.CheckJudgeBatchActivity,
			&agentqualityengine.CheckJudgeBatchRequest{
				TenantInfo: suite.tenantInfo(),
				Batch:      *batch,
			},
		).Get(pollCtx, &status)
		if err == nil && status.Ended {
			return true
		}
		if err != nil {
			logger.Warn("could not ask how the judge batch is going",
				"suiteRunId", suite.SuiteRunID.String(), "error", err.Error())
		}
		if !workflow.Now(ctx).Add(wait).Before(deadline) {
			logger.Warn("the judge batch did not end in time; judging each case on its own",
				"suiteRunId", suite.SuiteRunID.String(), "batchId", batch.BatchID)

			return false
		}
		wait = min(wait*2, judgeBatchMaxPoll)
	}
}

func judgeEach(
	ctx workflow.Context,
	a *Activities,
	payload *SuiteRunPayload,
	evaluationIDs []pulid.ID,
) (int, string, error) {
	logger := workflow.GetLogger(ctx)
	tenant := payload.tenantInfo()
	stepCtx := workflow.WithActivityOptions(ctx, withPriority(stepOptions, tenant.OrgID))
	judgeCtx := workflow.WithActivityOptions(ctx, withPriority(judgeOptions, tenant.OrgID))

	judged := 0
	for _, evaluationID := range evaluationIDs {
		budget, err := checkBudget(stepCtx, a, payload)
		if err != nil {
			return judged, "", err
		}
		if budget.Stop {
			return judged, budget.Reason, nil
		}

		var verdict agentqualityengine.JudgedCase
		if err = workflow.ExecuteActivity(judgeCtx, a.JudgeCaseActivity,
			&agentqualityengine.JudgeCaseRequest{
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

	return judged, "", nil
}

func finalizeSuite(
	ctx workflow.Context,
	a *Activities,
	payload *SuiteRunPayload,
	stop string,
	replayed int,
	judged int,
) (*SuiteRunResult, error) {
	finalizeCtx := workflow.WithActivityOptions(
		ctx,
		withPriority(finalizeOptions, payload.OrganizationID),
	)
	var finalized agentqualityengine.FinalizedSuite
	if err := workflow.ExecuteActivity(finalizeCtx, a.FinalizeSuiteActivity,
		&agentqualityengine.FinalizeSuiteRequest{
			TenantInfo: payload.tenantInfo(),
			SuiteRunID: payload.SuiteRunID,
			Settings:   payload.Settings,
			StopReason: stop,
		},
	).Get(finalizeCtx, &finalized); err != nil {
		return nil, failSuite(ctx, a, payload, err)
	}

	return &SuiteRunResult{
		Replayed:     replayed,
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
) (agentqualitysuite.BudgetDecision, error) {
	var decision agentqualitysuite.BudgetDecision
	err := workflow.ExecuteActivity(ctx, a.CheckEvalBudgetActivity,
		&agentqualityengine.CheckBudgetRequest{
			TenantInfo:   payload.tenantInfo(),
			DayStart:     payload.DayStart,
			MonthStart:   payload.MonthStart,
			Settings:     payload.Settings,
			CommittedUSD: payload.CommittedUSD,
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
		&agentqualityengine.FailSuiteRequest{
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
