package hazardousmaterial

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain"
	"github.com/emoss08/trenova/internal/core/domain/businessunit"
	"github.com/emoss08/trenova/internal/core/domain/organization"
	"github.com/emoss08/trenova/internal/pkg/errors"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*HazardousMaterial)(nil)
	_ domain.Validatable        = (*HazardousMaterial)(nil)
)

type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`

	ID                          pulid.ID       `json:"id "                         bun:"id,pk,type:VARCHAR(100)"`
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
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id"  json:"-"`
}

func (hm *HazardousMaterial) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, hm,

		// Code is required and must be between 1 and 100 characters
		validation.Field(&hm.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),

		// UN Number must be between 1 and 4 characters
		validation.Field(&hm.UNNumber,
			validation.Length(1, 4).Error("UN Number must be between 1 and 4 characters"),
		),

		// CAS Number must be between 1 and 10 characters
		validation.Field(&hm.CASNumber,
			validation.Length(1, 10).Error("CAS Number must be between 1 and 10 characters"),
		),

		// Name is required and must be between 1 and 100 characters
		validation.Field(&hm.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// Packing Group must be a valid packing group
		validation.Field(
			&hm.PackingGroup,
			validation.Required.Error("Packing Group is required"),
			validation.In(PackingGroupI, PackingGroupII, PackingGroupIII).
				Error("Packing Group must be a valid packing group"),
		),

		// Class is required
		validation.Field(&hm.Class,
			validation.Required.Error("Class is required"),
			validation.In(
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

		// Description is required
		validation.Field(&hm.Description,
			validation.Required.Error("Description is required"),
		),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromOzzoErrors(validationErrs, multiErr)
		}
	}
}

// Pagination Configuration
func (hm *HazardousMaterial) GetID() string {
	return hm.ID.String()
}

func (hm *HazardousMaterial) GetTableName() string {
	return "hazardous_materials"
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
