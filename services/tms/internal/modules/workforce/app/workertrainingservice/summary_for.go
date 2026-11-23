package workertrainingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/workforce/domain/worker"
	"github.com/emoss08/trenova/internal/modules/workforce/domain/workertraining"
	"github.com/emoss08/trenova/pkg/pagination"
)

func (s *Service) SummaryFor(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
	wrk *worker.Worker,
) (*workertraining.WorkerTrainingSummary, error) {
	return s.summarize(ctx, tenantInfo, wrk)
}
