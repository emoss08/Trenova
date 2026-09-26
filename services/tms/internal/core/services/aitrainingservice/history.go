package aitrainingservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"go.uber.org/fx"
)

var _ services.AITrainingHistoryService = (*History)(nil)

const historyLimit = 100

type HistoryParams struct {
	fx.In

	Records repositories.AITrainingRecordRepository
}

type History struct {
	records repositories.AITrainingRecordRepository
}

func NewHistory(p HistoryParams) services.AITrainingHistoryService {
	return &History{records: p.Records}
}

func (h *History) ListHistory(
	ctx context.Context,
	tenantInfo pagination.TenantInfo,
) ([]*aitraining.ExportHistoryEntry, error) {
	return h.records.ListHistory(ctx, repositories.ListAITrainingHistoryRequest{
		TenantInfo: tenantInfo,
		Limit:      historyLimit,
	})
}
