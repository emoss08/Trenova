package billingqueuecontract

import (
	"context"
	"errors"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.temporal.io/api/enums/v1"
	"go.temporal.io/api/serviceerror"
	workflowpb "go.temporal.io/api/workflow/v1"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	NovaGenerationWorkflowName     = "BillingQueueNovaGenerationWorkflow"
	NovaGenerationSignalName       = "billing-queue-nova-generation"
	NovaGenerationProgressQuery    = "progress"
	NovaGenerationRealtimeResource = "billing-queue-nova"
	novaGenerationWorkflowIDPrefix = "billing-queue-nova-"
	novaGenerationSummary          = "Write the billing queue's missing reviews and brief"
)

type NovaGenerationPayload struct {
	OrganizationID pulid.ID `json:"organizationId"`
	BusinessUnitID pulid.ID `json:"businessUnitId"`
	Iterations     int      `json:"iterations"`
	// Watched carries, across continue-as-new, that someone is waiting on the
	// generation.
	Watched bool `json:"watched"`
}

func (p *NovaGenerationPayload) Tenant() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

// NovaGenerationProgress is what the progress query answers while a
// generation runs, and what the generation returns when it ends.
type NovaGenerationProgress struct {
	Generation billingqueueports.NovaGeneration `json:"generation"`
	// LeftUnreviewed is set as the generation finishes: whether items it could
	// not review were left.
	LeftUnreviewed bool `json:"leftUnreviewed"`
}

// NovaGenerationSignal asks a running generation to look again once it has
// finished what it is on, and says who asked.
type NovaGenerationSignal struct {
	Trigger billingqueueports.NovaTrigger `json:"trigger"`
}

func NovaGenerationWorkflowID(tenantInfo pagination.TenantInfo) string {
	return novaGenerationWorkflowIDPrefix + tenantInfo.OrgID.String() + "-" + tenantInfo.BuID.String()
}

type NovaGeneratorParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter `optional:"true"`
	Client  client.Client                     `optional:"true"`
	Logger  *zap.Logger
}

type NovaGenerator struct {
	signals kernelports.WorkflowSignalStarter
	client  client.Client
	l       *zap.Logger
}

var _ billingqueueports.NovaGenerator = (*NovaGenerator)(nil)

func NewNovaGenerator(p NovaGeneratorParams) billingqueueports.NovaGenerator {
	return &NovaGenerator{
		signals: p.Signals,
		client:  p.Client,
		l:       p.Logger.Named("billing-queue-nova-generator"),
	}
}

// RequestGeneration starts the organization's generation, or tells the one
// running to look again when it is done, so every trigger shares one run.
func (g *NovaGenerator) RequestGeneration(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	trigger billingqueueports.NovaTrigger,
) {
	if g.signals == nil || tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}

	id := NovaGenerationWorkflowID(tenantInfo)
	if _, err := g.signals.SignalWithStartWorkflow(
		ctx,
		id,
		NovaGenerationSignalName,
		NovaGenerationSignal{Trigger: trigger},
		client.StartWorkflowOptions{
			ID:            id,
			TaskQueue:     temporaltype.TaskQueueBilling.String(),
			StaticSummary: novaGenerationSummary,
			Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
		},
		NovaGenerationWorkflowName,
		&NovaGenerationPayload{OrganizationID: tenantInfo.OrgID, BusinessUnitID: tenantInfo.BuID},
	); err != nil {
		g.l.Warn("could not start the billing queue's Nova generation",
			zap.String("organizationId", tenantInfo.OrgID.String()), zap.Error(err))
	}
}

// Generation reads the running generation's progress from the workflow
// itself, so a generation that died is never shown as running, and how the
// last one ended.
func (g *NovaGenerator) Generation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) (billingqueueports.NovaGenerationState, error) {
	state := billingqueueports.NovaGenerationState{}
	if g.client == nil {
		return state, nil
	}

	id := NovaGenerationWorkflowID(tenantInfo)
	described, err := g.client.DescribeWorkflowExecution(ctx, id, "")
	if _, missing := errors.AsType[*serviceerror.NotFound](err); missing {
		return state, nil
	}
	if err != nil {
		return state, fmt.Errorf("describe the billing queue's Nova generation: %w", err)
	}
	info := described.GetWorkflowExecutionInfo()
	switch info.GetStatus() {
	case enums.WORKFLOW_EXECUTION_STATUS_RUNNING:
	case enums.WORKFLOW_EXECUTION_STATUS_COMPLETED:
		return g.ended(ctx, info)
	default:
		state.EndedAt = info.GetCloseTime().AsTime().Unix()
		return state, nil
	}

	encoded, err := g.client.QueryWorkflow(ctx, id, "", NovaGenerationProgressQuery)
	if err != nil {
		return state, fmt.Errorf("query the billing queue's Nova generation: %w", err)
	}
	var progress NovaGenerationProgress
	if err = encoded.Get(&progress); err != nil {
		return state, fmt.Errorf("decode the billing queue's Nova generation: %w", err)
	}
	if progress.Generation.Phase == billingqueueports.NovaPhaseDone {
		state.EndedAt = timeutils.NowUnix()
		state.LeftUnreviewed = progress.LeftUnreviewed
		return state, nil
	}
	state.Running = &progress.Generation

	return state, nil
}

// ended reads how a completed generation ended from its result.
func (g *NovaGenerator) ended(
	ctx context.Context,
	info *workflowpb.WorkflowExecutionInfo,
) (billingqueueports.NovaGenerationState, error) {
	state := billingqueueports.NovaGenerationState{EndedAt: info.GetCloseTime().AsTime().Unix()}
	var result NovaGenerationProgress
	execution := info.GetExecution()
	if err := g.client.GetWorkflow(ctx, execution.GetWorkflowId(), execution.GetRunId()).
		Get(ctx, &result); err != nil {
		return state, fmt.Errorf("read how the billing queue's Nova generation ended: %w", err)
	}
	state.LeftUnreviewed = result.LeftUnreviewed

	return state, nil
}
