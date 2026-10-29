package accountingsyncjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/modules/accounting/jobs/accountingsyncjobs/accountingsynccontract"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingconnectionports"
	"github.com/emoss08/trenova/internal/modules/accounting/ports/accountingdriftports"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttrigger"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/zap"
)

const (
	KickAccountingDriftWorkflowName                 = "KickAccountingDriftWorkflow"
	AnnounceAccountingReconciliationDueWorkflowName = "AnnounceAccountingReconciliationDueWorkflow"
	driftBatchesPerRun                              = 50
	driftEventsPerRun                               = 20
	driftConnectionsPage                            = 100
)

var driftActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 10 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    30 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    4,
		MaximumInterval:    5 * time.Minute,
	},
}

type KickDriftResult struct {
	Connections int `json:"connections"`
	Started     int `json:"started"`
}

type AnnounceReconciliationResult struct {
	Connections int `json:"connections"`
}

func driftWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        accountingsynccontract.ReconcileAccountingDriftWorkflowName,
			Fn:          ReconcileAccountingDriftWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Compare what one accounting system holds with what Trenova sent it",
		},
		{
			Name:        KickAccountingDriftWorkflowName,
			Fn:          KickAccountingDriftWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Start the nightly drift check for every syncing accounting connection",
		},
		{
			Name:        AnnounceAccountingReconciliationDueWorkflowName,
			Fn:          AnnounceAccountingReconciliationDueWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Raise the weekly books reconciliation for every syncing accounting connection",
		},
	}
}

func ReconcileAccountingDriftWorkflow(
	ctx workflow.Context,
	payload *accountingsynccontract.DriftPayload,
) (*accountingsynccontract.DriftRunResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(
		ctx,
		withFairness(&driftActivityOptions, payload.OrganizationID),
	)
	if !payload.Started {
		payload.Started = true
		payload.EventsLeft = driftEventsPerRun
	}
	result := &payload.Totals

	for batches := 0; ; batches++ {
		if batches == driftBatchesPerRun {
			next := *payload
			return result, workflow.NewContinueAsNewError(
				ctx,
				ReconcileAccountingDriftWorkflow,
				&next,
			)
		}
		if payload.Balances {
			done, err := reconcileBalances(actx, a, payload)
			if err != nil || result.Held {
				return result, err
			}
			if done {
				break
			}
			continue
		}
		if err := reconcileRecords(actx, a, payload); err != nil || result.Held {
			return result, err
		}
	}

	err := workflow.ExecuteActivity(actx, a.FinishAccountingDriftCheckActivity, payload).
		Get(actx, nil)
	return result, err
}

func reconcileRecords(
	actx workflow.Context,
	a *Activities,
	payload *accountingsynccontract.DriftPayload,
) error {
	var batch accountingdriftports.AccountingDriftBatchResult
	if err := workflow.ExecuteActivity(actx, a.ReconcileAccountingDriftActivity, payload).
		Get(actx, &batch); err != nil {
		return err
	}
	totals := &payload.Totals
	totals.Batches++
	totals.Compared += batch.Compared
	totals.Opened += batch.Opened
	totals.Resolved += batch.Resolved
	totals.Events += batch.Events
	totals.Held = batch.Held
	payload.EventsLeft = max(payload.EventsLeft-batch.Events, 0)
	payload.AfterID = batch.LastID
	if !batch.More {
		payload.Balances = true
	}
	return nil
}

func reconcileBalances(
	actx workflow.Context,
	a *Activities,
	payload *accountingsynccontract.DriftPayload,
) (bool, error) {
	var page accountingdriftports.AccountingDriftBalanceResult
	if err := workflow.ExecuteActivity(actx, a.ReconcileAccountingDriftBalancesActivity, payload).
		Get(actx, &page); err != nil {
		return false, err
	}
	totals := &payload.Totals
	totals.Batches++
	totals.Customers += page.Customers
	totals.Opened += page.Opened
	totals.Resolved += page.Resolved
	totals.Events += page.Events
	totals.Held = page.Held
	payload.EventsLeft = max(payload.EventsLeft-page.Events, 0)
	payload.AfterCustomerID = page.LastCustomerID
	return !page.More, nil
}

