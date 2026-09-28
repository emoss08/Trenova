package extractionshadow

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*ShadowSettings)(nil)

const (
	DefaultSamplePercent = 10
	MinSamplePercent     = 1
	MaxSamplePercent     = 100
	DefaultDailyLimit    = 200
	MinDailyLimit        = 1
	MaxDailyLimit        = 5000
)

type ShadowSettings struct {
	bun.BaseModel `bun:"table:extraction_shadow_settings,alias:exss" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`

	Enabled       bool      `json:"enabled"       bun:"enabled,type:BOOLEAN,notnull,default:false"`
	ProviderID    *pulid.ID `json:"providerId"    bun:"provider_id,type:VARCHAR(100),nullzero"`
	SamplePercent int       `json:"samplePercent" bun:"sample_percent,type:INTEGER,notnull,default:10"`
	DailyLimit    int       `json:"dailyLimit"    bun:"daily_limit,type:INTEGER,notnull,default:200"`

	UpdatedByID *pulid.ID `json:"updatedById" bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version     int64     `json:"version"     bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt   int64     `json:"createdAt"   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt   int64     `json:"updatedAt"   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	BusinessUnit *tenant.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *tenant.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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

func (s *ShadowSettings) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("exss_")
		}
		if s.CreatedAt == 0 {
			s.CreatedAt = now
		}
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
