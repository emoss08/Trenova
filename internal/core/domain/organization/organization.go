package organization

import (
	"context"
	"crypto/rand"
	"encoding/hex"

	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/rotisserie/eris"
	"github.com/trenova-app/transport/internal/core/domain"
	businessunit "github.com/trenova-app/transport/internal/core/domain/businessunit"
	"github.com/trenova-app/transport/internal/core/domain/documentqualityconfig"
	"github.com/trenova-app/transport/internal/core/domain/usstate"
	"github.com/trenova-app/transport/internal/pkg/errors"
	"github.com/trenova-app/transport/internal/pkg/utils/queryutils"
	"github.com/trenova-app/transport/internal/pkg/utils/timeutils"
	"github.com/trenova-app/transport/pkg/types/pulid"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*Organization)(nil)
	_ domain.Validatable        = (*Organization)(nil)
)

// Metadata is the metadata for an organization.
type Metadata struct {
	// ObjectID is the ID of the organization logo object in the storage bucket.
	ObjectID string `json:"objectId"`
}

type Organization struct {
	bun.BaseModel `bun:"table:organizations,alias:org" json:"-"`

	// Primary identifiers
	ID             pulid.ID `json:"id" bun:",pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	StateID        pulid.ID `json:"stateId" bun:"state_id,type:VARCHAR(100),notnull"`

	// Core fields
	Name           string `json:"name" bun:"name,type:VARCHAR(100),notnull"`
	ScacCode       string `json:"scacCode" bun:"scac_code,type:VARCHAR(4),notnull"`
	DOTNumber      string `json:"dotNumber" bun:"dot_number,type:VARCHAR(8),notnull"`
	LogoURL        string `json:"logoUrl" bun:"logo_url,type:VARCHAR(255)"`
	OrgType        Type   `json:"orgType" bun:"org_type,type:org_type_enum,notnull,default:'Asset'"`
	BucketName     string `json:"-" bun:"bucket_name,type:VARCHAR(63),notnull"`
	AddressLine1   string `json:"addressLine1" bun:"address_line1,type:VARCHAR(150),notnull"`
	AddressLine2   string `json:"addressLine2" bun:"address_line2,type:VARCHAR(150)"`
	City           string `json:"city" bun:"city,type:VARCHAR(100),notnull"`
	PostalCode     string `json:"postalCode" bun:"postal_code,type:VARCHAR(20)"`
	Timezone       string `json:"timezone" bun:"timezone,type:VARCHAR(100),notnull,default:'America/New_York'"`
	TaxID          string `json:"taxId" bun:"tax_id,type:VARCHAR(50)"`
	PrimaryContact string `json:"primaryContact" bun:"primary_contact,type:VARCHAR(100)"`
	PrimaryEmail   string `json:"primaryEmail" bun:"primary_email,type:VARCHAR(255)"`
	PrimaryPhone   string `json:"primaryPhone" bun:"primary_phone,type:VARCHAR(20)"`

	// Metadata and versioning
	Metadata  *Metadata `json:"-" bun:"metadata,type:JSONB"` // Do not expose this to the API
	Version   int64     `json:"version" bun:"version,type:BIGINT"`
	CreatedAt int64     `json:"createdAt" bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt int64     `json:"updatedAt" bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	// Relationships
	BusinessUnit          *businessunit.BusinessUnit                   `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	State                 *usstate.UsState                             `json:"state,omitempty" bun:"rel:belongs-to,join:state_id=id"`
	DocumentQualityConfig *documentqualityconfig.DocumentQualityConfig `json:"documentQualityConfig,omitempty" bun:"rel:has-one,join:id=organization_id"`
}

