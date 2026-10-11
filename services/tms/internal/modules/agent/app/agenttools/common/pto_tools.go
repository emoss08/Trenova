package common

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
)

// ptoDecider is the slice of the PTO service these tools need. The service
// itself owns the ledger, the overlap check and the driver notification, none
// of which a tool has any business reimplementing.
type PtoDecider interface {
	Approve(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	Reject(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
	Cancel(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*worker.WorkerPTO, error)
}

const PtoIDNote = "The id of the time-off request, from list_time_off. " +
	"A worker id is not a request id."
