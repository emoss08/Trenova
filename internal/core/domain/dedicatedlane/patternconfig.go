/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package dedicatedlane

import (
	"context"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/shopspring/decimal"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*PatternConfig)(nil)
	_ domain.Validatable        = (*PatternConfig)(nil)
)

type PatternConfig struct {
	bun.BaseModel `bun:"table:pattern_configs,alias:pc" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),pk,notnull"`

	// Configuration fields - storing as individual fields for better querying and validation
	Enabled               bool            `json:"enabled"               bun:"enabled,type:BOOLEAN,notnull,default:true"`
	MinFrequency          int64           `json:"minFrequency"          bun:"min_frequency,type:INTEGER,notnull,default:3"`
	AnalysisWindowDays    int64           `json:"analysisWindowDays"    bun:"analysis_window_days,type:INTEGER,notnull,default:90"`
	MinConfidenceScore    decimal.Decimal `json:"minConfidenceScore"    bun:"min_confidence_score,type:NUMERIC(5,4),notnull,default:0.7"`
	SuggestionTTLDays     int64           `json:"suggestionTtlDays"     bun:"suggestion_ttl_days,type:INTEGER,notnull,default:30"`
	RequireExactMatch     bool            `json:"requireExactMatch"     bun:"require_exact_match,type:BOOLEAN,notnull,default:false"`
	WeightRecentShipments bool            `json:"weightRecentShipments" bun:"weight_recent_shipments,type:BOOLEAN,notnull,default:true"`

	Version   int64 `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt int64 `json:"createdAt" bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64 `json:"updatedAt" bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *organization.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
}

func (pc *PatternConfig) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, pc,
		// MinFrequency must be at least 1
		validation.Field(&pc.MinFrequency,
			validation.Required.Error("Minimum frequency is required"),
			validation.Min(int64(1)).Error("Minimum frequency must be at least 1"),
			validation.Max(int64(100)).Error("Minimum frequency must be at most 100"),
		),

		// AnalysisWindowDays must be at least 1
		validation.Field(&pc.AnalysisWindowDays,
			validation.Required.Error("Analysis window days is required"),
			validation.Min(int64(1)).Error("Analysis window days must be at least 1"),
			validation.Max(int64(365)).Error("Analysis window days must be at most 365"),
		),

		// MinConfidenceScore must be between 0 and 1
		validation.Field(&pc.MinConfidenceScore,
			validation.Required.Error("Minimum confidence score is required"),
			validation.By(func(value any) error {
				score, ok := value.(decimal.Decimal)
				if !ok {
					return eris.New("confidence score must be a decimal")
				}
				if score.LessThan(decimal.Zero) || score.GreaterThan(decimal.NewFromFloat(1.0)) {
					return eris.New("confidence score must be between 0 and 1")
				}
				return nil
			}),
		),

		// SuggestionTTLDays must be at least 1
		validation.Field(&pc.SuggestionTTLDays,
			validation.Required.Error("Suggestion TTL days is required"),
			validation.Min(int64(1)).Error("Suggestion TTL days must be at least 1"),
			validation.Max(int64(365)).Error("Suggestion TTL days must be at most 365"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the ID of the pattern config
func (pc *PatternConfig) GetID() string {
	return pc.ID.String()
}

// GetTableName returns the table name
func (pc *PatternConfig) GetTableName() string {
	return "pattern_configs"
}

// ToPatternDetectionConfig converts the PatternConfig to a PatternDetectionConfig
func (pc *PatternConfig) ToPatternDetectionConfig() *PatternDetectionConfig {
	return &PatternDetectionConfig{
		MinFrequency:          pc.MinFrequency,
		AnalysisWindowDays:    pc.AnalysisWindowDays,
		MinConfidenceScore:    pc.MinConfidenceScore,
		SuggestionTTLDays:     pc.SuggestionTTLDays,
		RequireExactMatch:     pc.RequireExactMatch,
		WeightRecentShipments: pc.WeightRecentShipments,
	}
}

// FromPatternDetectionConfig sets the PatternConfig fields from a PatternDetectionConfig
func (pc *PatternConfig) FromPatternDetectionConfig(config *PatternDetectionConfig) {
	pc.MinFrequency = config.MinFrequency
	pc.AnalysisWindowDays = config.AnalysisWindowDays
	pc.MinConfidenceScore = config.MinConfidenceScore
	pc.SuggestionTTLDays = config.SuggestionTTLDays
	pc.RequireExactMatch = config.RequireExactMatch
	pc.WeightRecentShipments = config.WeightRecentShipments
}

// ToJSON converts the PatternConfig to JSON for legacy compatibility
func (pc *PatternConfig) ToJSON() (string, error) {
	config := pc.ToPatternDetectionConfig()
	data, err := sonic.Marshal(config)
	if err != nil {
		return "", eris.Wrap(err, "failed to marshal pattern config to JSON")
	}
	return string(data), nil
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface
func (pc *PatternConfig) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if pc.ID.IsNil() {
			pc.ID = pulid.MustNew("pco_")
		}

		pc.CreatedAt = now
	case *bun.UpdateQuery:
		pc.UpdatedAt = now
	}

	return nil
}
