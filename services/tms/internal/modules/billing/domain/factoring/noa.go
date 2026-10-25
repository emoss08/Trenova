package factoring

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/parties/domain/customer"
	"github.com/emoss08/trenova/internal/modules/parties/domain/factoringcompany"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	MaxNoaNotesLength         = 2000
	MaxNoaReleaseReasonLength = 500
	NoaExpiryWarningDays      = 30
	NoaDocumentResourceType   = "factoring_noa"
	MaxNoaRecipientLength     = 320
	NoaFutureSlackSeconds     = 300
)

// Notices telling each customer to pay a factor, from draft to signed acknowledgement, expiry and release
type Noa struct {
	keel.Table `keel:"factoring_noas,report=Billing,label=Notice of Assignment,plural=Notices of Assignment"`

	ID                 pulid.ID  `json:"id"                 keel:"id,pk,autoid:facnoa_"`
	OrganizationID     pulid.ID  `json:"organizationId"     keel:"organization_id,pk,tenant"`
	BusinessUnitID     pulid.ID  `json:"businessUnitId"     keel:"business_unit_id,pk,tenant"`
	CustomerID         pulid.ID  `json:"customerId"         keel:"customer_id"`
	FactoringCompanyID pulid.ID  `json:"factoringCompanyId" keel:"factoring_company_id"`
	Status             NoaStatus `json:"status"             keel:"status"`
	SentAt             *int64    `json:"sentAt"             keel:"sent_at"`
	SentTo             string    `json:"sentTo"             keel:"sent_to,nullzero"`
	AcknowledgedAt     *int64    `json:"acknowledgedAt"     keel:"acknowledged_at"`
	SignedDocumentID   pulid.ID  `json:"signedDocumentId"   keel:"signed_document_id,nullzero,label=Signed Copy Document ID"`
	ExpiresAt          *int64    `json:"expiresAt"          keel:"expires_at"`
	ExpiryWarnedAt     *int64    `json:"expiryWarnedAt"     keel:"expiry_warned_at"`
	ReleasedAt         *int64    `json:"releasedAt"         keel:"released_at"`
	ReleaseReason      string    `json:"releaseReason"      keel:"release_reason,nullzero"`
	Notes              string    `json:"notes"              keel:"notes,nullzero"`
	Version            int64     `json:"version"            keel:"version,version"`
	CreatedAt          int64     `json:"createdAt"          keel:"created_at,autocreate"`
	UpdatedAt          int64     `json:"updatedAt"          keel:"updated_at,autoupdate"`

	Customer         *customer.Customer                 `json:"customer,omitempty"         keel:"belongs:customer_id"`
	FactoringCompany *factoringcompany.FactoringCompany `json:"factoringCompany,omitempty" keel:"belongs:factoring_company_id"`
}

func (n *Noa) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(
		n,
		validation.Field(&n.OrganizationID, validation.Required),
		validation.Field(&n.BusinessUnitID, validation.Required),
		validation.Field(&n.CustomerID, validation.Required.Error("Customer is required")),
		validation.Field(
			&n.FactoringCompanyID,
			validation.Required.Error("Factoring company is required"),
		),
		validation.Field(
			&n.Status,
			validation.Required,
			domainvalidation.ValidEnum[NoaStatus]("Invalid notice status"),
		),
		validation.Field(
			&n.SentTo,
			is.EmailFormat.Error("Recipient must be a valid email address"),
		),
		validation.Field(
			&n.ReleaseReason,
			validation.Length(0, MaxNoaReleaseReasonLength).
				Error("Release reason cannot be longer than 500 characters"),
		),
		validation.Field(
			&n.Notes,
			validation.Length(0, MaxNoaNotesLength).
				Error("Notes cannot be longer than 2000 characters"),
		),
	))

	switch n.Status {
	case NoaStatusSent:
		if n.SentAt == nil {
			multiErr.Add(
				"sentAt",
				errortypes.ErrRequired,
				"A sent notice needs the time it was sent",
			)
		}
	case NoaStatusAcknowledged:
		if n.AcknowledgedAt == nil {
			multiErr.Add(
				"acknowledgedAt",
				errortypes.ErrRequired,
				"An acknowledged notice needs the date the customer signed it",
			)
		}
		if n.SignedDocumentID.IsNil() {
			multiErr.Add(
				"signedDocumentId",
				errortypes.ErrRequired,
				"Attach the customer's signed copy of the notice",
			)
		}
		if n.ExpiresAt != nil && n.AcknowledgedAt != nil && *n.ExpiresAt <= *n.AcknowledgedAt {
			multiErr.Add(
				"expiresAt",
				errortypes.ErrInvalid,
				"The notice must expire after it was acknowledged",
			)
		}
	case NoaStatusReleased:
		if n.ReleasedAt == nil {
			multiErr.Add(
				"releasedAt",
				errortypes.ErrRequired,
				"A released notice needs its release time",
			)
		}
	case NoaStatusDraft:
	}
}

