package telematics

import (
	"context"
	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type FeedState struct {
	keel.Table `keel:"telematics_feed_states"`

	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	Provider       string   `json:"provider" keel:"provider,pk"`
	FeedType       FeedType `json:"feedType" keel:"feed_type,pk"`
	Cursor         string   `json:"cursor" keel:"cursor,nullzero"`
	LastPolledAt   int64    `json:"lastPolledAt" keel:"last_polled_at,nullzero"`
	LastSuccessAt  int64    `json:"lastSuccessAt" keel:"last_success_at,nullzero"`
	FailureCount   int      `json:"failureCount" keel:"failure_count"`
	LastError      string   `json:"lastError" keel:"last_error,nullzero"`
}

func (f *FeedState) BeforeCreate(_ context.Context) error {
	if f.Provider == "" {
		f.Provider = "Samsara"
	}

	return nil
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
