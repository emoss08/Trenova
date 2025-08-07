/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package pronumberconfig

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/sequencegen"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

type ProNumberConfig struct {
	bun.BaseModel `bun:"table:pro_number_configs,alias:pnc" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull"               json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull"  json:"organizationId"`

	// Configuration Settings
	Prefix              string `json:"prefix"              bun:"prefix,type:VARCHAR(5),notnull"`
	IncludeYear         bool   `json:"includeYear"         bun:"include_year,type:BOOLEAN,notnull,default:false"`
	YearDigits          int    `json:"yearDigits"          bun:"year_digits,type:INTEGER,notnull,default:2"`
	IncludeMonth        bool   `json:"includeMonth"        bun:"include_month,type:BOOLEAN,notnull,default:false"`
	SequenceDigits      int    `json:"sequenceDigits"      bun:"sequence_digits,type:INTEGER,notnull,default:4"`
	IncludeLocationCode bool   `json:"includeLocationCode" bun:"include_location_code,type:BOOLEAN,notnull,default:false"`
	LocationCode        string `json:"locationCode"        bun:"location_code,type:VARCHAR(10),notnull"`
	IncludeRandomDigits bool   `json:"includeRandomDigits" bun:"include_random_digits,type:BOOLEAN,notnull,default:false"`
	RandomDigitsCount   int    `json:"randomDigitsCount"   bun:"random_digits_count,type:INTEGER,notnull,default:4"`

	// New configuration settings matching ProNumberFormat enhancements
	IncludeCheckDigit       bool   `json:"includeCheckDigit"       bun:"include_check_digit,type:BOOLEAN,notnull,default:false"`
	IncludeBusinessUnitCode bool   `json:"includeBusinessUnitCode" bun:"include_business_unit_code,type:BOOLEAN,notnull,default:false"`
	BusinessUnitCode        string `json:"businessUnitCode"        bun:"business_unit_code,type:VARCHAR(10),notnull"`
	UseSeparators           bool   `json:"useSeparators"           bun:"use_separators,type:BOOLEAN,notnull,default:false"`
	SeparatorChar           string `json:"separatorChar"           bun:"separator_char,type:VARCHAR(1),notnull"`
	IncludeWeekNumber       bool   `json:"includeWeekNumber"       bun:"include_week_number,type:BOOLEAN,notnull,default:false"`
	IncludeDay              bool   `json:"includeDay"              bun:"include_day,type:BOOLEAN,notnull,default:false"`
	AllowCustomFormat       bool   `json:"allowCustomFormat"       bun:"allow_custom_format,type:BOOLEAN,notnull,default:false"`
	CustomFormat            string `json:"customFormat"            bun:"custom_format,type:VARCHAR(100),notnull"`

	IsActive bool `json:"isActive" bun:"is_active,type:BOOLEAN,notnull,default:true"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT"                                                                  json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
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
			validation.When(
				pnc.IncludeRandomDigits,
				validation.Required.Error(
					"Random digits count is required when including random digits",
				),
				validation.Min(1).Error("Random digits count must be at least 1"),
				validation.Max(10).Error("Random digits count must be at most 10"),
			),
		),

		// BusinessUnitCode Validation
		validation.Field(&pnc.BusinessUnitCode,
			validation.When(
				pnc.IncludeBusinessUnitCode,
				validation.Required.Error(
					"Business unit code is required when including business unit code",
				),
				validation.Length(1, 10).
					Error("Business unit code must be between 1 and 10 characters"),
			),
		),

		// SeparatorChar Validation
		validation.Field(&pnc.SeparatorChar,
			validation.When(pnc.UseSeparators,
				validation.Required.Error("Separator character is required when using separators"),
				validation.Length(1, 1).Error("Separator character must be exactly 1 character"),
			),
		),

		// CustomFormat Validation
		validation.Field(&pnc.CustomFormat,
			validation.When(
				pnc.AllowCustomFormat,
				validation.Required.Error("Custom format is required when allowing custom format"),
				validation.Length(1, 100).
					Error("Custom format must be between 1 and 100 characters"),
			),
		),

		// Conflicting options
		validation.Field(&pnc.IncludeMonth,
			validation.When(pnc.IncludeWeekNumber,
				validation.NotIn(true).Error("Cannot include both month and week number"),
			),
		),

		validation.Field(&pnc.IncludeWeekNumber,
			validation.When(pnc.IncludeMonth,
				validation.NotIn(true).Error("Cannot include both week number and month"),
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

// ToSequenceFormat converts the config to a SequenceFormat
func (pnc *ProNumberConfig) ToSequenceFormat() *sequencegen.SequenceFormat {
	return &sequencegen.SequenceFormat{
		Prefix:                  pnc.Prefix,
		IncludeYear:             pnc.IncludeYear,
		YearDigits:              pnc.YearDigits,
		IncludeMonth:            pnc.IncludeMonth,
		SequenceDigits:          pnc.SequenceDigits,
		IncludeLocationCode:     pnc.IncludeLocationCode,
		LocationCode:            pnc.LocationCode,
		IncludeRandomDigits:     pnc.IncludeRandomDigits,
		RandomDigitsCount:       pnc.RandomDigitsCount,
		IncludeCheckDigit:       pnc.IncludeCheckDigit,
		IncludeBusinessUnitCode: pnc.IncludeBusinessUnitCode,
		BusinessUnitCode:        pnc.BusinessUnitCode,
		UseSeparators:           pnc.UseSeparators,
		SeparatorChar:           pnc.SeparatorChar,
		IncludeWeekNumber:       pnc.IncludeWeekNumber,
		IncludeDay:              pnc.IncludeDay,
		AllowCustomFormat:       pnc.AllowCustomFormat,
		CustomFormat:            pnc.CustomFormat,
	}
}
