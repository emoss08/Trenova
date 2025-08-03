/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/dedicatedlane"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
