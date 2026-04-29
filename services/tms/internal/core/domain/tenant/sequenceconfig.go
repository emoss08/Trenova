package tenant

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var AllowedSeparators = []string{"-", "_", "/", "."}

const (
	LocationCodeComponentName       = LocationCodeComponent("name")
	LocationCodeComponentCity       = LocationCodeComponent("city")
	LocationCodeComponentState      = LocationCodeComponent("state")
	LocationCodeComponentPostalCode = LocationCodeComponent("postal_code")
	LocationCodeCasingUpper         = LocationCodeCasing("upper")
	LocationCodeCasingLower         = LocationCodeCasing("lower")
	MaxLocationCodeLength           = 32
)

type LocationCodeComponent string

type LocationCodeCasing string

type SequenceConfig struct {
	bun.BaseModel `bun:"table:sequence_configs,alias:sqcfg" json:"-"`

	ID                      pulid.ID              `json:"id"                             bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID              `json:"organizationId"                 bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID              `json:"businessUnitId"                 bun:"business_unit_id,type:VARCHAR(100),notnull"`
	SequenceType            SequenceType          `json:"sequenceType"                   bun:"sequence_type,notnull"`
	Prefix                  string                `json:"prefix"                         bun:"prefix,type:VARCHAR(20),notnull"`
	IncludeYear             bool                  `json:"includeYear"                    bun:"include_year,type:BOOLEAN,notnull"`
	YearDigits              int16                 `json:"yearDigits"                     bun:"year_digits,type:SMALLINT,notnull"`
	IncludeMonth            bool                  `json:"includeMonth"                   bun:"include_month,type:BOOLEAN,notnull"`
	IncludeWeekNumber       bool                  `json:"includeWeekNumber"              bun:"include_week_number,type:BOOLEAN,notnull"`
	IncludeDay              bool                  `json:"includeDay"                     bun:"include_day,type:BOOLEAN,notnull"`
	SequenceDigits          int16                 `json:"sequenceDigits"                 bun:"sequence_digits,type:SMALLINT,notnull"`
	IncludeLocationCode     bool                  `json:"includeLocationCode"            bun:"include_location_code,type:BOOLEAN,notnull"`
	IncludeRandomDigits     bool                  `json:"includeRandomDigits"            bun:"include_random_digits,type:BOOLEAN,notnull"`
	RandomDigitsCount       int16                 `json:"randomDigitsCount"              bun:"random_digits_count,type:SMALLINT,notnull"`
	IncludeCheckDigit       bool                  `json:"includeCheckDigit"              bun:"include_check_digit,type:BOOLEAN,notnull"`
	IncludeBusinessUnitCode bool                  `json:"includeBusinessUnitCode"        bun:"include_business_unit_code,type:BOOLEAN,notnull"`
	UseSeparators           bool                  `json:"useSeparators"                  bun:"use_separators,type:BOOLEAN,notnull"`
	SeparatorChar           string                `json:"separatorChar"                  bun:"separator_char,type:VARCHAR(2),notnull"`
	AllowCustomFormat       bool                  `json:"allowCustomFormat"              bun:"allow_custom_format,type:BOOLEAN,notnull"`
	CustomFormat            string                `json:"customFormat"                   bun:"custom_format,type:TEXT,notnull"`
	LocationCodeStrategy    *LocationCodeStrategy `json:"locationCodeStrategy,omitempty" bun:"location_code_strategy,type:JSONB,nullzero"`
	Version                 int64                 `json:"version"                        bun:"version,type:BIGINT,notnull"`
	CreatedAt               int64                 `json:"createdAt"                      bun:"created_at,notnull"`
	UpdatedAt               int64                 `json:"updatedAt"                      bun:"updated_at,notnull"`
}

type LocationCodeStrategy struct {
	Components     []LocationCodeComponent `json:"components"`
	ComponentWidth int16                   `json:"componentWidth"`
	SequenceDigits int16                   `json:"sequenceDigits"`
	Separator      string                  `json:"separator"`
	Casing         LocationCodeCasing      `json:"casing"`
	FallbackPrefix string                  `json:"fallbackPrefix"`
}

func DefaultLocationCodeStrategy() *LocationCodeStrategy {
	return &LocationCodeStrategy{
		Components: []LocationCodeComponent{
			LocationCodeComponentName,
			LocationCodeComponentCity,
			LocationCodeComponentState,
		},
		ComponentWidth: 3,
		SequenceDigits: 3,
		Separator:      "-",
		Casing:         LocationCodeCasingUpper,
		FallbackPrefix: "LOC",
	}
}

func EffectiveLocationCodeStrategy(strategy *LocationCodeStrategy) *LocationCodeStrategy {
	defaults := DefaultLocationCodeStrategy()
	if strategy == nil {
		return defaults
	}

	if strategy.Components != nil {
		defaults.Components = append([]LocationCodeComponent(nil), strategy.Components...)
	}
	if strategy.ComponentWidth > 0 {
		defaults.ComponentWidth = strategy.ComponentWidth
	}
	if strategy.SequenceDigits > 0 {
		defaults.SequenceDigits = strategy.SequenceDigits
	}
	defaults.Separator = strategy.Separator
	if strings.TrimSpace(string(strategy.Casing)) != "" {
		defaults.Casing = LocationCodeCasing(strings.TrimSpace(string(strategy.Casing)))
	}
	if strings.TrimSpace(strategy.FallbackPrefix) != "" {
		defaults.FallbackPrefix = strings.TrimSpace(strategy.FallbackPrefix)
	}

	return defaults
}

