package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/pkg/types/pulid"
)

type PatternConfigRepository interface {
	GetAll(ctx context.Context) ([]*dedicatedlane.PatternConfig, error)
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*dedicatedlane.PatternConfig, error)
	Update(
		ctx context.Context,
		pc *dedicatedlane.PatternConfig,
	) (*dedicatedlane.PatternConfig, error)
}
