package factoringcontract

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/billing/ports/factoringports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/temporal"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	EvaluateWorkflowName     = "EvaluateFactoringReadinessWorkflow"
	SweepWorkflowName        = "FactoringReadinessSweepWorkflow"
	EvaluateSignalName       = "factoring-readiness-evaluate"
	evaluateWorkflowIDPrefix = "factoring-readiness-"
	MaxPendingTargets        = 2000
)

type EvaluatePayload struct {
	OrganizationID pulid.ID                         `json:"organizationId"`
	BusinessUnitID pulid.ID                         `json:"businessUnitId"`
	Pending        factoringports.EvaluationTargets `json:"pending"`
	Everything     bool                             `json:"everything"`
	Iterations     int                              `json:"iterations"`
}

func (p *EvaluatePayload) TenantInfo() pagination.TenantInfo {
	return pagination.TenantInfo{OrgID: p.OrganizationID, BuID: p.BusinessUnitID}
}

func (p *EvaluatePayload) Merge(targets *factoringports.EvaluationTargets) {
	if targets == nil || p.Everything {
		return
	}
	p.Pending.InvoiceIDs = appendNew(p.Pending.InvoiceIDs, targets.InvoiceIDs)
	p.Pending.CustomerIDs = appendNew(p.Pending.CustomerIDs, targets.CustomerIDs)
	p.Pending.FactoringCompanyIDs = appendNew(
		p.Pending.FactoringCompanyIDs,
		targets.FactoringCompanyIDs,
	)
	if len(p.Pending.InvoiceIDs)+len(p.Pending.CustomerIDs)+
		len(p.Pending.FactoringCompanyIDs) > MaxPendingTargets {
		p.Everything = true
		p.Pending = factoringports.EvaluationTargets{}
	}
}

func (p *EvaluatePayload) HasWork() bool {
	return p.Everything || !p.Pending.Empty()
}

func (p *EvaluatePayload) Take() *EvaluatePayload {
	taken := &EvaluatePayload{
		OrganizationID: p.OrganizationID,
		BusinessUnitID: p.BusinessUnitID,
		Pending:        p.Pending,
		Everything:     p.Everything,
	}
	p.Pending = factoringports.EvaluationTargets{}
	p.Everything = false
	return taken
}

type EvaluateSignal struct {
	Targets factoringports.EvaluationTargets `json:"targets"`
}

func EvaluateWorkflowID(tenantInfo pagination.TenantInfo) string {
	return evaluateWorkflowIDPrefix + tenantInfo.OrgID.String() + "-" + tenantInfo.BuID.String()
}

type RequesterParams struct {
	fx.In

	Signals kernelports.WorkflowSignalStarter `optional:"true"`
	Logger  *zap.Logger
}

type Requester struct {
	signals kernelports.WorkflowSignalStarter
	l       *zap.Logger
}

var _ factoringports.EvaluationRequester = (*Requester)(nil)

func NewRequester(p RequesterParams) factoringports.EvaluationRequester {
	return &Requester{signals: p.Signals, l: p.Logger.Named("factoring-readiness-requester")}
}

func (r *Requester) RequestEvaluation(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	targets *factoringports.EvaluationTargets,
) {
	if r.signals == nil || targets.Empty() || tenantInfo.OrgID.IsNil() || tenantInfo.BuID.IsNil() {
		return
	}
	id := EvaluateWorkflowID(tenantInfo)
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		_, err := r.signals.SignalWithStartWorkflow(
			committed,
			id,
			EvaluateSignalName,
			EvaluateSignal{Targets: *targets},
			client.StartWorkflowOptions{
				ID:            id,
				TaskQueue:     temporaltype.TaskQueueBilling.String(),
				StaticSummary: "Work out which invoices are ready to factor",
				Priority:      temporal.Priority{FairnessKey: tenantInfo.OrgID.String()},
			},
			EvaluateWorkflowName,
			&EvaluatePayload{
				OrganizationID: tenantInfo.OrgID,
				BusinessUnitID: tenantInfo.BuID,
			},
		)
		if err != nil {
			r.l.Warn("could not request a factoring readiness evaluation",
				zap.String("organizationId", tenantInfo.OrgID.String()),
				zap.Error(err))
		}
	})
}

func appendNew(existing, incoming []pulid.ID) []pulid.ID {
	for _, id := range incoming {
		if id.IsNil() || slices.Contains(existing, id) {
			continue
		}
		existing = append(existing, id)
	}
	return existing
}
