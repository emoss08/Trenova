package assignmentservice

import (
	"github.com/emoss08/trenova/internal/modules/operations/app/dispatcheligibility"
	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func runWorkerComplianceChecks(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	hasHazmatCommodities bool,
	prefix string,
	now int64,
	multiErr *errortypes.MultiError,
) {
	dispatcheligibility.EvaluateWorkerCompliance(dispatcheligibility.WorkerComplianceInput{
		Worker:               w,
		Control:              dc,
		HasHazmatCommodities: hasHazmatCommodities,
		Now:                  now,
	}).AppendToMultiError(multiErr, prefix)
}
