package hazardousmaterial

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*HazardousMaterial)(nil)
	_ validationframework.TenantedEntity = (*HazardousMaterial)(nil)
	_ domaintypes.PostgresSearchable     = (*HazardousMaterial)(nil)
)

// Hazmat reference data
type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`
	keel.Table    `                                                  keel:"hazardous_materials,report=Freight"`

	pagination.CursorValueSet `json:"-" bun:",embed"`

	ID                          pulid.ID           `json:"id"                          bun:"id,type:VARCHAR(100),pk,notnull"                                          keel:"id,pk,autoid:hm_"`
	BusinessUnitID              pulid.ID           `json:"businessUnitId"              bun:"business_unit_id,type:VARCHAR(100),pk,notnull"                            keel:"business_unit_id,pk,tenant"`
	OrganizationID              pulid.ID           `json:"organizationId"              bun:"organization_id,type:VARCHAR(100),pk,notnull"                             keel:"organization_id,pk,tenant"`
	Status                      domaintypes.Status `json:"status"                      bun:"status,type:status_enum,notnull,default:'Active'"                         keel:"status"`
	Code                        string             `json:"code"                        bun:"code,type:VARCHAR(10),notnull"                                            keel:"code"`
	Name                        string             `json:"name"                        bun:"name,type:VARCHAR(100),notnull"                                           keel:"name"`
	Description                 string             `json:"description"                 bun:"description,type:TEXT,notnull"                                            keel:"description"`
	Class                       HazardousClass     `json:"class"                       bun:"class,type:hazardous_class_enum,notnull"                                  keel:"class"`
	UNNumber                    string             `json:"unNumber"                    bun:"un_number,type:VARCHAR(4)"                                                keel:"un_number,nullzero"`
	PackingGroup                PackingGroup       `json:"packingGroup"                bun:"packing_group,type:packing_group_enum,notnull"                            keel:"packing_group"`
	SubsidiaryHazardClass       string             `json:"subsidiaryHazardClass"       bun:"subsidiary_hazard_class,type:VARCHAR(20)"                                 keel:"subsidiary_hazard_class,nullzero"`
	ErgGuideNumber              string             `json:"ergGuideNumber"              bun:"erg_guide_number,type:VARCHAR(10)"                                        keel:"erg_guide_number,nullzero"`
	LabelCodes                  string             `json:"labelCodes"                  bun:"label_codes,type:VARCHAR(50)"                                             keel:"label_codes,nullzero"`
	SpecialProvisions           string             `json:"specialProvisions"           bun:"special_provisions,type:TEXT"                                             keel:"special_provisions,nullzero"`
	ProperShippingName          string             `json:"properShippingName"          bun:"proper_shipping_name,type:TEXT"                                           keel:"proper_shipping_name,nullzero"`
	HandlingInstructions        string             `json:"handlingInstructions"        bun:"handling_instructions,type:TEXT"                                          keel:"handling_instructions,nullzero"`
	EmergencyContact            string             `json:"emergencyContact"            bun:"emergency_contact,type:TEXT"                                              keel:"emergency_contact,nullzero"`
	EmergencyContactPhoneNumber string             `json:"emergencyContactPhoneNumber" bun:"emergency_contact_phone_number,type:TEXT"                                 keel:"emergency_contact_phone_number,nullzero"`
	QuantityThreshold           string             `json:"quantityThreshold"           bun:"quantity_threshold,type:VARCHAR(20)"                                      keel:"quantity_threshold,nullzero"`
	SearchVector                string             `json:"-"                           bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                        string             `json:"-"                           bun:"rank,type:VARCHAR(100),scanonly"                                          keel:"rank,computed"`
	PlacardRequired             bool               `json:"placardRequired"             bun:"placard_required,type:BOOLEAN"                                            keel:"placard_required"`
	IsReportableQuantity        bool               `json:"isReportableQuantity"        bun:"is_reportable_quantity,type:BOOLEAN"                                      keel:"is_reportable_quantity"`
	MarinePollutant             bool               `json:"marinePollutant"             bun:"marine_pollutant,type:BOOLEAN"                                            keel:"marine_pollutant"`
	InhalationHazard            bool               `json:"inhalationHazard"            bun:"inhalation_hazard,type:BOOLEAN"                                           keel:"inhalation_hazard"`
	Version                     int64              `json:"version"                     bun:"version,type:BIGINT"                                                      keel:"version,version"`
	CreatedAt                   int64              `json:"createdAt"                   bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt                   int64              `json:"updatedAt"                   bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`

	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
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

func (hm *HazardousMaterial) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if hm.ID.IsNil() {
			hm.ID = pulid.MustNew("hm_")
		}

		hm.CreatedAt = now
	case *bun.UpdateQuery:
		hm.UpdatedAt = now
	}

	return nil
}
