package seqgen

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

type FormatTemplate string

const (
	TemplateSimpleSequential FormatTemplate = "simple_sequential" // Just prefix and sequence: PRE0001
	TemplateYearMonth        FormatTemplate = "year_month"        // With year and month: PRE202401-0001
	TemplateFullDate         FormatTemplate = "full_date"         // Full date: PRE20240115-0001
	TemplateLocationBased    FormatTemplate = "location_based"    // With location: PRE-LAX-0001
	TemplateCheckDigit       FormatTemplate = "check_digit"       // With check digit: PRE0001-7
	TemplateRandom           FormatTemplate = "random"            // With random digits: PRE0001-ABC123
	TemplateComprehensive    FormatTemplate = "comprehensive"     // All components: PRE-2024-01-LAX-0001-ABC-7
)

var FormatTemplates = map[FormatTemplate]*Format{
	TemplateSimpleSequential: {
		Prefix:         "SEQ",
		SequenceDigits: 6,
	},
	TemplateYearMonth: {
		Prefix:         "YM",
		IncludeYear:    true,
		YearDigits:     4,
		IncludeMonth:   true,
		SequenceDigits: 4,
		UseSeparators:  true,
		SeparatorChar:  "-",
	},
	TemplateFullDate: {
		Prefix:         "FD",
		IncludeYear:    true,
		YearDigits:     4,
		IncludeMonth:   true,
		IncludeDay:     true,
		SequenceDigits: 4,
		UseSeparators:  true,
		SeparatorChar:  "-",
	},
	TemplateLocationBased: {
		Prefix:              "LOC",
		IncludeLocationCode: true,
		LocationCode:        "XXX", // Placeholder, should be replaced
		SequenceDigits:      5,
		UseSeparators:       true,
		SeparatorChar:       "-",
	},
	TemplateCheckDigit: {
		Prefix:            "CHK",
		SequenceDigits:    5,
		IncludeCheckDigit: true,
		UseSeparators:     true,
		SeparatorChar:     "-",
	},
	TemplateRandom: {
		Prefix:              "RND",
		SequenceDigits:      4,
		IncludeRandomDigits: true,
		RandomDigitsCount:   6,
		UseSeparators:       true,
		SeparatorChar:       "-",
	},
	TemplateComprehensive: {
		Prefix:                  "CMP",
		IncludeYear:             true,
		YearDigits:              4,
		IncludeMonth:            true,
		IncludeLocationCode:     true,
		LocationCode:            "XXX",
		SequenceDigits:          4,
		IncludeRandomDigits:     true,
		RandomDigitsCount:       3,
		IncludeCheckDigit:       true,
		IncludeBusinessUnitCode: true,
		BusinessUnitCode:        "01",
		UseSeparators:           true,
		SeparatorChar:           "-",
	},
}

func GetFormatFromTemplate(template FormatTemplate, overrides *Format) *Format {
	baseFormat, exists := FormatTemplates[template]
	if !exists {
		baseFormat = FormatTemplates[TemplateSimpleSequential]
	}

	if overrides != nil {
		baseFormat = setOverrideDefaults(baseFormat, overrides)
	}

	return baseFormat
}

func setOverrideDefaults(result, overrides *Format) *Format {
	if overrides.Type != "" {
		result.Type = overrides.Type
	}
	if overrides.Prefix != "" {
		result.Prefix = overrides.Prefix
	}
	if overrides.LocationCode != "" {
		result.LocationCode = overrides.LocationCode
	}
	if overrides.BusinessUnitCode != "" {
		result.BusinessUnitCode = overrides.BusinessUnitCode
	}
	if overrides.SequenceDigits > 0 {
		result.SequenceDigits = overrides.SequenceDigits
	}
	if overrides.YearDigits > 0 && result.IncludeYear {
		result.YearDigits = overrides.YearDigits
	}
	if overrides.RandomDigitsCount > 0 && result.IncludeRandomDigits {
		result.RandomDigitsCount = overrides.RandomDigitsCount
	}
	if overrides.SeparatorChar != "" && result.UseSeparators {
		result.SeparatorChar = overrides.SeparatorChar
	}
	if overrides.CustomFormat != "" {
		result.CustomFormat = overrides.CustomFormat
		result.AllowCustomFormat = true
	}

	return result
}

// DefaultShipmentProNumberFormat returns the default pro number format for shipments
// Generates sequences like: S241200011234567890
// Format: S (prefix) + 24 (year) + 12 (month) + 0001 (sequence) + 123456 (random)
func DefaultShipmentProNumberFormat() *Format {
	return &Format{
		Type:                    SequenceTypeProNumber,
		Prefix:                  "S",
		IncludeYear:             true,
		YearDigits:              2,
		IncludeMonth:            true,
		SequenceDigits:          4,
		IncludeLocationCode:     false, // Can be overridden if needed
		LocationCode:            "",    // Set by organization/business unit
		IncludeRandomDigits:     true,
		RandomDigitsCount:       6,
		IncludeCheckDigit:       false,
		IncludeBusinessUnitCode: false,
		BusinessUnitCode:        "",
		UseSeparators:           false, // No separators for cleaner pro numbers
		SeparatorChar:           "",
		IncludeWeekNumber:       false,
		IncludeDay:              false,
		AllowCustomFormat:       false,
		CustomFormat:            "",
	}
}

