package accountingsync

import (
	"context"
	"regexp"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/integration"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

const (
	maxAppClientIDLength = 255
	appFingerprintLength = 32
)

var appClientIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

type AppSource string

const (
	AppSourceInstance = AppSource("Instance")
	AppSourceTenant   = AppSource("Tenant")
)

func (s AppSource) String() string { return string(s) }

func (s AppSource) IsValid() bool {
	switch s {
	case AppSourceInstance, AppSourceTenant:
		return true
	default:
		return false
	}
}

type AppEnvironment string

const (
	AppEnvironmentSandbox    = AppEnvironment("Sandbox")
	AppEnvironmentProduction = AppEnvironment("Production")
)

func (e AppEnvironment) String() string { return string(e) }

func (e AppEnvironment) IsValid() bool {
	switch e {
	case AppEnvironmentSandbox, AppEnvironmentProduction:
		return true
	default:
		return false
	}
}

func ParseAppEnvironment(value string) (AppEnvironment, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "sandbox":
		return AppEnvironmentSandbox, true
	case "production":
		return AppEnvironmentProduction, true
	default:
		return "", false
	}
}

func AppFingerprint(environment AppEnvironment, clientID string) string {
	return hashutils.SHA256Hex(
		string(environment) + ":" + strings.TrimSpace(clientID),
	)[:appFingerprintLength]
}

var _ bun.BeforeAppendModelHook = (*AccountingAppCredential)(nil)

type AccountingAppCredential struct {
	bun.BaseModel `bun:"table:accounting_app_credentials,alias:acctapp" json:"-"`

	ID                        pulid.ID         `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	BusinessUnitID            pulid.ID         `json:"businessUnitId"  bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	OrganizationID            pulid.ID         `json:"organizationId"  bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	IntegrationType           integration.Type `json:"integrationType" bun:"integration_type,type:VARCHAR(50),notnull"`
	Environment               AppEnvironment   `json:"environment"     bun:"environment,type:VARCHAR(20),notnull"`
	ClientID                  string           `json:"clientId"        bun:"client_id,type:VARCHAR(255),notnull"`
	ClientSecretCiphertext    string           `json:"-"               bun:"client_secret_ciphertext,type:TEXT,notnull"`
	WebhookVerifierCiphertext string           `json:"-"               bun:"webhook_verifier_ciphertext,type:TEXT,nullzero"`
	Fingerprint               string           `json:"fingerprint"     bun:"fingerprint,type:VARCHAR(64),notnull"`
	UpdatedByID               pulid.ID         `json:"updatedById"     bun:"updated_by_id,type:VARCHAR(100),nullzero"`
	Version                   int64            `json:"version"         bun:"version,type:BIGINT"`
	CreatedAt                 int64            `json:"createdAt"       bun:"created_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt                 int64            `json:"updatedAt"       bun:"updated_at,nullzero,notnull,default:extract(epoch from current_timestamp)::bigint"`

	Organization *tenant.Organization `json:"organization,omitempty" bun:"rel:belongs-to,join:organization_id=id"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty" bun:"rel:belongs-to,join:business_unit_id=id"`
	UpdatedBy    *tenant.User         `json:"updatedBy,omitempty"    bun:"rel:belongs-to,join:updated_by_id=id"`
}

func (a *AccountingAppCredential) HasWebhookVerifier() bool {
	return a.WebhookVerifierCiphertext != ""
}

func (a *AccountingAppCredential) Stamp() {
	a.ClientID = strings.TrimSpace(a.ClientID)
	a.Fingerprint = AppFingerprint(a.Environment, a.ClientID)
}

func (a *AccountingAppCredential) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.IntegrationType,
			validation.Required.Error("Integration type is required"),
			validation.By(func(any) error {
				if !SupportsAccountingSync(a.IntegrationType) {
					return validation.NewError(
						"invalid",
						"Integration type is not an accounting system",
					)
				}
				return nil
			}),
		),
		validation.Field(&a.Environment,
			validation.Required.Error("Environment is required"),
			domainvalidation.ValidEnum[AppEnvironment]("Environment must be Sandbox or Production"),
		),
		validation.Field(&a.ClientID,
			validation.Required.Error("Client ID is required"),
			validation.Length(1, maxAppClientIDLength).
				Error("Client ID cannot be longer than 255 characters"),
			validation.Match(appClientIDPattern).
				Error("Client ID can only contain letters, numbers, dots, dashes and underscores"),
		),
		validation.Field(&a.ClientSecretCiphertext,
			validation.Required.Error("Client secret is required"),
		),
	))
}

func (a *AccountingAppCredential) GetTableName() string { return "accounting_app_credentials" }

func (a *AccountingAppCredential) GetID() pulid.ID { return a.ID }

func (a *AccountingAppCredential) GetOrganizationID() pulid.ID { return a.OrganizationID }

func (a *AccountingAppCredential) GetBusinessUnitID() pulid.ID { return a.BusinessUnitID }

func (a *AccountingAppCredential) BeforeAppendModel(_ context.Context, query bun.Query) error {
	now := timeutils.NowUnix()

	switch query.(type) {
	case *bun.InsertQuery:
		if a.ID.IsNil() {
			a.ID = pulid.MustNew("acctapp_")
		}
		a.CreatedAt = now
		a.UpdatedAt = now
	case *bun.UpdateQuery:
		a.UpdatedAt = now
	}

	return nil
}

func WebhookPath(typ integration.Type) string {
	switch typ { //nolint:exhaustive // only accounting systems take webhooks here
	case integration.TypeQuickBooksOnline:
		return "/webhooks/accounting/quickbooks/"
	default:
		return ""
	}
}
