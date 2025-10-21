package hazardousmaterial

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/core/domain"
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
	_ bun.BeforeAppendModelHook      = (*HazardousMaterial)(nil)
	_ domain.Validatable             = (*HazardousMaterial)(nil)
	_ domaintypes.PostgresSearchable = (*HazardousMaterial)(nil)
	_ framework.TenantedEntity       = (*HazardousMaterial)(nil)
)

type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`

	ID                          pulid.ID       `json:"id"                          bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID              pulid.ID       `json:"businessUnitId"              bun:"business_unit_id,notnull,type:VARCHAR(100),pk"`
	OrganizationID              pulid.ID       `json:"organizationId"              bun:"organization_id,notnull,type:VARCHAR(100),pk"`
	Status                      domain.Status  `json:"status"                      bun:"status,type:status,default:'Active'"`
	Code                        string         `json:"code"                        bun:"code,notnull,type:VARCHAR(10)"`
	Name                        string         `json:"name"                        bun:"name,notnull,type:VARCHAR(100)"`
	Description                 string         `json:"description"                 bun:"description,type:TEXT,notnull"`
	Class                       HazardousClass `json:"class"                       bun:"class,type:hazardous_class_enum,notnull"`
	UNNumber                    string         `json:"unNumber"                    bun:"un_number,type:VARCHAR(4)"`
	CASNumber                   string         `json:"casNumber"                   bun:"cas_number,type:VARCHAR(10)"`
	PackingGroup                PackingGroup   `json:"packingGroup"                bun:"packing_group,type:packing_group_enum,notnull"`
	SpecialProvisions           string         `json:"specialProvisions"           bun:"special_provisions,type:TEXT"`
	ProperShippingName          string         `json:"properShippingName"          bun:"proper_shipping_name,type:TEXT"`
	HandlingInstructions        string         `json:"handlingInstructions"        bun:"handling_instructions,type:TEXT"`
	EmergencyContact            string         `json:"emergencyContact"            bun:"emergency_contact,type:TEXT"`
	EmergencyContactPhoneNumber string         `json:"emergencyContactPhoneNumber" bun:"emergency_contact_phone_number,type:TEXT"`
	SearchVector                string         `json:"-"                           bun:"search_vector,type:TSVECTOR,scanonly"`
	Rank                        string         `json:"-"                           bun:"rank,type:VARCHAR(100),scanonly"`
	PlacardRequired             bool           `json:"placardRequired"             bun:"placard_required,type:BOOLEAN,default:false"`
	IsReportableQuantity        bool           `json:"isReportableQuantity"        bun:"is_reportable_quantity,type:BOOLEAN,default:false"`
	Version                     int64          `json:"version"                     bun:"version,type:BIGINT"`
	CreatedAt                   int64          `json:"createdAt"                   bun:"created_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                   int64          `json:"updatedAt"                   bun:"updated_at,type:BIGINT,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit *tenant.BusinessUnit `json:"-" bun:"rel:belongs-to,join:business_unit_id=id"`
	Organization *tenant.Organization `json:"-" bun:"rel:belongs-to,join:organization_id=id"`
}

func (hm *HazardousMaterial) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(hm,
		// validation.Field(&hm.Code,
		// 	validation.Required.Error("Code is required"),
		// 	validation.Length(4, 10).Error("Code must be between 1 and 100 characters"),
		// ),
		validation.Field(&hm.UNNumber,
			validation.Length(1, 4).Error("UN Number must be between 1 and 4 characters"),
		),
		validation.Field(&hm.CASNumber,
			validation.Length(1, 10).Error("CAS Number must be between 1 and 10 characters"),
		),
		validation.Field(&hm.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),
		validation.Field(
			&hm.PackingGroup,
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
		validation.Field(&hm.Description,
			validation.Required.Error("Description is required"),
		),
		validation.Field(&hm.SpecialProvisions,
			validation.When(
				hm.SpecialProvisions != "",
				validation.By(domain.ValidateStringOrCommaSeparated),
			),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if errors.As(err, &validationErrs) {
			errortypes.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

func (hm *HazardousMaterial) GetID() string {
	return hm.ID.String()
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
		TableAlias:      "hm",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{
				Name:   "description",
				Type:   domaintypes.FieldTypeText,
				Weight: domaintypes.SearchWeightB,
			},
			{Name: "class", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightB},
			{
				Name:   "packing_group",
				Type:   domaintypes.FieldTypeEnum,
				Weight: domaintypes.SearchWeightB,
			},
		},
	}
}

func (hm *HazardousMaterial) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := utils.NowUnix()

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
