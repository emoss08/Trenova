package invoice

import (
	"unicode/utf8"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
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

type InvoiceShare struct {
	keel.Table `keel:"invoice_shares"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:invsh_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	InvoiceID      pulid.ID `json:"invoiceId" keel:"invoice_id"`
	SharedWithID   pulid.ID `json:"sharedWithId" keel:"shared_with_id"`
	SharedByID     pulid.ID `json:"sharedById" keel:"shared_by_id"`
	Note           string   `json:"note" keel:"note,nullzero"`
	Tab            ShareTab `json:"tab" keel:"tab"`
	ShareCount     int      `json:"shareCount" keel:"share_count"`
	FirstSharedAt  int64    `json:"firstSharedAt" keel:"first_shared_at"`
	LastSharedAt   int64    `json:"lastSharedAt" keel:"last_shared_at"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`

	SharedWith *tenant.User `json:"sharedWith,omitempty" keel:"belongs:shared_with_id"`
	SharedBy   *tenant.User `json:"sharedBy,omitempty" keel:"belongs:shared_by_id"`
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
