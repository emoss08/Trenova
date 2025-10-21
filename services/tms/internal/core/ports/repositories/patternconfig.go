package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetPatternConfigRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type PatternConfigRepository interface {
	GetAll(ctx context.Context) ([]*dedicatedlane.PatternConfig, error)
	GetByOrgID(
		ctx context.Context,
		req GetPatternConfigRequest,
	) (*dedicatedlane.PatternConfig, error)
	Update(
		ctx context.Context,
		pc *dedicatedlane.PatternConfig,
	) (*dedicatedlane.PatternConfig, error)
}
