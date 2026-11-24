package bankreceiptcandidatecontract

import (
	"context"

	"github.com/emoss08/trenova/internal/infrastructure/temporaljobs"
	"github.com/emoss08/trenova/internal/modules/billing/ports/bankreceiptports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/workqueueports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/temporaltype"
	"github.com/emoss08/trenova/shared/pulid"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

const (
	DrainKind         = "bank-receipt-candidates"
	DrainWorkflowName = "BankReceiptCandidateDrainWorkflow"
	DrainSignalName   = "bank-receipt-candidates-drain"
	WakeWorkflowName  = "BankReceiptCandidateWakeWorkflow"
	SweepWorkflowName = "BankReceiptCandidateSweepWorkflow"
)

type RequesterParams struct {
	fx.In

	Queue   workqueueports.Queue
	Signals kernelports.WorkflowSignalStarter
	Logger  *zap.Logger
}

type Requester struct {
	drain *temporaljobs.DrainRequester
	l     *zap.Logger
}

var _ bankreceiptports.CandidateWaker = (*Requester)(nil)

func NewRequester(p RequesterParams) *Requester {
	l := p.Logger.Named("bank-receipt-candidate-requester")

	return &Requester{
		drain: &temporaljobs.DrainRequester{
			Queue:     p.Queue,
			Signals:   p.Signals,
			Kind:      DrainKind,
			Workflow:  DrainWorkflowName,
			TaskQueue: temporaltype.TaskQueueBilling.String(),
			Signal:    DrainSignalName,
			Summary:   "Find the open items each of the tenant's unapplied bank receipts could pay",
			Logger:    l,
		},
		l: l,
	}
}

func (r *Requester) Wake(ctx context.Context, tenant pagination.TenantInfo) {
	kernelports.AfterCommit(ctx, func(committed context.Context) {
		if err := r.drain.Wake(committed, tenant); err != nil {
			r.l.Warn("could not wake the bank receipt candidate drain; the backstop will",
				zap.String("organizationId", tenant.OrgID.String()),
				zap.Error(err))
		}
	})
}

func (r *Requester) Request(
	ctx context.Context,
	tenant pagination.TenantInfo,
	subjectIDs ...pulid.ID,
) error {
	return r.drain.Request(ctx, tenant, subjectIDs...)
}

func (r *Requester) WakeDue(ctx context.Context, limit int) (int, error) {
	return r.drain.WakeDue(ctx, limit)
}
