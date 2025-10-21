package hazmatsegregationrule

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/hazardousmaterial"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pulid"
	"github.com/emoss08/trenova/pkg/utils"
	"github.com/emoss08/trenova/pkg/validator/framework"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook      = (*HazmatSegregationRule)(nil)
	_ domaintypes.PostgresSearchable = (*HazmatSegregationRule)(nil)
	_ domain.Validatable             = (*HazmatSegregationRule)(nil)
	_ framework.TenantedEntity       = (*HazmatSegregationRule)(nil)
)

type HazmatSegregationRule struct {
	bun.BaseModel    `bun:"table:hazmat_segregation_rules,alias:hsr" json:"-"`
	ID               pulid.ID                             `bun:"id,type:VARCHAR(100),pk,notnull"                                          json:"id"`
	BusinessUnitID   pulid.ID                             `bun:"business_unit_id,type:VARCHAR(100),pk,notnull"                            json:"businessUnitId"`
	OrganizationID   pulid.ID                             `bun:"organization_id,type:VARCHAR(100),pk,notnull"                             json:"organizationId"`
	Status           domain.Status                        `bun:"status,type:status,default:'Active'"                                      json:"status"`
	Name             string                               `bun:"name,type:VARCHAR(100),notnull"                                           json:"name"`
	Description      string                               `bun:"description,type:TEXT"                                                    json:"description"`
	ExceptionNotes   string                               `bun:"exception_notes,type:TEXT"                                                json:"exceptionNotes"`
	ReferenceCode    string                               `bun:"reference_code,type:VARCHAR(100)"                                         json:"referenceCode"`
	RegulationSource string                               `bun:"regulation_source,type:VARCHAR(100)"                                      json:"regulationSource"`
	DistanceUnit     string                               `bun:"distance_unit,type:VARCHAR(10)"                                           json:"distanceUnit"`
	ClassA           hazardousmaterial.HazardousClass     `bun:"class_a,type:hazardous_class_enum,notnull"                                json:"classA"`
	ClassB           hazardousmaterial.HazardousClass     `bun:"class_b,type:hazardous_class_enum,notnull"                                json:"classB"`
	SearchVector     string                               `bun:"search_vector,type:TSVECTOR,scanonly"                                     json:"-"`
	Rank             string                               `bun:"rank,type:VARCHAR(100),scanonly"                                          json:"-"`
	SegregationType  SegregationType                      `bun:"segregation_type,type:segregation_type_enum,notnull"                      json:"segregationType"`
	HasExceptions    bool                                 `bun:"has_exceptions,type:BOOLEAN,default:false"                                json:"hasExceptions"`
	Version          int64                                `bun:"version,type:BIGINT"                                                      json:"version"`
	CreatedAt        int64                                `bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt        int64                                `bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`
	HazmatAID        *pulid.ID                            `bun:"hazmat_a_id,type:VARCHAR(100),nullzero"                                   json:"hazmatAId"`
	HazmatBID        *pulid.ID                            `bun:"hazmat_b_id,type:VARCHAR(100),nullzero"                                   json:"hazmatBId"`
	MinimumDistance  *float64                             `bun:"minimum_distance,type:FLOAT,nullzero"                                     json:"minimumDistance"`
	BusinessUnit     *tenant.BusinessUnit                 `bun:"rel:belongs-to,join:business_unit_id=id"                                  json:"businessUnit,omitempty"`
	Organization     *tenant.Organization                 `bun:"rel:belongs-to,join:organization_id=id"                                   json:"organization,omitempty"`
	HazmatAMaterial  *hazardousmaterial.HazardousMaterial `bun:"rel:belongs-to,join:hazmat_a_id=id"                                       json:"hazmatAMaterial,omitempty"`
	HazmatBMaterial  *hazardousmaterial.HazardousMaterial `bun:"rel:belongs-to,join:hazmat_b_id=id"                                       json:"hazmatBMaterial,omitempty"`
}

func (hsr *HazmatSegregationRule) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(hsr,
		validation.Field(&hsr.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&hsr.ClassA,
			validation.Required.Error("Class A is required"),
		),
		validation.Field(&hsr.ClassB,
			validation.Required.Error("Class B is required"),
		),
		validation.Field(&hsr.SegregationType,
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
				validation.In("ft", "m", "in", "cm").
					Error("Distance unit must be valid (ft, m, in, cm)"),
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

func (hsr *HazmatSegregationRule) GetID() string {
	return hsr.ID.String()
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
	now := utils.NowUnix()

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