func DefaultConsolidationFormat() *Format {
	return &Format{
		Type:                    SequenceTypeConsolidation,
		Prefix:                  "C",
		IncludeYear:             true,
		YearDigits:              2,
		IncludeMonth:            true,
		SequenceDigits:          5,
		IncludeLocationCode:     false,
		LocationCode:            "",
		IncludeRandomDigits:     false,
		RandomDigitsCount:       0,
		IncludeCheckDigit:       true,
		IncludeBusinessUnitCode: true,
		BusinessUnitCode:        "01", // Should be overridden per business unit
		UseSeparators:           true,
		SeparatorChar:           "-",
		AllowCustomFormat:       false,
		CustomFormat:            "",
	}
}

type SequenceType string

const (
	SequenceTypeProNumber     = SequenceType("pro_number")
	SequenceTypeConsolidation = SequenceType("consolidation")
)

var AllowedSeparators = []string{"-", "_", "/", "."}

type Sequence struct {
	bun.BaseModel `bun:"table:sequences,alias:seq"`

	ID              pulid.ID     `bun:"id,pk,type:VARCHAR(100)"`
	SequenceType    SequenceType `bun:"sequence_type,notnull"`
	OrganizationID  pulid.ID     `bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID     `bun:"business_unit_id,type:VARCHAR(100)"`
	Year            int16        `bun:"year,notnull"`
	Month           int16        `bun:"month,notnull"`
	CurrentSequence int64        `bun:"current_sequence,notnull"`
	LastGenerated   string       `bun:"last_generated"` // Store last generated sequence for verification
	Version         int64        `bun:"version,type:BIGINT"`
	CreatedAt       int64        `bun:"created_at,notnull"`
	UpdatedAt       int64        `bun:"updated_at,notnull"`
}

func (s *Sequence) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("seq_")
		}
		s.CreatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}

type Format struct {
	Type                    SequenceType
	Prefix                  string
	IncludeYear             bool
	YearDigits              int
	IncludeMonth            bool
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
	IncludeWeekNumber       bool
	IncludeDay              bool
}

func (f *Format) Validate() error {
	if f.UseSeparators && f.SeparatorChar != "" {
		validSeparator := slices.Contains(AllowedSeparators, f.SeparatorChar)
		if !validSeparator {
			return fmt.Errorf(
				"separator character %q is not allowed. Must be one of: %v",
				f.SeparatorChar,
				AllowedSeparators,
			)
		}
	}

	return validation.ValidateStruct(
		f,
		validation.Field(&f.Type, validation.Required),
		validation.Field(&f.YearDigits,
			validation.When(f.IncludeYear, validation.Min(2), validation.Max(4))),
		validation.Field(
			&f.SequenceDigits,
			validation.Required,
			validation.Min(1),
			validation.Max(10),
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

type SequenceStore interface {
	GetNextSequence(ctx context.Context, req *SequenceRequest) (int64, error)
	GetNextSequenceBatch(ctx context.Context, req *SequenceRequest) ([]int64, error)
}

type SequenceRequest struct {
	Type  SequenceType
	OrgID pulid.ID
	BuID  pulid.ID
	Year  int
	Month int
	Count int // For batch requests
}

type FormatProvider interface {
	GetFormat(
		ctx context.Context,
		sequenceType SequenceType,
		orgID, buID pulid.ID,
	) (*Format, error)
}

type Generator interface {
	Generate(ctx context.Context, req *GenerateRequest) (string, error)
	GenerateBatch(ctx context.Context, req *GenerateRequest) ([]string, error)
	GenerateShipmentProNumber(ctx context.Context, orgID, buID pulid.ID) (string, error)
	GenerateShipmentProNumberBatch(
		ctx context.Context,
		orgID, buID pulid.ID,
		count int,
	) ([]string, error)
	ValidateSequence(sequence string, format *Format) error
	ParseSequence(sequence string, format *Format) (*SequenceComponents, error)
	ClearCache()
	SetCacheTTL(ttl time.Duration)
}

type GenerateRequest struct {
	Type   SequenceType
	OrgID  pulid.ID
	BuID   pulid.ID
	Count  int       // For batch generation (recommend 100-500 for high volume)
	Time   time.Time // Optional: override current time
	Format *Format   // Optional: override default format
}

type SequenceComponents struct {
	Original         string
	Prefix           string
	BusinessUnitCode string
	Year             string
	Month            string
	Week             string
	Day              string
	LocationCode     string
	Sequence         string
	RandomDigits     string
	CheckDigit       string
}
