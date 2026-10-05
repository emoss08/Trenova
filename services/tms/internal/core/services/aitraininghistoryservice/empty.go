package aitraininghistoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aitraining"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Empty struct{}

var _ services.AITrainingHistoryService = Empty{}

func NewEmpty() services.AITrainingHistoryService {
	return Empty{}
}

func (Empty) ListHistory(
	context.Context,
	pagination.TenantInfo,
) ([]*aitraining.ExportHistoryEntry, error) {
	return []*aitraining.ExportHistoryEntry{}, nil
}
