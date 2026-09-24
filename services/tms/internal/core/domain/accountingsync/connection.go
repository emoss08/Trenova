package accountingsync

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	FailingAfterConsecutiveFailures = 3
	RefreshTokenWarningWindow       = int64(14 * 24 * 60 * 60)
	RefreshTokenAbsoluteLifetime    = int64(5 * 365 * 24 * 60 * 60)
	maxErrorMessageLength           = 2000
)

var _ bun.BeforeAppendModelHook = (*AccountingConnection)(nil)

type AccountingConnection struct {
	bun.BaseModel `bun:"table:accounting_connections,alias:acctc" json:"-"`

	ID                            pulid.ID         `json:"id"                             bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID                pulid.ID         `json:"businessUnitId"                 bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID                pulid.ID         `json:"organizationId"                 bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	IntegrationType               integration.Type `json:"integrationType"                bun:"integration_type,type:VARCHAR(50),notnull"`
	Status                        ConnectionStatus `json:"status"                         bun:"status,type:VARCHAR(20),notnull"`
	ExternalRealmID               string           `json:"externalRealmId"                bun:"external_realm_id,type:VARCHAR(100),notnull"`
	ExternalCompanyName           string           `json:"externalCompanyName"            bun:"external_company_name,type:VARCHAR(255),nullzero"`
	ExternalLegalName             string           `json:"externalLegalName"              bun:"external_legal_name,type:VARCHAR(255),nullzero"`
	ExternalCountry               string           `json:"externalCountry"                bun:"external_country,type:VARCHAR(10),nullzero"`
	ExternalHomeCurrency          string           `json:"externalHomeCurrency"           bun:"external_home_currency,type:VARCHAR(3),nullzero"`
	ExternalMultiCurrencyEnabled  bool             `json:"externalMultiCurrencyEnabled"   bun:"external_multi_currency_enabled,type:BOOLEAN,notnull"`
	ExternalBooksClosedThrough    *int64           `json:"externalBooksClosedThrough"     bun:"external_books_closed_through,type:BIGINT,nullzero"`
	AccessTokenCiphertext         string           `json:"-"                              bun:"access_token_ciphertext,type:TEXT,nullzero"`
	AccessTokenExpiresAt          int64            `json:"accessTokenExpiresAt"           bun:"access_token_expires_at,type:BIGINT,notnull"`
	RefreshTokenCiphertext        string           `json:"-"                              bun:"refresh_token_ciphertext,type:TEXT,nullzero"`
	RefreshTokenExpiresAt         int64            `json:"refreshTokenExpiresAt"          bun:"refresh_token_expires_at,type:BIGINT,notnull"`
	RefreshTokenAbsoluteExpiresAt int64            `json:"refreshTokenAbsoluteExpiresAt"  bun:"refresh_token_absolute_expires_at,type:BIGINT,notnull"`
	LastRefreshedAt               *int64           `json:"lastRefreshedAt"                bun:"last_refreshed_at,type:BIGINT,nullzero"`
	LastCheckedAt                 *int64           `json:"lastCheckedAt"                  bun:"last_checked_at,type:BIGINT,nullzero"`
	LastSuccessAt                 *int64           `json:"lastSuccessAt"                  bun:"last_success_at,type:BIGINT,nullzero"`
	LastFailureAt                 *int64           `json:"lastFailureAt"                  bun:"last_failure_at,type:BIGINT,nullzero"`
	ConsecutiveFailures           int              `json:"consecutiveFailures"            bun:"consecutive_failures,type:INTEGER,notnull"`
	LastErrorCategory             ErrorCategory    `json:"lastErrorCategory"              bun:"last_error_category,type:VARCHAR(30),nullzero"`
	LastErrorMessage              string           `json:"lastErrorMessage"               bun:"last_error_message,type:TEXT,nullzero"`
	LastWebhookAt                 *int64           `json:"lastWebhookAt"                  bun:"last_webhook_at,type:BIGINT,nullzero"`
	ConnectedByID                 pulid.ID         `json:"connectedById"                  bun:"connected_by_id,type:VARCHAR(100),nullzero"`
	ConnectedAt                   int64            `json:"connectedAt"                    bun:"connected_at,type:BIGINT,notnull"`
	DisconnectedByID              pulid.ID         `json:"disconnectedById"               bun:"disconnected_by_id,type:VARCHAR(100),nullzero"`
	DisconnectedAt                *int64           `json:"disconnectedAt"                 bun:"disconnected_at,type:BIGINT,nullzero"`
	Version                       int64            `json:"version"                        bun:"version,type:BIGINT"`
	CreatedAt                     int64            `json:"createdAt"                      bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                     int64            `json:"updatedAt"                      bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization   *tenant.Organization `json:"organization,omitempty"   bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit   *tenant.BusinessUnit `json:"businessUnit,omitempty"   bun:"rel:belongs-to,join:business_unit_id=id"`
	ConnectedBy    *tenant.User         `json:"connectedBy,omitempty"    bun:"rel:belongs-to,join:connected_by_id=id"`
	DisconnectedBy *tenant.User         `json:"disconnectedBy,omitempty" bun:"rel:belongs-to,join:disconnected_by_id=id"`
}

