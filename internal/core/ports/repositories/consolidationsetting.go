package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/consolidation"
	"github.com/emoss08/trenova/pkg/types/pulid"
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
