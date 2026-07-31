package apikey

import (
	"time"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type UsageDaily struct {
	bun.BaseModel `bun:"table:api_key_usage_daily,alias:akud" json:"-"`

	APIKeyID       pulid.ID  `json:"apiKeyId"       bun:"api_key_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	UsageDate      time.Time `json:"usageDate"      bun:"usage_date,pk,type:date,notnull"`
	RequestCount   int64     `json:"requestCount"   bun:"request_count,notnull,default:0"`
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
