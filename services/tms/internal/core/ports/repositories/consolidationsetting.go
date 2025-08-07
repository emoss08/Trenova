/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetConsolidationSettingRequest struct {
	OrgID  pulid.ID
	BuID   pulid.ID
	UserID pulid.ID
}

type ConsolidationSettingRepository interface {
	GetByOrgID(ctx context.Context, orgID pulid.ID) (*consolidation.ConsolidationSettings, error)
	Update(
		ctx context.Context,
		cs *consolidation.ConsolidationSettings,
	) (*consolidation.ConsolidationSettings, error)
}
