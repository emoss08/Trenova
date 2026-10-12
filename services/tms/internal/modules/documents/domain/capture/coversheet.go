package capture

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
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
	bun.BaseModel `bun:"table:capture_cover_sheets,alias:ccs" json:"-"`

	ID             pulid.ID  `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID  `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID  `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	TokenHash      string    `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull"`
	TargetType     string    `json:"targetType"     bun:"target_type,type:VARCHAR(50),nullzero"`
	TargetID       *pulid.ID `json:"targetId"       bun:"target_id,type:VARCHAR(100),nullzero"`
	DocumentTypeID *pulid.ID `json:"documentTypeId" bun:"document_type_id,type:VARCHAR(100),nullzero"`
	IssuedByID     pulid.ID  `json:"issuedById"     bun:"issued_by_id,type:VARCHAR(100),notnull"`
	ExpiresAt      int64     `json:"expiresAt"      bun:"expires_at,type:BIGINT,notnull"`
	LastUsedAt     *int64    `json:"lastUsedAt"     bun:"last_used_at,type:BIGINT,nullzero"`
	UseCount       int       `json:"useCount"       bun:"use_count,type:INTEGER,notnull,default:0"`
	CreatedAt      int64     `json:"createdAt"      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

func (c *CaptureCoverSheet) BeforeAppendModel(_ context.Context, query bun.Query) error {
	if _, ok := query.(*bun.InsertQuery); ok {
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("ccs_")
		}
		c.CreatedAt = timeutils.NowUnix()
	}

	return nil
}

func (c *CaptureCoverSheet) GetID() pulid.ID      { return c.ID }
func (c *CaptureCoverSheet) GetCreatedAt() int64  { return c.CreatedAt }
func (c *CaptureCoverSheet) GetTableName() string { return "capture_cover_sheets" }