func (s *LocationCodeStrategy) Validate() error {
	strategy := EffectiveLocationCodeStrategy(s)
	if strategy.Separator != "" && !slices.Contains(AllowedSeparators, strategy.Separator) {
		return fmt.Errorf("separator %q is not allowed", strategy.Separator)
	}
	if strategy.Casing != LocationCodeCasingUpper && strategy.Casing != LocationCodeCasingLower {
		return fmt.Errorf(
			"casing must be either %q or %q",
			LocationCodeCasingUpper,
			LocationCodeCasingLower,
		)
	}
	if len(strategy.Components) == 0 {
		return fmt.Errorf("at least one location code component is required")
	}
	for _, component := range strategy.Components {
		if !isAllowedLocationCodeComponent(component) {
			return fmt.Errorf("component %q is not supported", component)
		}
	}
	if strings.TrimSpace(strategy.FallbackPrefix) == "" {
		return fmt.Errorf("fallback prefix is required")
	}
	parts := len(strategy.Components) + 1
	length := len(strategy.Separator)*(parts-1) +
		int(strategy.ComponentWidth)*len(strategy.Components) +
		int(strategy.SequenceDigits)
	if length > MaxLocationCodeLength {
		return fmt.Errorf("location code format cannot exceed %d characters", MaxLocationCodeLength)
	}

	fallback := strings.TrimSpace(strategy.FallbackPrefix)
	if stringutils.NormalizeIdentifier(fallback) == "" {
		return fmt.Errorf("fallback prefix must contain letters or digits")
	}
	if len([]rune(fallback)) > MaxLocationCodeLength {
		return fmt.Errorf("location code format cannot exceed %d characters", MaxLocationCodeLength)
	}

	return validation.ValidateStruct(
		strategy,
		validation.Field(
			&strategy.ComponentWidth,
			validation.Required,
			validation.Min(1),
			validation.Max(10),
		),
		validation.Field(
			&strategy.SequenceDigits,
			validation.Required,
			validation.Min(1),
			validation.Max(10),
		),
		validation.Field(
			&strategy.FallbackPrefix,
			validation.Required,
			validation.Length(1, MaxLocationCodeLength),
		),
	)
}

func isAllowedLocationCodeComponent(component LocationCodeComponent) bool {
	switch component {
	case LocationCodeComponentName,
		LocationCodeComponentCity,
		LocationCodeComponentState,
		LocationCodeComponentPostalCode:
		return true
	default:
		return false
	}
}

type SequenceConfigDocument struct {
	OrganizationID pulid.ID          `json:"organizationId"`
	BusinessUnitID pulid.ID          `json:"businessUnitId"`
	Configs        []*SequenceConfig `json:"configs"`
}

func (sc *SequenceConfig) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if sc.ID.IsNil() {
			sc.ID = pulid.MustNew("sc_")
		}

		sc.CreatedAt = now
	case *bun.UpdateQuery:
		sc.UpdatedAt = now
	}

	return nil
}

type SequenceFormat struct {
	Type                    SequenceType
	Prefix                  string
	IncludeYear             bool
	YearDigits              int
	IncludeMonth            bool
	IncludeWeekNumber       bool
	IncludeDay              bool
	SequenceDigits          int
	IncludeLocationCode     bool
	LocationCode            string
	IncludeRandomDigits     bool
	RandomDigitsCount       int
	IncludeCheckDigit       bool
	IncludeBusinessUnitCode bool
	BusinessUnitCode        string
	UseSeparators           bool
	SeparatorChar           string
	AllowCustomFormat       bool
	CustomFormat            string
}

func (f *SequenceFormat) Validate() error {
	if f.UseSeparators && f.SeparatorChar != "" {
		if !slices.Contains(AllowedSeparators, f.SeparatorChar) {
			return fmt.Errorf("separator %q is not allowed", f.SeparatorChar)
		}
	}

	return validation.ValidateStruct(
		f,
		validation.Field(&f.Type, validation.Required),
		validation.Field(
			&f.SequenceDigits,
			validation.Required,
			validation.Min(1),
			validation.Max(10),
		),
		validation.Field(
			&f.YearDigits,
			validation.When(f.IncludeYear, validation.Min(2), validation.Max(4)),
		),
		validation.Field(
			&f.LocationCode,
			validation.When(f.IncludeLocationCode, validation.Required),
		),
		validation.Field(
			&f.RandomDigitsCount,
			validation.When(f.IncludeRandomDigits, validation.Min(1), validation.Max(10)),
		),
		validation.Field(
			&f.BusinessUnitCode,
			validation.When(f.IncludeBusinessUnitCode, validation.Required),
		),
		validation.Field(&f.SeparatorChar, validation.When(f.UseSeparators, validation.Required)),
		validation.Field(
			&f.CustomFormat,
			validation.When(f.AllowCustomFormat, validation.Required),
		),
	)
}

type Sequence struct {
	bun.BaseModel `bun:"table:sequences,alias:seq"`

	ID              pulid.ID     `bun:"id,pk,type:VARCHAR(100)"`
	SequenceType    SequenceType `bun:"sequence_type,notnull"`
	OrganizationID  pulid.ID     `bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID     `bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Year            int16        `bun:"year,notnull"`
	Month           int16        `bun:"month,notnull"`
	CurrentSequence int64        `bun:"current_sequence,notnull"`
	LastGenerated   string       `bun:"last_generated,notnull"`
	Version         int64        `bun:"version,type:BIGINT,notnull"`
	CreatedAt       int64        `bun:"created_at,notnull"`
	UpdatedAt       int64        `bun:"updated_at,notnull"`
}
