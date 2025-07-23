// # Copyright 2023-2025 Eric Moss
// # Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
// # Full license: https://github.com/emoss08/trenova/blob/main/LICENSE.md

package hazmatsegregationrule

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/core/ports/infra"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*HazmatSegregationRule)(nil)
	_ domain.Validatable        = (*HazmatSegregationRule)(nil)
	_ infra.PostgresSearchable  = (*HazmatSegregationRule)(nil)
)

type HazmatSegregationRule struct {
	bun.BaseModel `bun:"table:hazmat_segregation_rules,alias:hsr" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:"id,type:VARCHAR(100),pk,notnull"               json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,type:VARCHAR(100),pk,notnull" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,type:VARCHAR(100),pk,notnull"  json:"organizationId"`

	// Relationship identifiers (Non-Primary-Keys)
	HazmatAID *pulid.ID `bun:"hazmat_a_id,type:VARCHAR(100),nullzero" json:"hazmatAId"`
	HazmatBID *pulid.ID `bun:"hazmat_b_id,type:VARCHAR(100),nullzero" json:"hazmatBId"`

	// Core Fields
	Status           domain.Status                    `bun:"status,type:status,default:'Active'"                 json:"status"`
	Name             string                           `bun:"name,type:VARCHAR(100),notnull"                      json:"name"`
	Description      string                           `bun:"description,type:TEXT"                               json:"description"`
	ClassA           hazardousmaterial.HazardousClass `bun:"class_a,type:hazardous_class_enum,notnull"           json:"classA"`
	ClassB           hazardousmaterial.HazardousClass `bun:"class_b,type:hazardous_class_enum,notnull"           json:"classB"`
	SegregationType  SegregationType                  `bun:"segregation_type,type:segregation_type_enum,notnull" json:"segregationType"`
	MinimumDistance  *float64                         `bun:"minimum_distance,type:FLOAT,nullzero"                json:"minimumDistance"`
	DistanceUnit     string                           `bun:"distance_unit,type:VARCHAR(10)"                      json:"distanceUnit"`
	HasExceptions    bool                             `bun:"has_exceptions,type:BOOLEAN,default:false"           json:"hasExceptions"`
	ExceptionNotes   string                           `bun:"exception_notes,type:TEXT"                           json:"exceptionNotes"`
	ReferenceCode    string                           `bun:"reference_code,type:VARCHAR(100)"                    json:"referenceCode"`
	RegulationSource string                           `bun:"regulation_source,type:VARCHAR(100)"                 json:"regulationSource"`

	// Metadata
	Version      int64  `json:"version"   bun:"version,type:BIGINT"`
	CreatedAt    int64  `json:"createdAt" bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt    int64  `json:"updatedAt" bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	SearchVector string `json:"-"         bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank         string `json:"-"         bun:"rank,type:VARCHAR(100),scanonly"`

	// Relationships
	BusinessUnit    *businessunit.BusinessUnit           `json:"businessUnit,omitempty"    bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *organization.Organization           `json:"organization,omitempty"    bun:"rel:belongs-to,join:organization_id=id"`
	HazmatAMaterial *hazardousmaterial.HazardousMaterial `json:"hazmatAMaterial,omitempty" bun:"rel:belongs-to,join:hazmat_a_id=id"`
	HazmatBMaterial *hazardousmaterial.HazardousMaterial `json:"hazmatBMaterial,omitempty" bun:"rel:belongs-to,join:hazmat_b_id=id"`
}

func (hsr *HazmatSegregationRule) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, hsr,
		// Name validation
		validation.Field(&hsr.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// ClassA and ClassB validation
		validation.Field(&hsr.ClassA,
			validation.Required.Error("Class A is required"),
		),
		validation.Field(&hsr.ClassB,
			validation.Required.Error("Class B is required"),
		),

		// SegregationType validation
		validation.Field(&hsr.SegregationType,
			validation.Required.Error("Segregation type is required"),
			validation.In(
				SegregationTypeProhibited,
				SegregationTypeSeparated,
				SegregationTypeDistance,
				SegregationTypeBarrier,
			).Error("Segregation type must be valid"),
		),

		// Distance validation when type is Distance
		validation.Field(&hsr.MinimumDistance,
			validation.When(
				hsr.SegregationType == SegregationTypeDistance,
				validation.Required.Error(
					"Minimum distance is required when segregation type is Distance",
				),
				validation.Min(0.1).Error("Minimum distance must be greater than 0"),
			),
		),

		// Distance unit validation when distance is specified
		validation.Field(&hsr.DistanceUnit,
			validation.When(
				hsr.MinimumDistance != nil,
				validation.Required.Error(
					"Distance unit is required when minimum distance is specified",
				),
				validation.In("ft", "m", "in", "cm").
					Error("Distance unit must be valid (ft, m, in, cm)"),
			),
		),

		// Exception notes validation when hasExceptions is true
		validation.Field(&hsr.ExceptionNotes,
			validation.When(
				hsr.HasExceptions,
				validation.Required.Error(
					"Exception notes are required when exceptions are indicated",
				),
				validation.Length(5, 1000).
					Error("Exception notes must be between 5 and 1000 characters"),
			),
		),
	)

	// If ClassA and ClassB are the same, then HazmatAID and HazmatBID must be different
	if hsr.ClassA != "" && hsr.ClassB != "" && hsr.ClassA == hsr.ClassB {
		if hsr.HazmatAID == nil || hsr.HazmatBID == nil ||
			(hsr.HazmatAID != nil && hsr.HazmatBID != nil && *hsr.HazmatAID == *hsr.HazmatBID) {
			multiErr.Add(
				"classA",
				errors.ErrInvalid,
				"When ClassA and ClassB are the same, specific different hazardous materials must be specified",
			)
		}
	}

	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// GetID returns the ID as a string
func (hsr *HazmatSegregationRule) GetID() string {
	return hsr.ID.String()
}

// GetTableName returns the table name
func (hsr *HazmatSegregationRule) GetTableName() string {
	return "hazmat_segregation_rules"
}

// BeforeAppendModel implements bun.BeforeAppendModelHook
func (hsr *HazmatSegregationRule) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if hsr.ID.IsNil() {
			hsr.ID = pulid.MustNew("hsr_")
		}

		hsr.CreatedAt = now
	case *bun.UpdateQuery:
		hsr.UpdatedAt = now
	}

	return nil
}

func (hsr *HazmatSegregationRule) GetPostgresSearchConfig() infra.PostgresSearchConfig {
	return infra.PostgresSearchConfig{
		TableAlias: "hsr",
		Fields: []infra.PostgresSearchableField{
			{
				Name:   "name",
				Weight: "A",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "description",
				Weight: "B",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "reference_code",
				Weight: "C",
				Type:   infra.PostgresSearchTypeText,
			},
			{
				Name:   "regulation_source",
				Weight: "C",
				Type:   infra.PostgresSearchTypeText,
			},
		},
		MinLength:       2,
		MaxTerms:        6,
		UsePartialMatch: true,
	}
}
