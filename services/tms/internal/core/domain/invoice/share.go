package invoice

import (
	"context"
	"unicode/utf8"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	MaxShareRecipients = 25
	MaxShareNoteLength = 1000
)

type ShareTab string

const (
	ShareTabOverview  = ShareTab("overview")
	ShareTabDelivery  = ShareTab("delivery")
	ShareTabCharges   = ShareTab("charges")
	ShareTabDocuments = ShareTab("documents")
	ShareTabActivity  = ShareTab("activity")
)

func (t ShareTab) IsValid() bool {
	switch t {
	case ShareTabOverview, ShareTabDelivery, ShareTabCharges, ShareTabDocuments, ShareTabActivity:
		return true
	default:
		return false
	}
}

var _ bun.BeforeAppendModelHook = (*InvoiceShare)(nil)

type InvoiceShare struct {
	bun.BaseModel `bun:"table:invoice_shares,alias:invsh" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	InvoiceID      pulid.ID `json:"invoiceId"      bun:"invoice_id,type:VARCHAR(100),notnull"`
	SharedWithID   pulid.ID `json:"sharedWithId"   bun:"shared_with_id,type:VARCHAR(100),notnull"`
	SharedByID     pulid.ID `json:"sharedById"     bun:"shared_by_id,type:VARCHAR(100),notnull"`
	Note           string   `json:"note"           bun:"note,type:TEXT,nullzero"`
	Tab            ShareTab `json:"tab"            bun:"tab,type:VARCHAR(20),notnull"`
	ShareCount     int      `json:"shareCount"     bun:"share_count,type:INTEGER,notnull"`
	FirstSharedAt  int64    `json:"firstSharedAt"  bun:"first_shared_at,type:BIGINT,notnull"`
	LastSharedAt   int64    `json:"lastSharedAt"   bun:"last_shared_at,type:BIGINT,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,type:BIGINT,notnull"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,type:BIGINT,notnull"`

	SharedWith *tenant.User `json:"sharedWith,omitempty" bun:"rel:belongs-to,join:shared_with_id=id"`
	SharedBy   *tenant.User `json:"sharedBy,omitempty"   bun:"rel:belongs-to,join:shared_by_id=id"`
}

func (s *InvoiceShare) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(s,
		validation.Field(&s.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&s.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&s.InvoiceID,
			validation.Required.Error("Invoice is required"),
		),
		validation.Field(&s.SharedWithID,
			validation.Required.Error("Recipient is required"),
		),
		validation.Field(&s.SharedByID,
			validation.Required.Error("Sharer is required"),
		),
	))

	if !s.Tab.IsValid() {
		multiErr.Add("tab", errortypes.ErrInvalid, "Invoice tab is invalid")
	}
	if utf8.RuneCountInString(s.Note) > MaxShareNoteLength {
		multiErr.Add(
			"note",
			errortypes.ErrInvalid,
			"Note must be {0} characters or fewer",
			MaxShareNoteLength,
		)
	}
	if s.SharedWithID.IsNotNil() && s.SharedWithID == s.SharedByID {
		multiErr.Add("userIds", errortypes.ErrInvalid, "You cannot share an invoice with yourself")
	}
}

func (s *InvoiceShare) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("invsh_")
		}
		if s.CreatedAt == 0 {
			s.CreatedAt = now
		}
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
