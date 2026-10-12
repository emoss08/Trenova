package jobs

import (
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingqueuejobs"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/billingtransferjobs"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/creditreviewjobs"
	"github.com/emoss08/trenova/internal/modules/billing/jobs/invoiceadjustmentjobs"
	"go.uber.org/fx"
)

func WorkerOptions() fx.Option {
	return fx.Options(
		billingtransferjobs.Module,
		billingqueuejobs.Module,
		invoiceadjustmentjobs.Module,
		creditreviewjobs.Module,
	)
}
