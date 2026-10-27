package billingqueuefilterpreset

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*BillingQueueFilterPreset)(nil)
)

type BillingQueueFilterPreset struct {
	keel.Table `keel:"billing_queue_filter_presets"`

	ID             pulid.ID       `json:"id"             keel:"id,pk,autoid:bqfp_"`
	OrganizationID pulid.ID       `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	UserID         pulid.ID       `json:"userId"         keel:"user_id"`
	Name           string         `json:"name"           keel:"name"`
	Filters        map[string]any `json:"filters"        keel:"filters,jsonb"`
	IsDefault      bool           `json:"isDefault"      keel:"is_default"`
	Version        int64          `json:"version"        keel:"version,version"`
	CreatedAt      int64          `json:"createdAt"      keel:"created_at,autocreate"`
	UpdatedAt      int64          `json:"updatedAt"      keel:"updated_at,autoupdate"`
}

func (bp *BillingQueueFilterPreset) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		bp,
		validation.Field(&bp.Name, validation.Required.Error("Name is required")),
		validation.Field(
			&bp.Name,
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
	))
}

func (bp *BillingQueueFilterPreset) GetID() pulid.ID {
	return bp.ID
}

func (bp *BillingQueueFilterPreset) GetOrganizationID() pulid.ID {
	return bp.OrganizationID
}

func (bp *BillingQueueFilterPreset) GetBusinessUnitID() pulid.ID {
	return bp.BusinessUnitID
}

func (bp *BillingQueueFilterPreset) GetTableName() string {
	return "billing_queue_filter_presets"
}
