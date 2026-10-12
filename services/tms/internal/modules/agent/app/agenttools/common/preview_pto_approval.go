package common

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerports"
)

type PtoApprover interface {
	PtoDecider
	PreviewApprove(
		ctx context.Context,
		req *workerports.UpdatePTOStatusRequest,
	) (*workerports.WorkerPTOTransitionPreview, error)
}
