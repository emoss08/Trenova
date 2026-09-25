package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/pkg/pagination"
)

type PurgeAICorrectionsRequest struct {
	TenantInfo pagination.TenantInfo
	Before     int64
	Limit      int
}

type AICorrectionRepository interface {
	Upsert(ctx context.Context, entity *aicorrection.Correction) (*aicorrection.Correction, error)
	PurgeBefore(ctx context.Context, req PurgeAICorrectionsRequest) (int64, error)
}
