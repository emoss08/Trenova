package accountingsyncjobs

import (
	"context"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/accountingsync"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	PollAccountingChangesWorkflowName = "PollAccountingChangesWorkflow"
	KickAccountingChangesWorkflowName = "KickAccountingChangesWorkflow"
	ChangesWorkflowIDPrefix           = "accounting-changes:"
	ChangesSignalName                 = "accounting-changes-kick"
	changesReadsPerRun                = 50
	changesIdleWait                   = 30 * time.Second
	changesEvaluationRounds           = 10
	changesConnectionsPage            = 100
	changesSettleMargin               = 5 * time.Second
)

var changesActivityOptions = workflow.ActivityOptions{
	StartToCloseTimeout: 5 * time.Minute,
	RetryPolicy: &temporal.RetryPolicy{
		InitialInterval:    10 * time.Second,
		BackoffCoefficient: 2.0,
		MaximumAttempts:    3,
		MaximumInterval:    time.Minute,
	},
}

type ChangesPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	ConnectionID   pulid.ID `json:"connectionId"`
	Reads          int      `json:"reads"`
}

func (p *ChangesPayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

type ChangesSignal struct{}

type ChangesRunResult struct {
	Reads     int  `json:"reads"`
	Recorded  int  `json:"recorded"`
	Applied   int  `json:"applied"`
	Proposed  int  `json:"proposed"`
	Held      bool `json:"held"`
	Evaluated int  `json:"evaluated"`
}

type KickChangesResult struct {
	Connections int `json:"connections"`
	Kicked      int `json:"kicked"`
}

type (
	ChangesReadResult       = services.AccountingChangesPollResult
	ChangesEvaluationResult = services.AccountingInboundEvaluation
)

func ChangesWorkflowID(connectionID pulid.ID) string {
	return ChangesWorkflowIDPrefix + connectionID.String()
}

func changesWorkflows() []temporaltype.WorkflowDefinition {
	return []temporaltype.WorkflowDefinition{
		{
			Name:        PollAccountingChangesWorkflowName,
			Fn:          PollAccountingChangesWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Read what changed in one connection's accounting system and bring payments in",
		},
		{
			Name:        KickAccountingChangesWorkflowName,
			Fn:          KickAccountingChangesWorkflow,
			TaskQueue:   temporaltype.IntegrationTaskQueue,
			Description: "Wake the change reader for every accounting connection that is syncing",
		},
	}
}

func PollAccountingChangesWorkflow(
	ctx workflow.Context,
	payload *ChangesPayload,
) (*ChangesRunResult, error) {
	var a *Activities
	signals := workflow.GetSignalChannel(ctx, ChangesSignalName)
	actx := workflow.WithActivityOptions(
		ctx,
		withFairness(&changesActivityOptions, payload.OrganizationID),
	)
	result := &ChangesRunResult{Reads: payload.Reads}

	for {
		drainChangeSignals(signals)
		if result.Reads >= changesReadsPerRun {
			return result, workflow.NewContinueAsNewError(
				ctx,
				PollAccountingChangesWorkflow,
				&ChangesPayload{
					OrganizationID: payload.OrganizationID,
					BusinessUnitID: payload.BusinessUnitID,
					ConnectionID:   payload.ConnectionID,
				},
			)
		}

		var read ChangesReadResult
		if err := workflow.ExecuteActivity(actx, a.PollAccountingChangesActivity, payload).
			Get(actx, &read); err != nil {
			return result, err
		}
		result.Reads++
		result.Recorded += read.Recorded
		if read.Held {
			result.Held = true
			return result, nil
		}
		if read.More {
			continue
		}

		if read.Recorded > 0 {
			if err := workflow.Sleep(
				ctx,
				accountingsync.InboundEvaluationSettle+changesSettleMargin,
			); err != nil {
				return result, err
			}
		}
		if err := evaluateChanges(actx, a, payload, result); err != nil {
			return result, err
		}
		if !waitForChangeSignal(ctx, signals) {
			return result, nil
		}
	}
}

func evaluateChanges(
	actx workflow.Context,
	a *Activities,
	payload *ChangesPayload,
	result *ChangesRunResult,
) error {
	for range changesEvaluationRounds {
		var evaluated ChangesEvaluationResult
		if err := workflow.ExecuteActivity(actx, a.EvaluateAccountingInboundActivity, payload).
			Get(actx, &evaluated); err != nil {
			return err
		}
		result.Evaluated += evaluated.Evaluated
		result.Applied += evaluated.Applied
		result.Proposed += evaluated.Proposed
		if !evaluated.More {
			return nil
		}
	}
	return nil
}

func drainChangeSignals(signals workflow.ReceiveChannel) {
	for {
		var signal ChangesSignal
		if !signals.ReceiveAsync(&signal) {
			return
		}
	}
}

func waitForChangeSignal(ctx workflow.Context, signals workflow.ReceiveChannel) bool {
	timerCtx, cancel := workflow.WithCancel(ctx)
	defer cancel()

	received := false
	selector := workflow.NewSelector(ctx)
	selector.AddReceive(signals, func(channel workflow.ReceiveChannel, _ bool) {
		var signal ChangesSignal
		channel.Receive(ctx, &signal)
		received = true
	})
	selector.AddFuture(workflow.NewTimer(timerCtx, changesIdleWait), func(workflow.Future) {})
	selector.Select(ctx)
	return received
}

func KickAccountingChangesWorkflow(ctx workflow.Context) (*KickChangesResult, error) {
	var a *Activities
	actx := workflow.WithActivityOptions(ctx, sweepLongActivityOptions)
	result := new(KickChangesResult)
	err := workflow.ExecuteActivity(actx, a.KickAccountingChangesActivity).Get(actx, result)
	return result, err
}

func (a *Activities) PollAccountingChangesActivity(
	ctx context.Context,
	payload *ChangesPayload,
) (*ChangesReadResult, error) {
	result, err := a.inbound.PollChanges(ctx, &services.PollAccountingChangesRequest{
		TenantInfo:   payload.TenantInfo(),
		ConnectionID: payload.ConnectionID,
	})
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) EvaluateAccountingInboundActivity(
	ctx context.Context,
	payload *ChangesPayload,
) (*ChangesEvaluationResult, error) {
	result, err := a.inbound.Evaluate(ctx, &services.EvaluateAccountingInboundRequest{
		TenantInfo:   payload.TenantInfo(),
		ConnectionID: payload.ConnectionID,
	})
	if err != nil {
		return result, syncError(err)
	}
	return result, nil
}

func (a *Activities) KickAccountingChangesActivity(
	ctx context.Context,
) (*KickChangesResult, error) {
	result := new(KickChangesResult)
	afterID := pulid.Nil
	for {
		conns, err := a.connRepo.ListActive(
			ctx,
			repositories.ListActiveAccountingConnectionsRequest{
				AfterID: afterID,
				Limit:   changesConnectionsPage,
			},
		)
		if err != nil {
			return result, err
		}
		for _, conn := range conns {
			afterID = conn.ID
			if !conn.ReadsChanges() {
				continue
			}
			result.Connections++
			tenant := pagination.TenantInfo{OrgID: conn.OrganizationID, BuID: conn.BusinessUnitID}
			if kickErr := a.poller.PollNow(ctx, tenant, conn.ID); kickErr != nil {
				a.l.Warn("failed to wake an accounting change reader",
					zap.String("connectionId", conn.ID.String()), zap.Error(kickErr))
				continue
			}
			result.Kicked++
		}
		activity.RecordHeartbeat(ctx, result.Connections)
		if len(conns) < changesConnectionsPage {
			return result, nil
		}
	}
}

type ChangesPollerParams struct {
	fx.In

	Signals services.WorkflowSignalStarter `optional:"true"`
}

type ChangesPoller struct {
	signals services.WorkflowSignalStarter
}

var _ services.AccountingChangePoller = (*ChangesPoller)(nil)

func NewChangesPoller(p ChangesPollerParams) *ChangesPoller {
	return &ChangesPoller{signals: p.Signals}
}

func (p *ChangesPoller) PollNow(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	connectionID pulid.ID,
) error {
	if p.signals == nil {
		return services.ErrWorkflowStarterDisabled
	}
	_, err := p.signals.SignalWithStartWorkflow(
		ctx,
		ChangesWorkflowID(connectionID),
		ChangesSignalName,
		ChangesSignal{},
		client.StartWorkflowOptions{
			ID:        ChangesWorkflowID(connectionID),
			TaskQueue: temporaltype.IntegrationTaskQueue,
			StaticSummary: "Read changes from the accounting system for connection " +
				connectionID.String(),
		},
		PollAccountingChangesWorkflowName,
		&ChangesPayload{
			OrganizationID: tenantInfo.OrgID,
			BusinessUnitID: tenantInfo.BuID,
			ConnectionID:   connectionID,
		},
	)
	return err
}