func (n *Noa) IsOpen() bool {
	return n != nil && n.Status != NoaStatusReleased
}

func (n *Noa) IsEffective(at int64) bool {
	if n == nil || n.Status != NoaStatusAcknowledged {
		return false
	}
	return n.ExpiresAt == nil || at < *n.ExpiresAt
}

func (n *Noa) ExpiresWithin(at int64, days int) bool {
	if !n.IsEffective(at) || n.ExpiresAt == nil {
		return false
	}
	return *n.ExpiresAt-at <= int64(days)*timeutils.SecondsPerDay
}

func (n *Noa) CanSend() bool {
	return n != nil && (n.Status == NoaStatusDraft || n.Status == NoaStatusSent)
}

func (n *Noa) CanAcknowledge() bool {
	return n != nil && (n.Status == NoaStatusDraft || n.Status == NoaStatusSent)
}

func (n *Noa) MarkSent(at int64, recipients []string) error {
	if !n.CanSend() {
		return errortypes.NewBusinessError(
			"Only a drafted or sent notice can be sent; draft a new notice instead",
		)
	}
	if len(recipients) == 0 {
		return errortypes.NewValidationError(
			"recipient",
			errortypes.ErrRequired,
			"The notice needs someone to send it to",
		)
	}
	sentTo := strings.Join(recipients, ", ")
	if len(sentTo) > MaxNoaRecipientLength {
		sentTo = recipients[0]
	}
	n.Status = NoaStatusSent
	n.SentAt = &at
	n.SentTo = sentTo
	return nil
}

func (n *Noa) Acknowledge(
	at, acknowledgedAt int64,
	signedDocumentID pulid.ID,
	expiresAt *int64,
) error {
	if !n.CanAcknowledge() {
		return errortypes.NewBusinessError("Only a drafted or sent notice can be acknowledged")
	}
	if acknowledgedAt <= 0 || acknowledgedAt > at+NoaFutureSlackSeconds {
		return errortypes.NewValidationError(
			"acknowledgedAt",
			errortypes.ErrInvalid,
			"Enter the date the customer signed the notice; it cannot be in the future",
		)
	}
	if signedDocumentID.IsNil() {
		return errortypes.NewValidationError(
			"signedDocumentId",
			errortypes.ErrRequired,
			"Attach the customer's signed copy of the notice",
		)
	}
	n.Status = NoaStatusAcknowledged
	n.AcknowledgedAt = &acknowledgedAt
	n.SignedDocumentID = signedDocumentID
	n.ExpiresAt = expiresAt
	n.ExpiryWarnedAt = nil
	return nil
}

func (n *Noa) Release(at int64, reason string) error {
	reason = strings.TrimSpace(reason)
	if reason == "" {
		return errortypes.NewValidationError(
			"reason",
			errortypes.ErrRequired,
			"Say why the notice is released, such as the factor's release letter",
		)
	}
	if !n.IsOpen() {
		return errortypes.NewBusinessError("The notice is already released")
	}
	n.Status = NoaStatusReleased
	n.ReleasedAt = &at
	n.ReleaseReason = reason
	return nil
}

func (n *Noa) BeforeCreate(ctx context.Context) error {
	if n.Status == "" {
		n.Status = NoaStatusDraft
	}

	return nil
}

func (n *Noa) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias:      "facnoa",
		UseSearchVector: false,
		SearchableFields: []domaintypes.SearchableField{
			{Name: "status", Type: domaintypes.FieldTypeEnum, Weight: domaintypes.SearchWeightA},
			{Name: "sent_to", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightB},
		},
	}
}

func (n *Noa) GetID() pulid.ID             { return n.ID }
func (n *Noa) GetCreatedAt() int64         { return n.CreatedAt }
func (n *Noa) GetOrganizationID() pulid.ID { return n.OrganizationID }
func (n *Noa) GetBusinessUnitID() pulid.ID { return n.BusinessUnitID }
func (n *Noa) GetTableName() string        { return "factoring_noas" }
