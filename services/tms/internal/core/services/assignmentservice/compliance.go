package assignmentservice

import (
	"github.com/emoss08/trenova/internal/core/domain/dispatchcontrol"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/internal/core/services/dispatcheligibility"
	"github.com/emoss08/trenova/pkg/errortypes"
)

func runWorkerComplianceChecks(
	w *worker.Worker,
	dc *dispatchcontrol.DispatchControl,
	hasHazmatCommodities bool,
	prefix string,
	multiErr *errortypes.MultiError,
) {
	dispatcheligibility.EvaluateWorkerCompliance(dispatcheligibility.WorkerComplianceInput{
		Worker:               w,
		Control:              dc,
		HasHazmatCommodities: hasHazmatCommodities,
	}).AppendToMultiError(multiErr, prefix)
}
