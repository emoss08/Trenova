package apikey

import (
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type UsageDaily struct {
	keel.Table `keel:"api_key_usage_daily"`

	APIKeyID       pulid.ID  `json:"apiKeyId" keel:"api_key_id,pk"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,tenant"`
	UsageDate      time.Time `json:"usageDate" keel:"usage_date,pk"`
	RequestCount   int64     `json:"requestCount" keel:"request_count"`
}

func (u *UsageDaily) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(u,
		validation.Field(&u.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&u.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&u.APIKeyID, validation.Required.Error("API key is required")),
		validation.Field(&u.UsageDate, validation.Required.Error("Usage date is required")),
		validation.Field(&u.RequestCount,
			validation.Min(int64(0)).Error("Request count cannot be negative"),
		),
	))
}
