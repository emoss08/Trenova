package accountingsyncjobs

import (
	"context"
	"errors"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	ReconcileAccountingDriftWorkflowName     = "ReconcileAccountingDriftWorkflow"
	KickAccountingDriftWorkflowName          = "KickAccountingDriftWorkflow"
	AnnounceAccountingReconciliationWorkflow = "AnnounceAccountingReconciliationWorkflow"
	DriftWorkflowIDPrefix                    = "accounting-drift:"
	driftBatchesPerRun                       = 50
	driftEventsPerRun                        = 20
	driftConnectionsPage                     = 100
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

type DriftPayload struct {
	OrganizationID  pulid.ID       `json:"organizationId"`
	BusinessUnitID  pulid.ID       `json:"businessUnitId"`
	ConnectionID    pulid.ID       `json:"connectionId"`
	Started         bool           `json:"started"`
	Balances        bool           `json:"balances"`
	AfterID         pulid.ID       `json:"afterId"`
	AfterCustomerID pulid.ID       `json:"afterCustomerId"`
	EventsLeft      int            `json:"eventsLeft"`
	Totals          DriftRunResult `json:"totals"`
}

func (p *DriftPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type DriftRunResult struct {
	Batches   int  `json:"batches"`
	Compared  int  `json:"compared"`
	Customers int  `json:"customers"`
	Opened    int  `json:"opened"`
	Resolved  int  `json:"resolved"`
	Events    int  `json:"events"`
	Held      bool `json:"held"`
}

type KickDriftResult struct {
	Connections int `json:"connections"`
	Started     int `json:"started"`
}

type AnnounceReconciliationResult struct {
	Connections int `json:"connections"`
}

func DriftWorkflowID(connectionID pulid.ID) string {
	return DriftWorkflowIDPrefix + connectionID.String()
}

func driftWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        ReconcileAccountingDriftWorkflowName,
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
			Name:        AnnounceAccountingReconciliationWorkflow,
			Fn:          AnnounceAccountingReconciliationDueWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Raise the weekly books reconciliation for every syncing accounting connection",
		},
	}
}

func ReconcileAccountingDriftWorkflow(
	ctx workflow.Context,
	payload *DriftPayload,
) (*DriftRunResult, error) {
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

func reconcileRecords(actx workflow.Context, a *Activities, payload *DriftPayload) error {
	var batch services.AccountingDriftBatchResult
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

func reconcileBalances(actx workflow.Context, a *Activities, payload *DriftPayload) (bool, error) {
	var page services.AccountingDriftBalanceResult
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
	payload *DriftPayload,
) (*services.AccountingDriftBatchResult, error) {
	result, err := a.drift.ReconcileBatch(ctx, &services.ReconcileAccountingDriftRequest{
		TenantInfo:   payload.TenantInfo(),
		ConnectionID: payload.ConnectionID,
		AfterID:      payload.AfterID,
		EventBudget:  payload.EventsLeft,
	})
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) ReconcileAccountingDriftBalancesActivity(
	ctx context.Context,
	payload *DriftPayload,
) (*services.AccountingDriftBalanceResult, error) {
	result, err := a.drift.ReconcileBalances(ctx, &services.ReconcileAccountingDriftBalancesRequest{
		TenantInfo:      payload.TenantInfo(),
		ConnectionID:    payload.ConnectionID,
		AfterCustomerID: payload.AfterCustomerID,
		EventBudget:     payload.EventsLeft,
	})
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) FinishAccountingDriftCheckActivity(
	ctx context.Context,
	payload *DriftPayload,
) error {
	return syncError(a.drift.FinishCheck(ctx, &services.FinishAccountingDriftCheckRequest{
		TenantInfo:   payload.TenantInfo(),
		ConnectionID: payload.ConnectionID,
	}))
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
			repositories.ListActiveAccountingConnectionsRequest{
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
			if checkErr := a.driftChecker.CheckNow(ctx, tenant, connectionID); checkErr != nil {
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
			services.PublishAgentEvent(ctx, a.publisher, services.AgentEvent{
				Kind:       agent.EventAccountingReconciliationDue,
				SubjectID:  connectionID,
				TenantInfo: tenant,
			})
			return nil
		},
	)
	result.Connections = connections
	return result, err
}

type DriftCheckerParams struct {
	fx.In

	Workflows services.WorkflowStarter
}

type DriftChecker struct {
	workflows services.WorkflowStarter
}

var _ services.AccountingDriftChecker = (*DriftChecker)(nil)

func NewDriftChecker(p DriftCheckerParams) *DriftChecker {
	return &DriftChecker{workflows: p.Workflows}
}

func (c *DriftChecker) CheckNow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	_, err := c.workflows.StartWorkflow(ctx, client.StartWorkflowOptions{
		ID:                       DriftWorkflowID(connectionID),
		TaskQueue:                temporaltype.IntegrationTaskQueue,
		WorkflowIDReusePolicy:    enums.WORKFLOW_ID_REUSE_POLICY_ALLOW_DUPLICATE,
		WorkflowIDConflictPolicy: enums.WORKFLOW_ID_CONFLICT_POLICY_USE_EXISTING,
		StaticSummary: "Compare the accounting system with Trenova for connection " +
			connectionID.String(),
	}, ReconcileAccountingDriftWorkflowName, &DriftPayload{
		OrganizationID: tenantInfo.OrgID,
		BusinessUnitID: tenantInfo.BuID,
		ConnectionID:   connectionID,
	})
	if errors.Is(err, services.ErrWorkflowStarterDisabled) {
		return errortypes.NewBusinessError(
			"Background work is not available on this server, so the books cannot be checked now",
		).WithInternal(err)
	}
	return err
}
