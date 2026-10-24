package telematics

import (
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type FeedState struct {
	bun.BaseModel `bun:"table:telematics_feed_states,alias:tfst" json:"-"`
	keel.Table    `keel:"telematics_feed_states"`

	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull" keel:"business_unit_id,pk,tenant"`
	Provider       string   `json:"provider"       bun:"provider,pk,type:VARCHAR(32),notnull,default:'Samsara'" keel:"provider,pk"`
	FeedType       FeedType `json:"feedType"       bun:"feed_type,pk,type:VARCHAR(32),notnull" keel:"feed_type,pk"`
	Cursor         string   `json:"cursor"         bun:"cursor,type:TEXT,nullzero" keel:"cursor,nullzero"`
	LastPolledAt   int64    `json:"lastPolledAt"   bun:"last_polled_at,type:BIGINT,nullzero" keel:"last_polled_at,nullzero"`
	LastSuccessAt  int64    `json:"lastSuccessAt"  bun:"last_success_at,type:BIGINT,nullzero" keel:"last_success_at,nullzero"`
	FailureCount   int      `json:"failureCount"   bun:"failure_count,type:INT,notnull" keel:"failure_count"`
	LastError      string   `json:"lastError"      bun:"last_error,type:TEXT,nullzero" keel:"last_error,nullzero"`
}

func (f *FeedState) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(f,
		validation.Field(&f.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&f.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&f.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxProviderLength).
				Error("Provider cannot be longer than 32 characters"),
		),
		validation.Field(&f.FeedType,
			validation.Required.Error("Feed type is required"),
			domainvalidation.ValidEnum[FeedType]("Feed type is invalid"),
		),
		validation.Field(&f.FailureCount,
			validation.Min(0).Error("Failure count cannot be negative"),
		),
	))
}