func KickAccountingDriftWorkflow(ctx workflow.Context) (*KickDriftResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepLongActivityOptions)
	result := new(KickDriftResult)
	err := workflow.ExecuteActivity(actx, a.KickAccountingDriftActivity).Get(actx, result)
	return result, err
}

func AnnounceAccountingReconciliationDueWorkflow(
	ctx workflow.Context,
) (*AnnounceReconciliationResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepLongActivityOptions)
	result := new(AnnounceReconciliationResult)
	err := workflow.ExecuteActivity(actx, a.AnnounceAccountingReconciliationActivity).
		Get(actx, result)
	return result, err
}

func (a *Activities) ReconcileAccountingDriftActivity(
	ctx context.Context,
	payload *accountingsynccontract.DriftPayload,
) (*accountingdriftports.AccountingDriftBatchResult, error) {
	result, err := a.drift.ReconcileBatch(
		ctx,
		&accountingdriftports.ReconcileAccountingDriftRequest{
			TenantInfo:   payload.TenantInfo(),
			ConnectionID: payload.ConnectionID,
			AfterID:      payload.AfterID,
			EventBudget:  payload.EventsLeft,
		},
	)
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) ReconcileAccountingDriftBalancesActivity(
	ctx context.Context,
	payload *accountingsynccontract.DriftPayload,
) (*accountingdriftports.AccountingDriftBalanceResult, error) {
	result, err := a.drift.ReconcileBalances(
		ctx,
		&accountingdriftports.ReconcileAccountingDriftBalancesRequest{
			TenantInfo:      payload.TenantInfo(),
			ConnectionID:    payload.ConnectionID,
			AfterCustomerID: payload.AfterCustomerID,
			EventBudget:     payload.EventsLeft,
		},
	)
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) FinishAccountingDriftCheckActivity(
	ctx context.Context,
	payload *accountingsynccontract.DriftPayload,
) error {
	return syncError(
		a.drift.FinishCheck(ctx, &accountingdriftports.FinishAccountingDriftCheckRequest{
			TenantInfo:   payload.TenantInfo(),
			ConnectionID: payload.ConnectionID,
		}),
	)
}

func (a *Activities) eachDriftConnection(
	ctx context.Context,
	visit func(tenant pagination.TenantInfo, connectionID pulid.ID) error,
) (int, error) {
	visited := 0
	afterID := pulid.Nil
	for {
		conns, err := a.connRepo.ListActive(
			ctx,
			accountingconnectionports.ListActiveAccountingConnectionsRequest{
				AfterID: afterID,
				Limit:   driftConnectionsPage,
			},
		)
		if err != nil {
			return visited, err
		}
		for _, conn := range conns {
			afterID = conn.ID
			if !conn.ChecksDrift() {
				continue
			}
			visited++
			tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
			if visitErr := visit(tenant, conn.ID); visitErr != nil {
				a.l.Warn("failed to start accounting drift work for a connection",
					zap.String("connectionId", conn.ID.String()), zap.Error(visitErr))
			}
		}
		activity.RecordHeartbeat(ctx, visited)
		if len(conns) < driftConnectionsPage {
			return visited, nil
		}
	}
}

func (a *Activities) KickAccountingDriftActivity(ctx context.Context) (*KickDriftResult, error) {
	result := new(KickDriftResult)
	connections, err := a.eachDriftConnection(
		ctx,
		func(tenant pagination.TenantInfo, connectionID pulid.ID) error {
			if checkErr := a.driftChecker.CheckNow(
				dbscope.WithTenant(ctx, tenant.DBTenant()),
				tenant,
				connectionID,
			); checkErr != nil {
				return checkErr
			}
			result.Started++
			return nil
		},
	)
	result.Connections = connections
	return result, err
}

func (a *Activities) AnnounceAccountingReconciliationActivity(
	ctx context.Context,
) (*AnnounceReconciliationResult, error) {
	result := new(AnnounceReconciliationResult)
	connections, err := a.eachDriftConnection(
		ctx,
		func(tenant pagination.TenantInfo, connectionID pulid.ID) error {
			agentports.PublishAgentEvent(
				dbscope.WithTenant(ctx, tenant.DBTenant()),
				a.publisher,
				agentports.AgentEvent{
					Kind:       agenttrigger.EventAccountingReconciliationDue,
					SubjectID:  connectionID,
					TenantInfo: tenant,
				},
			)
			return nil
		},
	)
	result.Connections = connections
	return result, err
}
