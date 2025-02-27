package pronumberconfig

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/pronumbergen"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type ProNumberConfig struct {
	bun.BaseModel `bun:"table:pro_number_configs,alias:pnc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull" json:"organizationId"`

	// Configuration Settings
	Prefix              string `json:"prefix" bun:"prefix,type:VARCHAR(5),notnull"`
	IncludeYear         bool   `json:"includeYear" bun:"include_year,type:BOOLEAN,notnull,default:false"`
	YearDigits          int    `json:"yearDigits" bun:"year_digits,type:INTEGER,notnull,default:2"`
	IncludeMonth        bool   `json:"includeMonth" bun:"include_month,type:BOOLEAN,notnull,default:false"`
	SequenceDigits      int    `json:"sequenceDigits" bun:"sequence_digits,type:INTEGER,notnull,default:4"`
	IncludeLocationCode bool   `json:"includeLocationCode" bun:"include_location_code,type:BOOLEAN,notnull,default:false"`
	LocationCode        string `json:"locationCode" bun:"location_code,type:VARCHAR(10),notnull"`
	IncludeRandomDigits bool   `json:"includeRandomDigits" bun:"include_random_digits,type:BOOLEAN,notnull,default:false"`
	RandomDigitsCount   int    `json:"randomDigitsCount" bun:"random_digits_count,type:INTEGER,notnull,default:4"`
	IsActive            bool   `json:"isActive" bun:"is_active,type:BOOLEAN,notnull,default:true"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (pnc *ProNumberConfig) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, pnc,
		// Prefix Validation
		validation.Field(&pnc.Prefix,
			validation.Required.Error("Prefix is required"),
			validation.Length(1, 5).Error("Prefix must be between 1 and 5 characters"),
		),

		// YearDigits Validation
		validation.Field(&pnc.YearDigits,
			validation.When(pnc.IncludeYear,
				validation.Required.Error("Year digits is required when including year"),
				validation.Min(1).Error("Year digits must be at least 1"),
				validation.Max(4).Error("Year digits must be at most 4"),
			),
		),

		// SequenceDigits Validation
		validation.Field(&pnc.SequenceDigits,
			validation.Required.Error("Sequence digits is required"),
			validation.Min(1).Error("Sequence digits must be at least 1"),
			validation.Max(10).Error("Sequence digits must be at most 10"),
		),

		// LocationCode Validation
		validation.Field(&pnc.LocationCode,
			validation.When(pnc.IncludeLocationCode,
				validation.Required.Error("Location code is required when including location code"),
				validation.Length(1, 10).Error("Location code must be between 1 and 10 characters"),
			),
		),

		// RandomDigitsCount Validation
		validation.Field(&pnc.RandomDigitsCount,
			validation.When(pnc.IncludeRandomDigits,
				validation.Required.Error("Random digits count is required when including random digits"),
				validation.Min(1).Error("Random digits count must be at least 1"),
				validation.Max(10).Error("Random digits count must be at most 10"),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (pnc *ProNumberConfig) GetID() string {
	return pnc.ID.String()
}

func (pnc *ProNumberConfig) GetTableName() string {
	return "pro_number_configs"
}

func (pnc *ProNumberConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if pnc.ID.IsNil() {
			pnc.ID = pulid.MustNew("pnc_")
		}

		pnc.CreatedAt = now
	case *bun.UpdateQuery:
		pnc.UpdatedAt = now
	}

	return nil
}

// ToProNumberFormat converts the config to a ProNumberFormat
func (p *ProNumberConfig) ToProNumberFormat() *pronumbergen.ProNumberFormat {
	return &pronumbergen.ProNumberFormat{
		Prefix:              p.Prefix,
		IncludeYear:         p.IncludeYear,
		YearDigits:          p.YearDigits,
		IncludeMonth:        p.IncludeMonth,
		SequenceDigits:      p.SequenceDigits,
		IncludeLocationCode: p.IncludeLocationCode,
		LocationCode:        p.LocationCode,
		IncludeRandomDigits: p.IncludeRandomDigits,
		RandomDigitsCount:   p.RandomDigitsCount,
	}
}
