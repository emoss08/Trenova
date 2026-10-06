package accountingsync

import (
	"context"
	"crypto/subtle"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*AccountingWebhookSubscription)(nil)

type AccountingWebhookSubscription struct {
	bun.BaseModel `bun:"table:accounting_webhook_subscriptions,alias:acctwhs" json:"-"`

	ID                     pulid.ID                  `json:"id"                     bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID         pulid.ID                  `json:"businessUnitId"         bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID         pulid.ID                  `json:"organizationId"         bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	ConnectionID           pulid.ID                  `json:"connectionId"           bun:"connection_id,type:VARCHAR(100),notnull"`
	IntegrationType        integration.Type          `json:"integrationType"        bun:"integration_type,type:VARCHAR(50),notnull"`
	Resource               string                    `json:"resource"               bun:"resource,type:VARCHAR(100),notnull"`
	ExternalSubscriptionID string                    `json:"externalSubscriptionId" bun:"external_subscription_id,type:VARCHAR(100),nullzero"`
	NotificationURL        string                    `json:"notificationUrl"        bun:"notification_url,type:TEXT,nullzero"`
	ClientStateCiphertext  string                    `json:"-"                      bun:"client_state_ciphertext,type:TEXT,nullzero"`
	ETag                   string                    `json:"-"                      bun:"etag,type:VARCHAR(200),nullzero"`
	Status                 WebhookSubscriptionStatus `json:"status"                 bun:"status,type:VARCHAR(20),notnull,default:'Pending'"`
	ExpiresAt              *int64                    `json:"expiresAt"              bun:"expires_at,type:BIGINT,nullzero"`
	LastAttemptAt          *int64                    `json:"lastAttemptAt"          bun:"last_attempt_at,type:BIGINT,nullzero"`
	LastError              string                    `json:"lastError"              bun:"last_error,type:TEXT,nullzero"`
	Version                int64                     `json:"version"                bun:"version,type:BIGINT"`
	CreatedAt              int64                     `json:"createdAt"              bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt              int64                     `json:"updatedAt"              bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
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

func (s *AccountingWebhookSubscription) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if s.ID.IsNil() {
			s.ID = pulid.MustNew("acctwhs_")
		}
		s.CreatedAt = now
		s.UpdatedAt = now
	case *bun.UpdateQuery:
		s.UpdatedAt = now
	}

	return nil
}
