package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/billingqueuefilterpreset"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type ListBillingQueueFilterPresetsRequest struct {
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type DeleteBillingQueueFilterPresetRequest struct {
	PresetID   pulid.ID
	UserID     pulid.ID
	TenantInfo pagination.TenantInfo
}

type BillingQueueFilterPresetRepository interface {
	ListByUserID(
		ctx context.Context,
		req *ListBillingQueueFilterPresetsRequest,
	) ([]*billingqueuefilterpreset.BillingQueueFilterPreset, error)
	Create(
		ctx context.Context,
		entity *billingqueuefilterpreset.BillingQueueFilterPreset,
	) (*billingqueuefilterpreset.BillingQueueFilterPreset, error)
	Update(
		ctx context.Context,
		entity *billingqueuefilterpreset.BillingQueueFilterPreset,
	) (*billingqueuefilterpreset.BillingQueueFilterPreset, error)
	Delete(
		ctx context.Context,
		req *DeleteBillingQueueFilterPresetRequest,
	) error
}
