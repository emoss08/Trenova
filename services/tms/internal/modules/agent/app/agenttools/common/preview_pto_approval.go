package common

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/ports/workerptoports"
)

type PtoApprover interface {
	PtoDecider
	PreviewApprove(
		ctx context.Context,
		req *workerptoports.UpdatePTOStatusRequest,
	) (*workerptoports.WorkerPTOTransitionPreview, error)
}
