package businessunit

import (
	"context"
	"regexp"

	"github.com/emoss08/trenova/internal/core/domain/usstate"
	"github.com/emoss08/trenova/internal/pkg/utils/timeutils"
	"github.com/emoss08/trenova/pkg/types/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var _ bun.BeforeAppendModelHook = (*BusinessUnit)(nil)

type BusinessUnit struct {
	bun.BaseModel `bun:"table:business_units,alias:bu" json:"-"`

	ID                   pulid.ID       `json:"id" bun:",pk,type:VARCHAR(100)"`
	ParentBusinessUnitID *pulid.ID      `json:"parentBusinessUnitId" bun:"parent_business_unit_id,type:VARCHAR(100),nullzero"`
	StateID              pulid.ID       `json:"stateId" bun:"state_id,type:VARCHAR(100)"`
	Name                 string         `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	Code                 string         `json:"code" bun:"code,type:VARCHAR(10),notnull"`
	Description          string         `json:"description" bun:"description,type:TEXT,notnull"`
	PrimaryContact       string         `json:"primaryContact" bun:"primary_contact,type:VARCHAR(100)"`
	PrimaryEmail         string         `json:"primaryEmail" bun:"primary_email,type:VARCHAR(255)"`
	PrimaryPhone         string         `json:"primaryPhone" bun:"primary_phone,type:VARCHAR(20)"`
	AddressLine1         string         `json:"addressLine1" bun:"address_line1,type:VARCHAR(100)"`
	AddressLine2         string         `json:"addressLine2" bun:"address_line2,type:VARCHAR(100)"`
	City                 string         `json:"city" bun:"city,type:VARCHAR(100)"`
	PostalCode           string         `json:"postalCode" bun:"postal_code,type:us_postal_code,notnull"`
	Timezone             string         `json:"timezone" bun:"timezone,type:VARCHAR(50),notnull,default:'America/New_York'"`
	Locale               string         `json:"locale" bun:"locale,type:VARCHAR(10),notnull,default:'en-US'"`
	TaxID                string         `json:"taxId" bun:"tax_id,type:VARCHAR(50)"`
	Metadata             map[string]any `json:"-" bun:"metadata,type:JSONB,default:'{}'::jsonb"`
	Version              int64          `json:"version" bun:"version,type:BIGINT"`
	CreatedAt            int64          `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt            int64          `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	ParentBusinessUnit *BusinessUnit    `json:"parentBusinessUnit" bun:"rel:belongs-to,join:parent_business_unit_id=id"`
	State              *usstate.UsState `json:"state" bun:"rel:belongs-to,join:state_id=id"`
}

func (bu *BusinessUnit) Validate() error {
	return validation.ValidateStruct(bu,
		validation.
			Field(&bu.Name,
				validation.Required.Error("Name is required. Please try again"),
				validation.Length(1, 100).Error("Name must be between 1 and 100 characters. Please try again"),
				validation.Match(regexp.MustCompile(`^[a-zA-Z0-9\s\-&.]+$`)).
					Error("Name can only contain letters, numbers, spaces, hyphens, ampersands, and periods"),
			),
		// Code validation
		validation.Field(&bu.Code,
			validation.Required.Error("Code is required"),
			validation.Length(2, 10).Error("Code must be between 2 and 10 characters"),
			validation.Match(regexp.MustCompile(`^[A-Z0-9]+$`)).
				Error("Code must contain only uppercase letters and numbers"),
		),
		// Description validation
		validation.Field(&bu.Description,
			validation.Length(0, 255).Error("Description cannot exceed 255 characters"),
		),

		// Contact information validation
		validation.Field(&bu.PrimaryContact,
			validation.Length(0, 100).Error("Primary contact name cannot exceed 100 characters"),
			validation.Match(regexp.MustCompile(`^[a-zA-Z\s\-'.]*$`)).
				Error("Primary contact can only contain letters, spaces, hyphens, apostrophes, and periods"),
		),

		validation.Field(&bu.PrimaryEmail,
			validation.When(bu.PrimaryEmail != "", validation.Required.Error("Email must not be empty if provided")),
			is.EmailFormat.Error("Invalid email format"),
		),

		validation.Field(&bu.PrimaryPhone,
			validation.When(bu.PrimaryPhone != "",
				validation.Match(regexp.MustCompile(`^\+?1?\d{10,14}$`)).
					Error("Invalid phone number format. Must be E.164 format"),
			),
		),
	)
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (bu *BusinessUnit) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if bu.ID.IsNil() {
			bu.ID = pulid.MustNew("bu_")
		}

		bu.CreatedAt = now
	case *bun.UpdateQuery:
		bu.UpdatedAt = now
	}

	return nil
}
