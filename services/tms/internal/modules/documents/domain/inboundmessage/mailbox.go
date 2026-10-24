package inboundmessage

import (
	"context"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/emailutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

const (
	SigningSecretField  = "signing_secret"
	ProviderAPIKeyField = "provider_api_key"

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
	keel.Table     `keel:"inbound_mailboxes"`
	ID             pulid.ID `json:"id" keel:"id,pk,autoid:imbx_"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,pk,tenant"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,pk,tenant"`
	Name           string   `json:"name" keel:"name"`
	Address        string   `json:"address" keel:"address"`
	Provider       Provider `json:"provider" keel:"provider"`
	// TokenHash is the webhook token as stored. The token itself is shown once,
	// when the mailbox is created or rotated, and never again.
	TokenHash string `json:"-" keel:"token_hash"`
	// SigningSecret is the provider endpoint's own secret, encrypted at rest.
	//
	// It belongs to the mailbox rather than to the outbound email integration
	// because they are different endpoints with separately rotated secrets, and
	// borrowing one for the other would mean rotating outbound mail silently
	// stopped inbound mail from being verified.
	SigningSecret string `json:"-" keel:"signing_secret,nullzero"`
	// ProviderAPIKey is the key the mailbox reads message content with,
	// encrypted at rest. Resend posts only a message's metadata to the
	// webhook, so its body, headers and attachments are fetched afterwards.
	ProviderAPIKey string        `json:"-" keel:"provider_api_key,nullzero"`
	Purpose        string        `json:"purpose" keel:"purpose,nullzero"`
	ReviewPolicy   ReviewPolicy  `json:"reviewPolicy" keel:"review_policy"`
	MinConfidence  float64       `json:"minConfidence" keel:"min_confidence"`
	Status         MailboxStatus `json:"status" keel:"status"`
	LastReceivedAt int64         `json:"lastReceivedAt" keel:"last_received_at,nullzero"`
	TestToken      string        `json:"-" keel:"test_token,nullzero"`
	TestSentAt     int64         `json:"testSentAt" keel:"test_sent_at,nullzero"`
	TestReceivedAt int64         `json:"testReceivedAt" keel:"test_received_at,nullzero"`
	BlockedSenders []string      `json:"blockedSenders" keel:"blocked_senders,jsonb,nullzero"`
	Version        int64         `json:"version" keel:"version,version"`
	CreatedAt      int64         `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64         `json:"updatedAt" keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
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

func (m *Mailbox) Blocks(address string) bool {
	if len(m.BlockedSenders) == 0 {
		return false
	}
	domain := emailutils.Domain(address)
	normalized := strings.ToLower(strings.TrimSpace(address))
	for _, blocked := range m.BlockedSenders {
		if blocked == normalized {
			return true
		}
		if !strings.Contains(blocked, "@") && emailutils.DomainAllowed(domain, []string{blocked}) {
			return true
		}
	}

	return false
}

func (m *Mailbox) Health(now int64) SourceHealth {
	return HealthOf(SourceHealthInput{
		Status:         m.Status,
		CreatedAt:      m.CreatedAt,
		LastReceivedAt: m.LastReceivedAt,
		Now:            now,
	})
}

func (m *Mailbox) BeforeCreate(ctx context.Context) error {
	m.ApplyDefaults()

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

// GetPostgresSearchConfig lets somebody find a mailbox by the address itself,
// which is how anybody actually refers to one. The token hash is deliberately
// absent: it is not a thing to search for, and putting it in a tsvector would
// be putting it somewhere it can be read back.
func (m *Mailbox) GetPostgresSearchConfig() domaintypes.PostgresSearchConfig {
	return domaintypes.PostgresSearchConfig{
		TableAlias: "imbx",
		SearchableFields: []domaintypes.SearchableField{
			{Name: "address", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "name", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightA},
			{Name: "purpose", Type: domaintypes.FieldTypeText, Weight: domaintypes.SearchWeightC},
		},
	}
}

func (m *Mailbox) GetID() pulid.ID      { return m.ID }
func (m *Mailbox) GetCreatedAt() int64  { return m.CreatedAt }
func (m *Mailbox) GetTableName() string { return "inbound_mailboxes" }
