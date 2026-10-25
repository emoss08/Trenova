package hazmatsegregationrule

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/reference/domain/hazardousmaterial"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ domaintypes.PostgresSearchable     = (*HazmatSegregationRule)(nil)
	_ validationframework.TenantedEntity = (*HazmatSegregationRule)(nil)
)

type HazmatSegregationRule struct {
	keel.Table `keel:"hazmat_segregation_rules"`

	ID               pulid.ID                         `json:"id"               keel:"id,pk,autoid:hsr_"`
	BusinessUnitID   pulid.ID                         `json:"businessUnitId"   keel:"business_unit_id,pk,tenant"`
	OrganizationID   pulid.ID                         `json:"organizationId"   keel:"organization_id,pk,tenant"`
	Status           domaintypes.Status               `json:"status"           keel:"status"`
	Name             string                           `json:"name"             keel:"name"`
	Description      string                           `json:"description"      keel:"description,nullzero"`
	ExceptionNotes   string                           `json:"exceptionNotes"   keel:"exception_notes,nullzero"`
	ReferenceCode    string                           `json:"referenceCode"    keel:"reference_code,nullzero"`
	RegulationSource string                           `json:"regulationSource" keel:"regulation_source,nullzero"`
	DistanceUnit     string                           `json:"distanceUnit"     keel:"distance_unit,nullzero"`
	ClassA           hazardousmaterial.HazardousClass `json:"classA"           keel:"class_a"`
	ClassB           hazardousmaterial.HazardousClass `json:"classB"           keel:"class_b"`
	SearchVector     string                           `json:"-"                keel:"search_vector,tsvector,readonly,noselect"`
	Rank             string                           `json:"-"                keel:"rank,computed"`
	SegregationType  SegregationType                  `json:"segregationType"  keel:"segregation_type"`
	HasExceptions    bool                             `json:"hasExceptions"    keel:"has_exceptions"`
	Version          int64                            `json:"version"          keel:"version,version"`
	CreatedAt        int64                            `json:"createdAt"        keel:"created_at,autocreate"`
	UpdatedAt        int64                            `json:"updatedAt"        keel:"updated_at,autoupdate"`
	HazmatAID        *pulid.ID                        `json:"hazmatAId"        keel:"hazmat_a_id"`
	HazmatBID        *pulid.ID                        `json:"hazmatBId"        keel:"hazmat_b_id"`
	MinimumDistance  *float64                         `json:"minimumDistance"  keel:"minimum_distance"`

	BusinessUnit    *tenant.BusinessUnit                 `json:"businessUnit,omitempty"`
	Organization    *tenant.Organization                 `json:"organization,omitempty"`
	HazmatAMaterial *hazardousmaterial.HazardousMaterial `json:"hazmatAMaterial,omitempty" keel:"belongs:hazmat_a_id"`
	HazmatBMaterial *hazardousmaterial.HazardousMaterial `json:"hazmatBMaterial,omitempty" keel:"belongs:hazmat_b_id"`
}

func (hsr *HazmatSegregationRule) BeforeCreate(_ context.Context) error {
	if hsr.Status == "" {
		hsr.Status = domaintypes.StatusActive
	}

	return nil
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

	multiErr.AddOzzoError(err)
}

func (hsr *HazmatSegregationRule) GetID() pulid.ID {
	return hsr.ID
}

func (hsr *HazmatSegregationRule) GetCreatedAt() int64 {
	return hsr.CreatedAt
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

func (HazmatSegregationRule) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_hazmat_segregation_rules_unique",
			Message: "A segregation rule for these classes and materials already exists in your organization",
		},
		{
			Name:    "fk_hazmat_segregation_rules_hazmat_a",
			Message: "Hazardous material A does not exist or belongs to a different organization",
		},
		{
			Name:    "fk_hazmat_segregation_rules_hazmat_b",
			Message: "Hazardous material B does not exist or belongs to a different organization",
		},
	}
}
