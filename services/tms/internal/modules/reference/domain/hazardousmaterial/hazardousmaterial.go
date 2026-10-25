package hazardousmaterial

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var (
	_ validationframework.TenantedEntity = (*HazardousMaterial)(nil)
	_ domaintypes.PostgresSearchable     = (*HazardousMaterial)(nil)
)

// Hazmat reference data
type HazardousMaterial struct {
	keel.Table `keel:"hazardous_materials,report=Freight"`

	ID                          pulid.ID           `json:"id"                          keel:"id,pk,autoid:hm_"`
	BusinessUnitID              pulid.ID           `json:"businessUnitId"              keel:"business_unit_id,pk,tenant"`
	OrganizationID              pulid.ID           `json:"organizationId"              keel:"organization_id,pk,tenant"`
	Status                      domaintypes.Status `json:"status"                      keel:"status"`
	Code                        string             `json:"code"                        keel:"code"`
	Name                        string             `json:"name"                        keel:"name"`
	Description                 string             `json:"description"                 keel:"description"`
	Class                       HazardousClass     `json:"class"                       keel:"class"`
	UNNumber                    string             `json:"unNumber"                    keel:"un_number,nullzero"`
	PackingGroup                PackingGroup       `json:"packingGroup"                keel:"packing_group"`
	SubsidiaryHazardClass       string             `json:"subsidiaryHazardClass"       keel:"subsidiary_hazard_class,nullzero"`
	ErgGuideNumber              string             `json:"ergGuideNumber"              keel:"erg_guide_number,nullzero"`
	LabelCodes                  string             `json:"labelCodes"                  keel:"label_codes,nullzero"`
	SpecialProvisions           string             `json:"specialProvisions"           keel:"special_provisions,nullzero"`
	ProperShippingName          string             `json:"properShippingName"          keel:"proper_shipping_name,nullzero"`
	HandlingInstructions        string             `json:"handlingInstructions"        keel:"handling_instructions,nullzero"`
	EmergencyContact            string             `json:"emergencyContact"            keel:"emergency_contact,nullzero"`
	EmergencyContactPhoneNumber string             `json:"emergencyContactPhoneNumber" keel:"emergency_contact_phone_number,nullzero"`
	QuantityThreshold           string             `json:"quantityThreshold"           keel:"quantity_threshold,nullzero"`
	SearchVector                string             `json:"-"`
	Rank                        string             `json:"-"                           keel:"rank,computed"`
	PlacardRequired             bool               `json:"placardRequired"             keel:"placard_required"`
	IsReportableQuantity        bool               `json:"isReportableQuantity"        keel:"is_reportable_quantity"`
	MarinePollutant             bool               `json:"marinePollutant"             keel:"marine_pollutant"`
	InhalationHazard            bool               `json:"inhalationHazard"            keel:"inhalation_hazard"`
	Version                     int64              `json:"version"                     keel:"version,version"`
	CreatedAt                   int64              `json:"createdAt"                   keel:"created_at,autocreate"`
	UpdatedAt                   int64              `json:"updatedAt"                   keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	Organization *tenant.Organization `json:"organization,omitempty"`
}

func (hm *HazardousMaterial) BeforeCreate(_ context.Context) error {
	if hm.Status == "" {
		hm.Status = domaintypes.StatusActive
	}

	return nil
}

func (hm *HazardousMaterial) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(hm,
		validation.Field(&hm.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(&hm.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&hm.PackingGroup,
			validation.Required.Error("Packing Group is required"),
			validation.In(PackingGroupI, PackingGroupII, PackingGroupIII).
				Error("Packing Group must be a valid packing group"),
		),
		validation.Field(&hm.Class,
			validation.Required.Error("Class is required"),
			validation.In(
				HazardousClass1,
				HazardousClass1And1,
				HazardousClass1And2,
				HazardousClass1And3,
				HazardousClass1And4,
				HazardousClass1And5,
				HazardousClass1And6,
				HazardousClass2And1,
				HazardousClass2And2,
				HazardousClass2And3,
				HazardousClass3,
				HazardousClass4And1,
				HazardousClass4And2,
				HazardousClass4And3,
				HazardousClass5And1,
				HazardousClass5And2,
				HazardousClass6And1,
				HazardousClass6And2,
				HazardousClass7,
				HazardousClass8,
				HazardousClass9,
			).Error("Class is invalid"),
		),
		validation.Field(&hm.UNNumber,
			validation.Length(1, 4).Error("UN Number must be between 1 and 4 characters"),
		),
		validation.Field(&hm.SubsidiaryHazardClass,
			validation.Length(0, 20).Error("Subsidiary Hazard Class must be at most 20 characters"),
		),
		validation.Field(&hm.ErgGuideNumber,
			validation.Length(0, 10).Error("ERG Guide Number must be at most 10 characters"),
		),
		validation.Field(&hm.LabelCodes,
			validation.Length(0, 50).Error("Label Codes must be at most 50 characters"),
		),
		validation.Field(&hm.QuantityThreshold,
			validation.When(
				hm.IsReportableQuantity,
				validation.Required.Error(
					"Quantity Threshold is required when Reportable Quantity is enabled",
				),
			),
			validation.Length(0, 20).Error("Quantity Threshold must be at most 20 characters"),
		),
		validation.Field(&hm.SpecialProvisions,
			validation.When(
				hm.SpecialProvisions != "",
				validation.By(domaintypes.ValidateStringOrCommaSeparated),
			),
		),
	))
}

func (hm *HazardousMaterial) GetID() pulid.ID {
	return hm.ID
}

func (hm *HazardousMaterial) GetCreatedAt() int64 {
	return hm.CreatedAt
}

func (hm *HazardousMaterial) GetTableName() string {
	return "hazardous_materials"
}

func (hm *HazardousMaterial) GetOrganizationID() pulid.ID {
	return hm.OrganizationID
}

func (hm *HazardousMaterial) GetBusinessUnitID() pulid.ID {
	return hm.BusinessUnitID
}

func (hm *HazardousMaterial) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "hm",
		SearchableFields: []domaintypes.SearchableField{
			{
				Name:   "name",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightA,
			},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "class",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
			{
				Name:   "packing_group",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (HazardousMaterial) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "idx_hazardous_materials_code",
			Message: "Hazardous material with this code already exists in your organization",
		},
	}
}
