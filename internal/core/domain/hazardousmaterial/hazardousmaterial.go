package hazardousmaterial

import (
	"context"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain"
	"github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/organization"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/timeutils"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*HazardousMaterial)(nil)
	_ domain.Validatable        = (*HazardousMaterial)(nil)
)

type HazardousMaterial struct {
	bun.BaseModel `bun:"table:hazardous_materials,alias:hm" json:"-"`

	// Primary identifiers
	ID             pulid.ID `bun:",pk,type:VARCHAR(100)" json:"id"`
	BusinessUnitID pulid.ID `bun:"business_unit_id,notnull,type:VARCHAR(100)" json:"businessUnitId"`
	OrganizationID pulid.ID `bun:"organization_id,notnull,type:VARCHAR(100)" json:"organizationId"`

	// Core Fields
	Status                      domain.Status  `bun:"status,type:status,default:'Active'" json:"status"`
	Code                        string         `bun:"code,notnull,type:VARCHAR(100)" json:"code"`
	Name                        string         `bun:"name,notnull,type:VARCHAR(100)" json:"name"`
	Description                 string         `bun:"description,type:TEXT,notnull" json:"description"`
	Class                       HazardousClass `bun:"class,type:hazardous_class_enum,notnull" json:"class"`
	UNNumber                    string         `bun:"un_number,type:VARCHAR(100)" json:"unNumber"`
	ERGNumber                   string         `bun:"erg_number,type:VARCHAR(100)" json:"ergNumber"`
	PackingGroup                PackingGroup   `bun:"packing_group,type:packing_group_enum,notnull" json:"packingGroup"`
	ProperShippingName          string         `bun:"proper_shipping_name,type:TEXT" json:"properShippingName"`
	HandlingInstructions        string         `bun:"handling_instructions,type:TEXT" json:"handlingInstructions"`
	EmergencyContact            string         `bun:"emergency_contact,type:TEXT" json:"emergencyContact"`
	EmergencyContactPhoneNumber string         `bun:"emergency_contact_phone_number,type:TEXT" json:"emergencyContactPhoneNumber"`
	PlacardRequired             bool           `bun:"placard_required,type:BOOLEAN,default:false" json:"placardRequired"`
	IsReportableQuantity        bool           `bun:"is_reportable_quantity,type:BOOLEAN,default:false" json:"isReportableQuantity"`

	// Metadata
	Version   int64 `bun:"version,type:BIGINT" json:"version"`
	CreatedAt int64 `bun:"created_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"createdAt"`
	UpdatedAt int64 `bun:"updated_at,type:BIGINT,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint" json:"updatedAt"`

	// Relationships
	BusinessUnit *businessunit.BusinessUnit `bun:"rel:belongs-to,join:business_unit_id=id" json:"-"`
	Organization *organization.Organization `bun:"rel:belongs-to,join:organization_id=id" json:"-"`
}

func (hm *HazardousMaterial) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, hm,

		// Code is required and must be between 1 and 100 characters
		validation.Field(&hm.Code,
			validation.Required.Error("Code is required"),
			validation.Length(1, 100).Error("Code must be between 1 and 100 characters"),
		),

		// Name is required and must be between 1 and 100 characters
		validation.Field(&hm.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters"),
		),

		// Packing Group must be a valid packing group
		validation.Field(&hm.PackingGroup,
			validation.Required.Error("Packing Group is required"),
			validation.In(PackingGroupI, PackingGroupII, PackingGroupIII).Error("Packing Group must be a valid packing group"),
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
			errors.FromValidationErrors(validationErrs, multiErr, "")
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
