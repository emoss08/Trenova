package billingqueuefilterpreset

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*BillingQueueFilterPreset)(nil)
	_ validationframework.TenantedEntity = (*BillingQueueFilterPreset)(nil)
)

type BillingQueueFilterPreset struct {
	bun.BaseModel `bun:"table:billing_queue_filter_presets,alias:bqfp" json:"-"`

	ID             pulid.ID       `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID       `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull,pk"`
	BusinessUnitID pulid.ID       `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull,pk"`
	UserID         pulid.ID       `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	Name           string         `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Filters        map[string]any `json:"filters"        bun:"filters,type:JSONB,notnull,default:'{}'"`
	IsDefault      bool           `json:"isDefault"      bun:"is_default,type:BOOLEAN,notnull,default:false"`
	Version        int64          `json:"version"        bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt      int64          `json:"createdAt"      bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64          `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (bp *BillingQueueFilterPreset) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		bp,
		validation.Field(&bp.Name, validation.Required.Error("Name is required")),
		validation.Field(
			&bp.Name,
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
	)
	if err != nil {
		if validationErrs, ok := errors.AsType[validation.Errors](err); ok {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (bp *BillingQueueFilterPreset) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if bp.ID.IsNil() {
			bp.ID = pulid.MustNew("bqfp_")
		}
		bp.CreatedAt = now
	case *bun.UpdateQuery:
		bp.UpdatedAt = now
	}

	return nil
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
