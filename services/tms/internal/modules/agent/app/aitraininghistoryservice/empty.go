package aitraininghistoryservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aitraining"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aitrainingports"
	"github.com/emoss08/trenova/pkg/pagination"
)

type Empty struct{}

var _ aitrainingports.AITrainingHistoryService = Empty{}

func NewEmpty() aitrainingports.AITrainingHistoryService {
	return Empty{}
}

func (Empty) ListHistory(
	context.Context,
	pagination.TenantInfo,
) ([]*aitraining.ExportHistoryEntry, error) {
	return []*aitraining.ExportHistoryEntry{}, nil
}