type TokenGrant struct {
	AccessTokenCiphertext  string
	AccessTokenExpiresAt   int64
	RefreshTokenCiphertext string
	RefreshTokenExpiresAt  int64
}

type CompanyFacts struct {
	CompanyName          string
	LegalName            string
	Country              string
	HomeCurrency         string
	MultiCurrencyEnabled bool
	BooksClosedThrough   *int64
}

func (c *AccountingConnection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.IntegrationType,
			validation.Required.Error("Integration type is required"),
			validation.By(func(any) error {
				if !SupportsAccountingSync(c.IntegrationType) {
					return validation.NewError("invalid", "Integration type is not an accounting system")
				}
				return nil
			}),
		),
		validation.Field(&c.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[ConnectionStatus]("Status is not a connection status"),
		),
		validation.Field(&c.ExternalRealmID,
			validation.Required.Error("Company id is required"),
			validation.Length(1, 100),
		),
		validation.Field(&c.ExternalHomeCurrency, validation.Length(0, 3)),
		validation.Field(&c.LastErrorCategory,
			domainvalidation.ValidEnum[ErrorCategory]("Error category is not recognized"),
		),
		validation.Field(&c.ConnectedAt, validation.Required.Error("Connected time is required")),
	))
}

func (c *AccountingConnection) GetTableName() string { return "accounting_connections" }

func (c *AccountingConnection) GetID() pulid.ID { return c.ID }

func (c *AccountingConnection) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *AccountingConnection) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *AccountingConnection) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("acctc_")
		}
		c.CreatedAt = now
		c.UpdatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}

func SupportsAccountingSync(typ integration.Type) bool {
	return typ == integration.TypeQuickBooksOnline
}

func (c *AccountingConnection) IsActive() bool {
	return c.Status != ConnectionStatusDisconnected && c.Status != ConnectionStatusRevoked
}

func (c *AccountingConnection) HasTokens() bool {
	return c.AccessTokenCiphertext != "" && c.RefreshTokenCiphertext != ""
}

func (c *AccountingConnection) AccessTokenExpiresWithin(now, window int64) bool {
	return c.AccessTokenExpiresAt-now <= window
}

func (c *AccountingConnection) RefreshTokenNearAbsoluteExpiry(now int64) bool {
	return c.RefreshTokenAbsoluteExpiresAt > 0 &&
		c.RefreshTokenAbsoluteExpiresAt-now <= RefreshTokenWarningWindow
}

