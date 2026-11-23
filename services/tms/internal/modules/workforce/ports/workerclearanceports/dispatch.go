package workerclearanceports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workercompliance"
	"github.com/emoss08/trenova/pkg/pagination"
)

type DispatchRequest struct {
	TenantInfo pagination.TenantInfo
	Worker     *worker.Worker
	Now        int64
	Probes     []workercompliance.DispatchProbe
}

type DispatchCheck interface {
	Verdicts(
		ctx context.Context,
		req *DispatchRequest,
	) (*workercompliance.DispatchVerdicts, error)
}
