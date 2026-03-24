package hazmatsegregationrule

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*HazmatSegregationRule)(nil)
	_ domaintypes.PostgresSearchable     = (*HazmatSegregationRule)(nil)
	_ validationframework.TenantedEntity = (*HazmatSegregationRule)(nil)
)

type HazmatSegregationRule struct {
	bun.BaseModel `bun:"table:hazmat_segregation_rules,alias:hsr" json:"-"`

	ID               pulid.ID                         `json:"id"               bun:"id,type:VARCHAR(100),pk,notnull"`
	BusinessUnitID   pulid.ID                         `json:"businessUnitId"   bun:"business_unit_id,type:VARCHAR(100),pk,notnull"`
	OrganizationID   pulid.ID                         `json:"organizationId"   bun:"organization_id,type:VARCHAR(100),pk,notnull"`
	Status           domaintypes.Status               `json:"status"           bun:"status,type:status_enum,notnull,default:'Active'"`
	Name             string                           `json:"name"             bun:"name,type:VARCHAR(100),notnull"`
	Description      string                           `json:"description"      bun:"description,type:TEXT"`
	ExceptionNotes   string                           `json:"exceptionNotes"   bun:"exception_notes,type:TEXT"`
	ReferenceCode    string                           `json:"referenceCode"    bun:"reference_code,type:VARCHAR(100)"`
	RegulationSource string                           `json:"regulationSource" bun:"regulation_source,type:VARCHAR(100)"`
	DistanceUnit     string                           `json:"distanceUnit"     bun:"distance_unit,type:VARCHAR(10)"`
	ClassA           hazardousmaterial.HazardousClass `json:"classA"           bun:"class_a,type:hazardous_class_enum,notnull"`
	ClassB           hazardousmaterial.HazardousClass `json:"classB"           bun:"class_b,type:hazardous_class_enum,notnull"`
	SearchVector     string                           `json:"-"                bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank             string                           `json:"-"                bun:"rank,type:VARCHAR(100),scanonly"`
	SegregationType  SegregationType                  `json:"segregationType"  bun:"segregation_type,type:segregation_type_enum,notnull"`
	HasExceptions    bool                             `json:"hasExceptions"    bun:"has_exceptions,type:BOOLEAN,default:false"`
	Version          int64                            `json:"version"          bun:"version,type:BIGINT"`
	CreatedAt        int64                            `json:"createdAt"        bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt        int64                            `json:"updatedAt"        bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	HazmatAID        *pulid.ID                        `json:"hazmatAId"        bun:"hazmat_a_id,type:VARCHAR(100),nullzero"`
	HazmatBID        *pulid.ID                        `json:"hazmatBId"        bun:"hazmat_b_id,type:VARCHAR(100),nullzero"`
	MinimumDistance  *float64                         `json:"minimumDistance"  bun:"minimum_distance,type:FLOAT,nullzero"`

	BusinessUnit    *tenant.BusinessUnit                 `json:"businessUnit,omitempty"    bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization    *tenant.Organization                 `json:"organization,omitempty"    bun:"rel:belongs-to,join:organization_id=id"`
	HazmatAMaterial *hazardousmaterial.HazardousMaterial `json:"hazmatAMaterial,omitempty" bun:"rel:belongs-to,join:hazmat_a_id=id"`
	HazmatBMaterial *hazardousmaterial.HazardousMaterial `json:"hazmatBMaterial,omitempty" bun:"rel:belongs-to,join:hazmat_b_id=id"`
}

func (hsr *HazmatSegregationRule) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(hsr,
		validation.Field(&hsr.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&hsr.ClassA, validation.Required.Error("Class A is required")),
		validation.Field(&hsr.ClassB, validation.Required.Error("Class B is required")),
		validation.Field(
			&hsr.SegregationType,
			validation.Required.Error("Segregation type is required"),
			validation.In(
				SegregationTypeProhibited,
				SegregationTypeSeparated,
				SegregationTypeDistance,
				SegregationTypeBarrier,
			).Error("Segregation type must be valid"),
		),
		validation.Field(&hsr.MinimumDistance,
			validation.When(
				hsr.SegregationType == SegregationTypeDistance,
				validation.Required.Error(
					"Minimum distance is required when segregation type is Distance",
				),
				validation.Min(0.1).Error("Minimum distance must be greater than 0"),
			),
		),
		validation.Field(&hsr.DistanceUnit,
			validation.When(
				hsr.MinimumDistance != nil,
				validation.Required.Error(
					"Distance unit is required when minimum distance is specified",
				),
				validation.In("FT", "M", "IN", "CM").
					Error("Distance unit must be valid (FT, M, IN, CM)"),
			),
		),
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

	if hsr.ClassA != "" && hsr.ClassB != "" && hsr.ClassA == hsr.ClassB {
		if hsr.HazmatAID == nil || hsr.HazmatBID == nil ||
			(hsr.HazmatAID != nil && hsr.HazmatBID != nil && *hsr.HazmatAID == *hsr.HazmatBID) {
			multiErr.Add(
				"classA",
				errortypes.ErrInvalid,
				"When ClassA and ClassB are the same, specific different hazardous materials must be specified",
			)
		}
	}

	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (hsr *HazmatSegregationRule) GetID() pulid.ID {
	return hsr.ID
}

func (hsr *HazmatSegregationRule) GetTableName() string {
	return "hazmat_segregation_rules"
}

func (hsr *HazmatSegregationRule) GetOrganizationID() pulid.ID {
	return hsr.OrganizationID
}

func (hsr *HazmatSegregationRule) GetBusinessUnitID() pulid.ID {
	return hsr.BusinessUnitID
}

func (hsr *HazmatSegregationRule) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "hsr",
		UseSearchVector: true,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "reference_code",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
			{
				Name:   "regulation_source",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightC,
			},
			{
				Name:   "segregation_type",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "class_a", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{Name: "class_b", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
		},
	}
}

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
