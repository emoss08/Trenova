package capture

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

const (
	// CoverSheetPrefix begins every cover sheet's QR payload, so a QR code that
	// happens to be on a page (a carrier's tracking link, a BOL's own code) is
	// never mistaken for one.
	CoverSheetPrefix = "TRNV-CS1:"
	// CoverSheetLifetimeSeconds is how long a printed sheet keeps routing. A
	// sheet found in a drawer a year later should split the stack, not file
	// into a shipment that closed eleven months ago.
	CoverSheetLifetimeSeconds = 180 * 24 * 60 * 60
)

// CaptureCoverSheet is a printed separator that says where the pages behind it go.
//
// The QR code carries a random token, not the record: a sheet photocopied
// into another tenant's stack finds no row there, and reading one tells
// nobody which shipment it belongs to. The token is stored hashed, like every
// other bearer secret here.
type CaptureCoverSheet struct {
	keel.Table `keel:"capture_cover_sheets"`

	ID             pulid.ID  `json:"id" keel:"id,pk,autoid:ccs_"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID  `json:"organizationId" keel:"organization_id,pk,tenant"`
	TokenHash      string    `json:"-" keel:"token_hash"`
	TargetType     string    `json:"targetType" keel:"target_type,nullzero"`
	TargetID       *pulid.ID `json:"targetId" keel:"target_id"`
	DocumentTypeID *pulid.ID `json:"documentTypeId" keel:"document_type_id"`
	IssuedByID     pulid.ID  `json:"issuedById" keel:"issued_by_id"`
	ExpiresAt      int64     `json:"expiresAt" keel:"expires_at"`
	LastUsedAt     *int64    `json:"lastUsedAt" keel:"last_used_at"`
	UseCount       int       `json:"useCount" keel:"use_count"`
	CreatedAt      int64     `json:"createdAt" keel:"created_at,autocreate"`
}

func (c *CaptureCoverSheet) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.IssuedByID, validation.Required.Error("Issuer is required")),
		validation.Field(&c.ExpiresAt, validation.Required.Error("Expiry is required")),
	))

	c.Target().Validate(multiErr, targetFields, false)
}

// Target is where the sheet routes. A sheet with no record only divides the
// stack.
func (c *CaptureCoverSheet) Target() Target {
	return Target{
		ResourceType:   c.TargetType,
		ResourceID:     c.TargetID,
		DocumentTypeID: c.DocumentTypeID,
	}
}

// IsExpired reports whether the sheet no longer routes.
func (c *CaptureCoverSheet) IsExpired(now int64) bool {
	return now >= c.ExpiresAt
}

// CoverSheetToken extracts the token from a decoded QR payload, reporting
// whether the payload was a cover sheet at all.
func CoverSheetToken(payload string) (string, bool) {
	token, ok := strings.CutPrefix(strings.TrimSpace(payload), CoverSheetPrefix)
	if !ok || token == "" {
		return "", false
	}

	return token, true
}

// CoverSheetPayload is what the QR code on a sheet encodes.
func CoverSheetPayload(token string) string {
	return CoverSheetPrefix + token
}

func (c *CaptureCoverSheet) GetID() pulid.ID      { return c.ID }
func (c *CaptureCoverSheet) GetCreatedAt() int64  { return c.CreatedAt }
func (c *CaptureCoverSheet) GetTableName() string { return "capture_cover_sheets" }
