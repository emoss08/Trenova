package extractionshadow

import (
	"context"
	"fmt"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	DefaultSamplePercent = 10
	MinSamplePercent     = 1
	MaxSamplePercent     = 100
	DefaultDailyLimit    = 200
	MinDailyLimit        = 1
	MaxDailyLimit        = 5000
)

type ShadowSettings struct {
	keel.Table `keel:"extraction_shadow_settings"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:exss_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`

	Enabled       bool      `json:"enabled" keel:"enabled"`
	ProviderID    *pulid.ID `json:"providerId" keel:"provider_id"`
	SamplePercent int       `json:"samplePercent" keel:"sample_percent"`
	DailyLimit    int       `json:"dailyLimit" keel:"daily_limit"`

	UpdatedByID *pulid.ID `json:"updatedById" keel:"updated_by_id"`
	Version     int64     `json:"version" keel:"version,version"`
	CreatedAt   int64     `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt   int64     `json:"updatedAt" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"-"`
	Organization *tenant.Organization `json:"-"`
}

func DefaultSettings(orgID, buID pulid.ID) *ShadowSettings {
	return &ShadowSettings{
		OrganizationID: orgID,
		BusinessUnitID: buID,
		SamplePercent:  DefaultSamplePercent,
		DailyLimit:     DefaultDailyLimit,
	}
}

func (s *ShadowSettings) CandidateID() pulid.ID {
	if s == nil || !s.Enabled || s.ProviderID == nil {
		return pulid.Nil
	}

	return *s.ProviderID
}

func (s *ShadowSettings) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID, validation.Required.Error("Organization is required")),
		validation.Field(&s.BusinessUnitID, validation.Required.Error("Business unit is required")),
		validation.Field(&s.SamplePercent,
			validation.Min(MinSamplePercent).Error(
				fmt.Sprintf("Sample between %d and %d percent of extractions", MinSamplePercent, MaxSamplePercent),
			),
			validation.Max(MaxSamplePercent).Error(
				fmt.Sprintf("Sample between %d and %d percent of extractions", MinSamplePercent, MaxSamplePercent),
			),
		),
		validation.Field(&s.DailyLimit,
			validation.Min(MinDailyLimit).Error(
				fmt.Sprintf("The daily limit must be between %d and %d extractions", MinDailyLimit, MaxDailyLimit),
			),
			validation.Max(MaxDailyLimit).Error(
				fmt.Sprintf("The daily limit must be between %d and %d extractions", MinDailyLimit, MaxDailyLimit),
			),
		),
	))

	if s.Enabled && (s.ProviderID == nil || s.ProviderID.IsNil()) {
		multiErr.Add(
			"providerId",
			errortypes.ErrRequired,
			"Choose the AI provider to shadow production with",
		)
	}
}

func (s *ShadowSettings) GetID() pulid.ID { return s.ID }

func (s *ShadowSettings) GetTableName() string { return "extraction_shadow_settings" }

func (s *ShadowSettings) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *ShadowSettings) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }

func (s *ShadowSettings) BeforeCreate(_ context.Context) error {
	if s.SamplePercent == 0 {
		s.SamplePercent = DefaultSamplePercent
	}
	if s.DailyLimit == 0 {
		s.DailyLimit = DefaultDailyLimit
	}

	return nil
}
