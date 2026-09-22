package inboundmessage

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

const (
	maxMailboxNameLength = 100
	maxAddressLength     = 255
	// MinConfidenceFloor is the lowest bar a mailbox may set before it acts
	// without a person. Below this the classifier is guessing, and a mailbox
	// configured at 0.1 would be an auto-handling mailbox wearing a threshold.
	MinConfidenceFloor = 0.5
)

// Mailbox is one address we listen on.
//
// The token is in the URL the provider posts to and is the only thing that
// identifies the tenant, so it is stored hashed: a leaked database row is a
// row somebody can read, not an address they can post to. An unknown token is
// answered with a plain 404 and no side effect at all, which is what stops the
// endpoint being a probe for which addresses exist.
type Mailbox struct {
	bun.BaseModel             `bun:"table:inbound_mailboxes,alias:imbx" json:"-"`
	pagination.CursorValueSet `bun:",embed"                             json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	Name           string   `json:"name"           bun:"name,type:VARCHAR(100),notnull"`
	Address        string   `json:"address"        bun:"address,type:VARCHAR(255),notnull"`
	Provider       Provider `json:"provider"       bun:"provider,type:VARCHAR(20),notnull"`
	// TokenHash is the webhook token as stored. The token itself is shown once,
	// when the mailbox is created or rotated, and never again.
	TokenHash     string        `json:"-"             bun:"token_hash,type:VARCHAR(128),notnull"`
	Purpose       string        `json:"purpose"       bun:"purpose,type:VARCHAR(255),nullzero"`
	ReviewPolicy  ReviewPolicy  `json:"reviewPolicy"  bun:"review_policy,type:VARCHAR(30),notnull"`
	MinConfidence float64       `json:"minConfidence" bun:"min_confidence,type:NUMERIC(4,3),notnull"`
	Status        MailboxStatus `json:"status"        bun:"status,type:VARCHAR(20),notnull"`
	Version       int64         `json:"version"       bun:"version,type:BIGINT"`
	CreatedAt     int64         `json:"createdAt"     bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt     int64         `json:"updatedAt"     bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
}

func (m *Mailbox) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(m,
		validation.Field(&m.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxMailboxNameLength),
		),
		validation.Field(&m.Address,
			validation.Required.Error("Address is required"),
			validation.Length(1, maxAddressLength),
			is.EmailFormat.Error("Address must be an email address"),
		),
		validation.Field(&m.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[Provider]("Provider must be Postmark or Resend"),
		),
		validation.Field(&m.ReviewPolicy,
			validation.Required.Error("Review policy is required"),
			domainvalidation.ValidEnum[ReviewPolicy]("Review policy is not one of the three"),
		),
		validation.Field(&m.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[MailboxStatus]("Status must be Active or Inactive"),
		),
	))

	// A confidence bar only means anything on the policy that reads it, and
	// below the floor it is not a bar at all — it is auto-handling with a
	// number in front of it.
	if m.ReviewPolicy == ReviewBelowConfidence {
		if m.MinConfidence < MinConfidenceFloor || m.MinConfidence > 1 {
			multiErr.Add("minConfidence", errortypes.ErrInvalid,
				"Confidence must be between {0} and 1", MinConfidenceFloor)
		}
	}
}

// HandlesWithoutReview reports whether a classified message may be acted on.
//
// It is the one place the policy is read, so a mailbox cannot be interpreted
// one way by the workflow and another by the inbox.
func (m *Mailbox) HandlesWithoutReview(confidence float64) bool {
	switch m.ReviewPolicy {
	case ReviewAutoHandle:
		return true
	case ReviewBelowConfidence:
		return confidence >= m.MinConfidence
	default:
		return false
	}
}

func (m *Mailbox) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if m.ID.IsNil() {
			m.ID = pulid.MustNew("imbx_")
		}
		m.ApplyDefaults()
		m.CreatedAt = now
		m.UpdatedAt = now
	case *bun.UpdateQuery:
		m.UpdatedAt = now
	}

	return nil
}

// ApplyDefaults leaves a mailbox reviewing everything, because a mailbox
// somebody created and did not finish configuring should not be acting on its
// own.
func (m *Mailbox) ApplyDefaults() {
	if m.ReviewPolicy == "" {
		m.ReviewPolicy = ReviewAlways
	}
	if m.Status == "" {
		m.Status = MailboxActive
	}
}

func (m *Mailbox) GetID() pulid.ID      { return m.ID }
func (m *Mailbox) GetCreatedAt() int64  { return m.CreatedAt }
func (m *Mailbox) GetTableName() string { return "inbound_mailboxes" }