func (o *Organization) Validate(ctx context.Context, multiErr *errors.MultiError) {
	err := validation.ValidateStructWithContext(ctx, o,
		validation.Field(&o.Name,
			validation.Required.Error("Name is required. Please try again"),
			validation.Length(1, 100).Error("Name must be between 1 and 100 characters")),

		validation.Field(&o.ScacCode,
			validation.Required.Error("SCAC code is required. Please try again"),
			validation.Length(4, 4).Error("SCAC code must be 4 characters")),

		validation.Field(&o.DOTNumber,
			validation.Required.Error("DOT number is required. Please try again"),
			validation.Length(1, 8).Error("DOT number must be between 1 and 8 characters"),
			is.Digit.Error("DOT number must be numeric")),

		validation.Field(&o.OrgType,
			validation.Required.Error("Organization type is required. Please try again")),

		validation.Field(&o.AddressLine1,
			validation.Length(0, 150).Error("Address line 1 must be less than 150 characters. Please try again")),

		validation.Field(&o.City,
			validation.Required.Error("City is required. Please try again")),

		// Combined timezone validation
		validation.Field(&o.Timezone,
			validation.Required.Error("Timezone is required. Please try again"),
			validation.By(domain.ValidateTimezone)),
	)
	if err != nil {
		var validationErrs validation.Errors
		if eris.As(err, &validationErrs) {
			errors.FromValidationErrors(validationErrs, multiErr, "")
		}
	}
}

func (o *Organization) ValidateUniqueness(ctx context.Context, tx bun.IDB, multiErr *errors.MultiError) {
	vb := queryutils.NewUniquenessValidator(o.GetTableName()).
		WithModelName("Organization").
		WithBusinessUnit(o.BusinessUnitID).
		WithFieldAndTemplate(
			"name",
			o.Name,
			"Organization with name ':value' already exists in the business unit. Please try again with a different name.",
			map[string]string{
				"value": o.Name,
			}).
		WithFieldAndTemplate(
			"scac_code",
			o.ScacCode,
			"Organization with SCAC code ':value' already exists in the business unit. Please try again with a different SCAC code.",
			map[string]string{
				"value": o.ScacCode,
			}).
		WithFieldAndTemplate(
			"dot_number",
			o.DOTNumber,
			"Organization with DOT number ':value' already exists in the business unit. Please try again with a different DOT number.",
			map[string]string{
				"value": o.DOTNumber,
			})

	if o.ID.IsNotNil() {
		vb.WithOperation(queryutils.OperationUpdate).WithPrimaryKey("id", o.ID.String())
	} else {
		vb.WithOperation(queryutils.OperationCreate)
	}

	queryutils.CheckFieldUniqueness(ctx, tx, vb.Build(), multiErr)
}

func (o *Organization) GetTableName() string {
	return "organizations"
}

func (o *Organization) DBValidate(ctx context.Context, tx bun.IDB) *errors.MultiError {
	multiErr := errors.NewMultiError()

	// Run the standard validation
	o.Validate(ctx, multiErr)

	// Run the uniqueness validation
	o.ValidateUniqueness(ctx, tx, multiErr)

	if multiErr.HasErrors() {
		return multiErr
	}

	return nil
}

// generateUniqueBucketName generates a unique bucket name for an organization.
func generateUniqueBucketName() (string, error) {
	rb := make([]byte, 16)
	if _, err := rand.Read(rb); err != nil {
		return "", eris.Wrap(err, "failed to generate random bytes")
	}

	// Convert the bytes to a hex string.
	bucketName := "org-" + hex.EncodeToString(rb)

	// Ensure the bucket name is not longer than 63 characters.
	if len(bucketName) > 63 {
		// Truncate the bucket name to 63 characters.
		bucketName = bucketName[:63]
	}

	return bucketName, nil
}

// BeforeAppendModel implements the bun.BeforeAppendModelHook interface.
func (o *Organization) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if o.ID.IsNil() {
			o.ID = pulid.MustNew("org_")
		}

		if o.BucketName == "" {
			bucketName, err := generateUniqueBucketName()
			if err != nil {
				return eris.Wrap(err, "failed to generate unique bucket name")
			}
			o.BucketName = bucketName
		}

		o.CreatedAt = now
	case *bun.UpdateQuery:
		o.UpdatedAt = now
	}

	return nil
}
