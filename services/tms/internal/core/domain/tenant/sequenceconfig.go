package tenant

import (
	"context"
	"fmt"
	"slices"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var AllowedSeparators = []string{"-", "_", "/", "."}

type SequenceConfig struct {
	bun.BaseModel `bun:"table:sequence_configs,alias:sqcfg" json:"-"`

	ID                      pulid.ID     `json:"id"                      bun:"id,pk,type:VARCHAR(100),notnull"`
	OrganizationID          pulid.ID     `json:"organizationId"          bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID          pulid.ID     `json:"businessUnitId"          bun:"business_unit_id,type:VARCHAR(100),notnull"`
	SequenceType            SequenceType `json:"sequenceType"            bun:"sequence_type,notnull"`
	Prefix                  string       `json:"prefix"                  bun:"prefix,type:VARCHAR(20),notnull"`
	IncludeYear             bool         `json:"includeYear"             bun:"include_year,type:BOOLEAN,notnull"`
	YearDigits              int16        `json:"yearDigits"              bun:"year_digits,type:SMALLINT,notnull"`
	IncludeMonth            bool         `json:"includeMonth"            bun:"include_month,type:BOOLEAN,notnull"`
	IncludeWeekNumber       bool         `json:"includeWeekNumber"       bun:"include_week_number,type:BOOLEAN,notnull"`
	IncludeDay              bool         `json:"includeDay"              bun:"include_day,type:BOOLEAN,notnull"`
	SequenceDigits          int16        `json:"sequenceDigits"          bun:"sequence_digits,type:SMALLINT,notnull"`
	IncludeLocationCode     bool         `json:"includeLocationCode"     bun:"include_location_code,type:BOOLEAN,notnull"`
	IncludeRandomDigits     bool         `json:"includeRandomDigits"     bun:"include_random_digits,type:BOOLEAN,notnull"`
	RandomDigitsCount       int16        `json:"randomDigitsCount"       bun:"random_digits_count,type:SMALLINT,notnull"`
	IncludeCheckDigit       bool         `json:"includeCheckDigit"       bun:"include_check_digit,type:BOOLEAN,notnull"`
	IncludeBusinessUnitCode bool         `json:"includeBusinessUnitCode" bun:"include_business_unit_code,type:BOOLEAN,notnull"`
	UseSeparators           bool         `json:"useSeparators"           bun:"use_separators,type:BOOLEAN,notnull"`
	SeparatorChar           string       `json:"separatorChar"           bun:"separator_char,type:VARCHAR(2),notnull"`
	AllowCustomFormat       bool         `json:"allowCustomFormat"       bun:"allow_custom_format,type:BOOLEAN,notnull"`
	CustomFormat            string       `json:"customFormat"            bun:"custom_format,type:TEXT,notnull"`
	Version                 int64        `json:"version"                 bun:"version,type:BIGINT,notnull"`
	CreatedAt               int64        `json:"createdAt"               bun:"created_at,notnull"`
	UpdatedAt               int64        `json:"updatedAt"               bun:"updated_at,notnull"`
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
