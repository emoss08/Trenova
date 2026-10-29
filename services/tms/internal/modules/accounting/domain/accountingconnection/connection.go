package accountingconnection

import (
	"context"
	"strings"
	"time"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingsync"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/stringutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

type AccountingConnection struct {
	keel.Table `keel:"accounting_connections"`

	ID                            pulid.ID                          `json:"id"                            keel:"id,pk,autoid:acctc_"`
	BusinessUnitID                pulid.ID                          `json:"businessUnitId"                keel:"business_unit_id,pk,tenant"`
	OrganizationID                pulid.ID                          `json:"organizationId"                keel:"organization_id,pk,tenant"`
	IntegrationType               integration.Type                  `json:"integrationType"               keel:"integration_type"`
	Status                        ConnectionStatus                  `json:"status"                        keel:"status"`
	ExternalRealmID               string                            `json:"externalRealmId"               keel:"external_realm_id"`
	AppSource                     AppSource                         `json:"appSource"                     keel:"app_source,nullzero"`
	AppEnvironment                accountingprovider.AppEnvironment `json:"appEnvironment"                keel:"app_environment,nullzero"`
	AppFingerprint                string                            `json:"-"                             keel:"app_fingerprint,nullzero"`
	AuthMethod                    AuthMethod                        `json:"authMethod"                    keel:"auth_method,nullzero"`
	ClientKeyCiphertext           string                            `json:"-"                             keel:"client_key_ciphertext,nullzero"`
	ClientCertificateID           string                            `json:"clientCertificateId"           keel:"client_certificate_id,nullzero"`
	ClientCertificateExpiresAt    *int64                            `json:"clientCertificateExpiresAt"    keel:"client_certificate_expires_at"`
	ExternalCompanyName           string                            `json:"externalCompanyName"           keel:"external_company_name,nullzero"`
	ExternalShortCode             string                            `json:"externalShortCode"             keel:"external_short_code,nullzero"`
	ExternalLegalName             string                            `json:"externalLegalName"             keel:"external_legal_name,nullzero"`
	ExternalCountry               string                            `json:"externalCountry"               keel:"external_country,nullzero"`
	ExternalHomeCurrency          string                            `json:"externalHomeCurrency"          keel:"external_home_currency,nullzero"`
	ExternalMultiCurrencyEnabled  bool                              `json:"externalMultiCurrencyEnabled"  keel:"external_multi_currency_enabled"`
	ExternalBooksClosedThrough    *int64                            `json:"externalBooksClosedThrough"    keel:"external_books_closed_through"`
	AccessTokenCiphertext         string                            `json:"-"                             keel:"access_token_ciphertext,nullzero"`
	AccessTokenExpiresAt          int64                             `json:"accessTokenExpiresAt"          keel:"access_token_expires_at"`
	RefreshTokenCiphertext        string                            `json:"-"                             keel:"refresh_token_ciphertext,nullzero"`
	RefreshTokenExpiresAt         int64                             `json:"refreshTokenExpiresAt"         keel:"refresh_token_expires_at"`
	RefreshTokenAbsoluteExpiresAt int64                             `json:"refreshTokenAbsoluteExpiresAt" keel:"refresh_token_absolute_expires_at"`
	LastRefreshedAt               *int64                            `json:"lastRefreshedAt"               keel:"last_refreshed_at"`
	LastCheckedAt                 *int64                            `json:"lastCheckedAt"                 keel:"last_checked_at"`
	LastSuccessAt                 *int64                            `json:"lastSuccessAt"                 keel:"last_success_at"`
	LastFailureAt                 *int64                            `json:"lastFailureAt"                 keel:"last_failure_at"`
	ConsecutiveFailures           int                               `json:"consecutiveFailures"           keel:"consecutive_failures"`
	LastErrorCategory             accountingsync.ErrorCategory      `json:"lastErrorCategory"             keel:"last_error_category,nullzero"`
	LastErrorMessage              string                            `json:"lastErrorMessage"              keel:"last_error_message,nullzero"`
	LastWebhookAt                 *int64                            `json:"lastWebhookAt"                 keel:"last_webhook_at"`
	SetupStep                     SetupStep                         `json:"setupStep"                     keel:"setup_step"`
	ReferenceRefreshStartedAt     *int64                            `json:"referenceRefreshStartedAt"     keel:"reference_refresh_started_at"`
	ReferenceRefreshedAt          *int64                            `json:"referenceRefreshedAt"          keel:"reference_refreshed_at"`
	ReferenceRefreshError         string                            `json:"referenceRefreshError"         keel:"reference_refresh_error,nullzero"`
	SyncStartDate                 *int64                            `json:"syncStartDate"                 keel:"sync_start_date"`
	SyncEnabledAt                 *int64                            `json:"syncEnabledAt"                 keel:"sync_enabled_at"`
	AutoSync                      bool                              `json:"autoSync"                      keel:"auto_sync"`
	DriverSettlementsEnabledAt    *int64                            `json:"driverSettlementsEnabledAt"    keel:"driver_settlements_enabled_at"`
	InboundPaymentPolicy          InboundPaymentPolicy              `json:"inboundPaymentPolicy"          keel:"inbound_payment_policy,nullzero"`
	SyncMode                      accountingprovider.SyncMode       `json:"syncMode"                      keel:"sync_mode,nullzero"`
	LedgerGranularity             LedgerGranularity                 `json:"ledgerGranularity"             keel:"ledger_granularity,nullzero"`
	LedgerOpeningBalancesSentAt   *int64                            `json:"ledgerOpeningBalancesSentAt"   keel:"ledger_opening_balances_sent_at"`
	ExternalFiscalYearStartMonth  int                               `json:"externalFiscalYearStartMonth"  keel:"external_fiscal_year_start_month,nullzero"`
	ChangeCursor                  string                            `json:"-"                             keel:"change_cursor,nullzero"`
	ChangesReadAt                 *int64                            `json:"changesReadAt"                 keel:"changes_read_at"`
	ChangesErrorCategory          accountingsync.SyncErrorCategory  `json:"changesErrorCategory"          keel:"changes_error_category,nullzero"`
	ChangesErrorMessage           string                            `json:"changesErrorMessage"           keel:"changes_error_message,nullzero"`
	DriftCheckedAt                *int64                            `json:"driftCheckedAt"                keel:"drift_checked_at"`
	DriftErrorCategory            accountingsync.SyncErrorCategory  `json:"driftErrorCategory"            keel:"drift_error_category,nullzero"`
	DriftErrorMessage             string                            `json:"driftErrorMessage"             keel:"drift_error_message,nullzero"`
	PausedAt                      *int64                            `json:"pausedAt"                      keel:"paused_at"`
	PausedByID                    pulid.ID                          `json:"pausedById"                    keel:"paused_by_id,nullzero"`
	PausedReason                  string                            `json:"pausedReason"                  keel:"paused_reason,nullzero"`
	ConnectedByID                 pulid.ID                          `json:"connectedById"                 keel:"connected_by_id,nullzero"`
	ConnectedAt                   int64                             `json:"connectedAt"                   keel:"connected_at"`
	DisconnectedByID              pulid.ID                          `json:"disconnectedById"              keel:"disconnected_by_id,nullzero"`
	DisconnectedAt                *int64                            `json:"disconnectedAt"                keel:"disconnected_at"`
	Version                       int64                             `json:"version"                       keel:"version,version"`
	CreatedAt                     int64                             `json:"createdAt"                     keel:"created_at,autocreate"`
	UpdatedAt                     int64                             `json:"updatedAt"                     keel:"updated_at,autoupdate"`

	Organization   *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit   *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	ConnectedBy    *tenant.User         `json:"connectedBy,omitempty"    keel:"belongs:connected_by_id"`
	DisconnectedBy *tenant.User         `json:"disconnectedBy,omitempty" keel:"belongs:disconnected_by_id"`
	PausedBy       *tenant.User         `json:"pausedBy,omitempty"       keel:"belongs:paused_by_id"`
}

func (c *AccountingConnection) BeforeCreate(_ context.Context) error {
	if c.AppSource == "" {
		c.AppSource = AppSourceInstance
	}
	if c.AuthMethod == "" {
		c.AuthMethod = AuthMethodAuthorization
	}
	if c.InboundPaymentPolicy == "" {
		c.InboundPaymentPolicy = InboundPaymentsPropose
	}
	if c.SyncMode == "" {
		c.SyncMode = accountingprovider.SyncModeDocument
	}
	if c.ExternalFiscalYearStartMonth == 0 {
		c.ExternalFiscalYearStartMonth = 1
	}

	return nil
}

type ClientCertificate struct {
	KeyCiphertext string
	CertificateID string
	ExpiresAt     int64
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
	FiscalYearStartMonth time.Month
	ShortCode            string
}

func (c *AccountingConnection) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.IntegrationType,
			validation.Required.Error("Integration type is required"),
			validation.By(func(any) error {
				if !accountingprovider.SupportsAccountingSync(c.IntegrationType) {
					return validation.NewError(
						"invalid",
						"Integration type is not an accounting system",
					)
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
		validation.Field(&c.AppSource,
			validation.Required.Error("App source is required"),
			domainvalidation.ValidEnum[AppSource]("App source is not recognized"),
		),
		validation.Field(
			&c.AppEnvironment,
			domainvalidation.ValidEnum[accountingprovider.AppEnvironment](
				"App environment is not recognized",
			),
		),
		validation.Field(&c.AuthMethod,
			domainvalidation.ValidEnum[AuthMethod]("Connection method is not recognized"),
		),
		validation.Field(&c.ClientCertificateID,
			validation.When(
				c.UsesCertificate(),
				validation.Required.Error("Certificate ID is required"),
			),
			validation.Length(0, 100).Error("Certificate ID cannot be longer than 100 characters"),
		),
		validation.Field(&c.ExternalHomeCurrency, validation.Length(0, 3)),
		validation.Field(
			&c.LastErrorCategory,
			domainvalidation.ValidEnum[accountingsync.ErrorCategory](
				"Error category is not recognized",
			),
		),
		validation.Field(&c.ConnectedAt, validation.Required.Error("Connected time is required")),
		validation.Field(&c.SetupStep,
			validation.Required.Error("Setup step is required"),
			domainvalidation.ValidEnum[SetupStep]("Setup step is not recognized"),
		),
		validation.Field(&c.SyncStartDate, validation.By(func(any) error {
			if c.SetupStep == SetupStepComplete &&
				(c.SyncStartDate == nil || c.SyncEnabledAt == nil) {
				return validation.NewError(
					"required",
					"A start date is required before documents are sent",
				)
			}
			return nil
		})),
		validation.Field(&c.PausedReason, validation.Length(0, maxPausedReasonLength)),
		validation.Field(&c.SyncMode,
			domainvalidation.ValidEnum[accountingprovider.SyncMode]("Sync mode is not recognized"),
		),
		validation.Field(&c.LedgerGranularity,
			domainvalidation.ValidEnum[LedgerGranularity]("Journal detail is not recognized"),
			validation.By(func(any) error {
				if c.SendsLedger() && !c.LedgerGranularity.IsValid() {
					return validation.NewError("required", ErrGranularityRequired.Error())
				}
				return nil
			}),
		),
		validation.Field(&c.ExternalFiscalYearStartMonth, validation.Min(0), validation.Max(12)),
	))
}

func (c *AccountingConnection) GetTableName() string { return "accounting_connections" }

func (c *AccountingConnection) GetID() pulid.ID { return c.ID }

func (c *AccountingConnection) GetOrganizationID() pulid.ID { return c.OrganizationID }

func (c *AccountingConnection) GetBusinessUnitID() pulid.ID { return c.BusinessUnitID }

func (c *AccountingConnection) IsActive() bool {
	return c.Status != ConnectionStatusDisconnected && c.Status != ConnectionStatusRevoked
}

func (c *AccountingConnection) UsesCertificate() bool {
	return c.AuthMethod == AuthMethodClientCredentials
}

func (c *AccountingConnection) HasTokens() bool {
	if c.UsesCertificate() {
		return c.AccessTokenCiphertext != "" && c.HasCertificate()
	}
	return c.AccessTokenCiphertext != "" && c.RefreshTokenCiphertext != ""
}

func (c *AccountingConnection) HasCertificate() bool {
	return c.ClientKeyCiphertext != "" && c.ClientCertificateID != ""
}

func (c *AccountingConnection) UseCertificate(cert ClientCertificate) {
	c.AuthMethod = AuthMethodClientCredentials
	c.ClientKeyCiphertext = cert.KeyCiphertext
	c.ClientCertificateID = cert.CertificateID
	expiresAt := cert.ExpiresAt
	c.ClientCertificateExpiresAt = &expiresAt
	c.RefreshTokenAbsoluteExpiresAt = expiresAt
	c.RefreshTokenCiphertext = ""
	c.RefreshTokenExpiresAt = 0
}

func (c *AccountingConnection) UseAuthorization() {
	c.AuthMethod = AuthMethodAuthorization
	c.ClientKeyCiphertext = ""
	c.ClientCertificateID = ""
	c.ClientCertificateExpiresAt = nil
}

func (c *AccountingConnection) AccessTokenExpiresWithin(now, window int64) bool {
	return c.AccessTokenExpiresAt-now <= window
}

func (c *AccountingConnection) ReconnectWarningSeconds() int64 {
	if c.UsesCertificate() {
		return int64(accountingprovider.DefaultReconnectWarning / time.Second)
	}
	profile := accountingprovider.MustProfile(c.IntegrationType)
	return int64(profile.ReconnectWarningSeconds())
}

func (c *AccountingConnection) RefreshTokenNearAbsoluteExpiry(now int64) bool {
	return c.RefreshTokenAbsoluteExpiresAt > 0 &&
		c.RefreshTokenAbsoluteExpiresAt-now <= c.ReconnectWarningSeconds()
}

type AppIdentity struct {
	Source      AppSource
	Environment accountingprovider.AppEnvironment
	Fingerprint string
}

func (c *AccountingConnection) BindApp(app AppIdentity) {
	c.AppSource = app.Source
	c.AppEnvironment = app.Environment
	c.AppFingerprint = app.Fingerprint
}

func (c *AccountingConnection) ConnectedThrough(app AppIdentity) bool {
	if c.AppSource != app.Source {
		return false
	}
	if c.AppFingerprint == "" {
		return app.Source == AppSourceInstance
	}
	return c.AppFingerprint == app.Fingerprint
}

func (c *AccountingConnection) Connect(userID pulid.ID, grant TokenGrant, now int64) {
	c.Status = ConnectionStatusConnected
	c.ConnectedByID = userID
	c.ConnectedAt = now
	c.DisconnectedByID = pulid.Nil
	c.DisconnectedAt = nil
	c.RefreshTokenAbsoluteExpiresAt = 0
	profile := accountingprovider.MustProfile(c.IntegrationType)
	switch {
	case c.UsesCertificate():
		if c.ClientCertificateExpiresAt != nil {
			c.RefreshTokenAbsoluteExpiresAt = *c.ClientCertificateExpiresAt
		}
	case profile.RefreshTokenAbsoluteLifetime > 0:
		c.RefreshTokenAbsoluteExpiresAt = now + int64(
			profile.RefreshTokenAbsoluteLifetime/time.Second,
		)
	}
	if c.SetupStep == "" {
		c.SetupStep = SetupStepMode
	}
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

func (c *AccountingConnection) ApplyCompanyFacts(facts *CompanyFacts) {
	c.ExternalCompanyName = facts.CompanyName
	c.ExternalShortCode = facts.ShortCode
	c.ExternalLegalName = facts.LegalName
	c.ExternalCountry = facts.Country
	c.ExternalHomeCurrency = facts.HomeCurrency
	c.ExternalMultiCurrencyEnabled = facts.MultiCurrencyEnabled
	c.ExternalBooksClosedThrough = facts.BooksClosedThrough
	if facts.FiscalYearStartMonth >= time.January && facts.FiscalYearStartMonth <= time.December {
		c.ExternalFiscalYearStartMonth = int(facts.FiscalYearStartMonth)
	}
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

func (c *AccountingConnection) RecordFailure(
	category accountingsync.ErrorCategory,
	message string,
	now int64,
) {
	c.LastCheckedAt = &now
	c.LastFailureAt = &now
	c.LastErrorCategory = category
	c.LastErrorMessage = stringutils.TruncateRunes(message, maxErrorMessageLength)
	if c.Status == ConnectionStatusDisconnected {
		return
	}

	if category == accountingsync.ErrorCategoryRevoked {
		c.Status = ConnectionStatusRevoked
		c.AccessTokenCiphertext = ""
		c.RefreshTokenCiphertext = ""
		c.ClientKeyCiphertext = ""
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
	c.ClientKeyCiphertext = ""
	c.AccessTokenExpiresAt = 0
	c.RefreshTokenExpiresAt = 0
	c.ConsecutiveFailures = 0
	c.LastErrorCategory = ""
	c.LastErrorMessage = ""
}

func (c *AccountingConnection) IsSyncing() bool {
	return c.IsActive() && c.SetupStep == SetupStepComplete &&
		c.SyncStartDate != nil && c.SyncEnabledAt != nil
}

func (c *AccountingConnection) IsPaused() bool {
	return c.PausedAt != nil
}

func (c *AccountingConnection) CanDispatch() bool {
	return c.IsSyncing() && !c.IsPaused()
}

func (c *AccountingConnection) Covers(documentDate int64) bool {
	return c.SyncStartDate != nil && documentDate >= *c.SyncStartDate
}

func (c *AccountingConnection) BooksClosedOn(documentDate int64) bool {
	return c.ExternalBooksClosedThrough != nil && documentDate <= *c.ExternalBooksClosedThrough
}

func (c *AccountingConnection) FirstOpenDay(loc *time.Location) (int64, bool) {
	if c.ExternalBooksClosedThrough == nil {
		return 0, false
	}
	if loc == nil {
		loc = time.UTC
	}
	closed := *c.ExternalBooksClosedThrough
	year, month, day := time.Unix(closed, 0).UTC().Date()
	next := time.Date(year, month, day+1, 0, 0, 0, 0, loc).Unix()
	return max(next, closed+1), true
}

func (c *AccountingConnection) FinishMappings() bool {
	if c.SetupStep != SetupStepMappings {
		return false
	}
	c.SetupStep = SetupStepStartDate
	return true
}

type SyncSettings struct {
	StartDate         int64
	AutoSync          bool
	DriverSettlements bool
	InboundPayments   InboundPaymentPolicy
	OpeningBalances   bool
}

func (c *AccountingConnection) EnableSync(settings SyncSettings, now int64) {
	startDate := settings.StartDate
	c.SyncStartDate = &startDate
	c.AutoSync = settings.AutoSync
	c.SetDriverSettlements(settings.DriverSettlements, now)
	c.SetInboundPayments(settings.InboundPayments)
	if c.SyncEnabledAt == nil {
		c.SyncEnabledAt = &now
		if settings.OpeningBalances && c.SendsLedger() {
			c.LedgerOpeningBalancesSentAt = &now
		}
	}
	c.SetupStep = SetupStepComplete
}

func (c *AccountingConnection) SyncsDriverSettlements() bool {
	return c.DriverSettlementsEnabledAt != nil
}

func (c *AccountingConnection) SetDriverSettlements(enabled bool, now int64) {
	switch {
	case !enabled:
		c.DriverSettlementsEnabledAt = nil
	case c.DriverSettlementsEnabledAt == nil:
		c.DriverSettlementsEnabledAt = &now
	}
}

func (c *AccountingConnection) SetInboundPayments(policy InboundPaymentPolicy) {
	if !c.ReadsPayments() {
		c.InboundPaymentPolicy = InboundPaymentsOff
		return
	}
	if policy.IsValid() {
		c.InboundPaymentPolicy = policy
	}
}

func (c *AccountingConnection) ReadsPayments() bool {
	profile := accountingprovider.MustProfile(c.IntegrationType)
	return profile.InboundPayments
}

func (c *AccountingConnection) PaymentPolicy() InboundPaymentPolicy {
	if !c.ReadsPayments() {
		return InboundPaymentsOff
	}
	if c.InboundPaymentPolicy.IsValid() {
		return c.InboundPaymentPolicy
	}
	return InboundPaymentsPropose
}

func (c *AccountingConnection) ReadsChanges() bool {
	return c.CanDispatch()
}

func (c *AccountingConnection) StartChangeFeed(cursor string) bool {
	if c.ChangeCursor != "" {
		return false
	}
	c.ChangeCursor = cursor
	return true
}

func (c *AccountingConnection) AdvanceChanges(cursor string, now int64) {
	if cursor != "" {
		c.ChangeCursor = cursor
	}
	c.ChangesReadAt = &now
	c.ChangesErrorCategory = ""
	c.ChangesErrorMessage = ""
}

func (c *AccountingConnection) RecordChangesFailure(
	category accountingsync.SyncErrorCategory,
	message string,
) {
	c.ChangesErrorCategory = category
	c.ChangesErrorMessage = stringutils.TruncateRunes(message, maxChangesErrorMessage)
}

func (c *AccountingConnection) ChecksDrift() bool {
	return c.CanDispatch()
}

func (c *AccountingConnection) FinishDriftCheck(now int64) {
	c.DriftCheckedAt = &now
	c.DriftErrorCategory = ""
	c.DriftErrorMessage = ""
}

func (c *AccountingConnection) RecordDriftFailure(
	category accountingsync.SyncErrorCategory,
	message string,
) {
	c.DriftErrorCategory = category
	c.DriftErrorMessage = stringutils.TruncateRunes(message, maxChangesErrorMessage)
}

func (c *AccountingConnection) Pause(userID pulid.ID, reason string, now int64) {
	c.PausedAt = &now
	c.PausedByID = userID
	c.PausedReason = stringutils.TruncateRunes(strings.TrimSpace(reason), maxPausedReasonLength)
}

func (c *AccountingConnection) Resume() {
	c.PausedAt = nil
	c.PausedByID = pulid.Nil
	c.PausedReason = ""
}

func (c *AccountingConnection) RecordWebhook(now int64) {
	c.LastWebhookAt = &now
}

func (c *AccountingConnection) AgentErrorSummary() string {
	if c.LastErrorCategory == "" {
		return ""
	}
	if c.LastErrorCategory == accountingsync.ErrorCategoryUnknown {
		return "The provider rejected the last call. Its own message is on the connection page."
	}

	return c.LastErrorMessage
}