func (c *AccountingConnection) Connect(userID pulid.ID, grant TokenGrant, now int64) {
	c.Status = ConnectionStatusConnected
	c.ConnectedByID = userID
	c.ConnectedAt = now
	c.DisconnectedByID = pulid.Nil
	c.DisconnectedAt = nil
	c.RefreshTokenAbsoluteExpiresAt = now + RefreshTokenAbsoluteLifetime
	c.ApplyGrant(grant, now)
	c.RecordSuccess(now)
}

func (c *AccountingConnection) ApplyGrant(grant TokenGrant, now int64) {
	c.AccessTokenCiphertext = grant.AccessTokenCiphertext
	c.AccessTokenExpiresAt = grant.AccessTokenExpiresAt
	c.RefreshTokenCiphertext = grant.RefreshTokenCiphertext
	c.RefreshTokenExpiresAt = grant.RefreshTokenExpiresAt
	c.LastRefreshedAt = &now
}

func (c *AccountingConnection) ApplyCompanyFacts(facts CompanyFacts) {
	c.ExternalCompanyName = facts.CompanyName
	c.ExternalLegalName = facts.LegalName
	c.ExternalCountry = facts.Country
	c.ExternalHomeCurrency = facts.HomeCurrency
	c.ExternalMultiCurrencyEnabled = facts.MultiCurrencyEnabled
	c.ExternalBooksClosedThrough = facts.BooksClosedThrough
}

func (c *AccountingConnection) RecordSuccess(now int64) {
	c.LastCheckedAt = &now
	c.LastSuccessAt = &now
	c.ConsecutiveFailures = 0
	c.LastErrorCategory = ""
	c.LastErrorMessage = ""
	if c.Status != ConnectionStatusDisconnected {
		c.Status = ConnectionStatusConnected
	}
}

func (c *AccountingConnection) RecordFailure(category ErrorCategory, message string, now int64) {
	c.LastCheckedAt = &now
	c.LastFailureAt = &now
	c.LastErrorCategory = category
	c.LastErrorMessage = stringutils.TruncateRunes(message, maxErrorMessageLength)
	if c.Status == ConnectionStatusDisconnected {
		return
	}

	if category == ErrorCategoryRevoked {
		c.Status = ConnectionStatusRevoked
		c.AccessTokenCiphertext = ""
		c.RefreshTokenCiphertext = ""
		c.AccessTokenExpiresAt = 0
		c.RefreshTokenExpiresAt = 0
		return
	}

	c.ConsecutiveFailures++
	if c.ConsecutiveFailures >= FailingAfterConsecutiveFailures {
		c.Status = ConnectionStatusFailing
		return
	}
	c.Status = ConnectionStatusDegraded
}

func (c *AccountingConnection) Disconnect(userID pulid.ID, now int64) {
	c.Status = ConnectionStatusDisconnected
	c.DisconnectedByID = userID
	c.DisconnectedAt = &now
	c.AccessTokenCiphertext = ""
	c.RefreshTokenCiphertext = ""
	c.AccessTokenExpiresAt = 0
	c.RefreshTokenExpiresAt = 0
	c.ConsecutiveFailures = 0
	c.LastErrorCategory = ""
	c.LastErrorMessage = ""
}

func (c *AccountingConnection) RecordWebhook(now int64) {
	c.LastWebhookAt = &now
}

func ProviderName(typ integration.Type) string {
	switch typ { //nolint:exhaustive // only accounting systems have a display name here
	case integration.TypeQuickBooksOnline:
		return "QuickBooks Online"
	default:
		return string(typ)
	}
}

func SetupPath(typ integration.Type) string {
	return "/admin/integrations?type=" + string(typ)
}

func (c *AccountingConnection) AgentErrorSummary() string {
	switch c.LastErrorCategory {
	case "":
		return ""
	case ErrorCategoryUnknown:
		return "The provider rejected the last call. Its own message is on the connection page."
	default:
		return c.LastErrorMessage
	}
}
