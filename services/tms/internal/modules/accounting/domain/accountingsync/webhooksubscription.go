package accountingsync

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/shared/pulid"
)

const maxSubscriptionErrorLength = 2000

type WebhookSubscriptionStatus string

const (
	WebhookSubscriptionPending = WebhookSubscriptionStatus("Pending")
	WebhookSubscriptionActive  = WebhookSubscriptionStatus("Active")
	WebhookSubscriptionFailed  = WebhookSubscriptionStatus("Failed")
)

func (s WebhookSubscriptionStatus) String() string { return string(s) }

func (s WebhookSubscriptionStatus) IsValid() bool {
	switch s {
	case WebhookSubscriptionPending, WebhookSubscriptionActive, WebhookSubscriptionFailed:
		return true
	default:
		return false
	}
}

func AllWebhookSubscriptionStatuses() []WebhookSubscriptionStatus {
	return []WebhookSubscriptionStatus{
		WebhookSubscriptionPending,
		WebhookSubscriptionActive,
		WebhookSubscriptionFailed,
	}
}

func WebhookSubscriptionSystems() []integration.Type {
	systems := make([]integration.Type, 0, len(providerProfiles))
	for idx := range providerProfiles {
		if providerProfiles[idx].WebhookSubscriptions {
			systems = append(systems, providerProfiles[idx].Type)
		}
	}
	return systems
}

type AccountingWebhookSubscription struct {
	keel.Table `keel:"accounting_webhook_subscriptions"`

	ID                     pulid.ID                  `json:"id"                     keel:"id,pk,autoid:acctwhs_"`
	BusinessUnitID         pulid.ID                  `json:"businessUnitId"         keel:"business_unit_id,pk,tenant"`
	OrganizationID         pulid.ID                  `json:"organizationId"         keel:"organization_id,pk,tenant"`
	ConnectionID           pulid.ID                  `json:"connectionId"           keel:"connection_id"`
	IntegrationType        integration.Type          `json:"integrationType"        keel:"integration_type"`
	Resource               string                    `json:"resource"               keel:"resource"`
	ExternalSubscriptionID string                    `json:"externalSubscriptionId" keel:"external_subscription_id,nullzero"`
	NotificationURL        string                    `json:"notificationUrl"        keel:"notification_url,nullzero"`
	ClientStateCiphertext  string                    `json:"-"                      keel:"client_state_ciphertext,nullzero"`
	ETag                   string                    `json:"-"                      keel:"etag,nullzero"`
	Status                 WebhookSubscriptionStatus `json:"status"                 keel:"status"`
	ExpiresAt              *int64                    `json:"expiresAt"              keel:"expires_at"`
	LastAttemptAt          *int64                    `json:"lastAttemptAt"          keel:"last_attempt_at"`
	LastError              string                    `json:"lastError"              keel:"last_error,nullzero"`
	Version                int64                     `json:"version"                keel:"version,version"`
	CreatedAt              int64                     `json:"createdAt"              keel:"created_at,autocreate"`
	UpdatedAt              int64                     `json:"updatedAt"              keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
}

func (s *AccountingWebhookSubscription) BeforeCreate(_ context.Context) error {
	if s.Status == "" {
		s.Status = WebhookSubscriptionPending
	}

	return nil
}

type WebhookSubscriptionGrant struct {
	ExternalID      string
	NotificationURL string
	ETag            string
	ExpiresAt       time.Time
}

func (s *AccountingWebhookSubscription) Activate(grant *WebhookSubscriptionGrant, now int64) {
	expires := grant.ExpiresAt.Unix()
	s.ExternalSubscriptionID = grant.ExternalID
	s.NotificationURL = grant.NotificationURL
	s.ETag = grant.ETag
	s.ExpiresAt = &expires
	s.Status = WebhookSubscriptionActive
	s.LastAttemptAt = &now
	s.LastError = ""
}

func (s *AccountingWebhookSubscription) Fail(message string, now int64) {
	if len(message) > maxSubscriptionErrorLength {
		message = message[:maxSubscriptionErrorLength]
	}
	s.Status = WebhookSubscriptionFailed
	s.LastAttemptAt = &now
	s.LastError = message
}

func (s *AccountingWebhookSubscription) Forget() {
	s.ExternalSubscriptionID = ""
	s.ClientStateCiphertext = ""
	s.ETag = ""
	s.ExpiresAt = nil
	s.Status = WebhookSubscriptionPending
}

func (s *AccountingWebhookSubscription) IsHeld() bool {
	return s.ExternalSubscriptionID != ""
}

func (s *AccountingWebhookSubscription) RenewDue(now int64, window time.Duration) bool {
	if s.Status != WebhookSubscriptionActive || s.ExpiresAt == nil {
		return false
	}
	return *s.ExpiresAt-now <= int64(window/time.Second)
}

func (s *AccountingWebhookSubscription) Receives() bool {
	return s.Status == WebhookSubscriptionActive && s.ClientStateCiphertext != ""
}

func ClientStatesMatch(held, received string) bool {
	if held == "" || received == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(held), []byte(received)) == 1
}

func (s *AccountingWebhookSubscription) GetTableName() string {
	return "accounting_webhook_subscriptions"
}

func (s *AccountingWebhookSubscription) GetID() pulid.ID { return s.ID }

func (s *AccountingWebhookSubscription) GetOrganizationID() pulid.ID { return s.OrganizationID }

func (s *AccountingWebhookSubscription) GetBusinessUnitID() pulid.ID { return s.BusinessUnitID }
