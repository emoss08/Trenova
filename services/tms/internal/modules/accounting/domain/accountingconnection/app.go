package accountingconnection

import (
	"regexp"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/accounting/domain/accountingprovider"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/integration"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/hashutils"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
)

var appClientIDPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+$`)

func AppFingerprint(environment accountingprovider.AppEnvironment, clientID string) string {
	return hashutils.SHA256Hex(
		string(environment) + ":" + strings.TrimSpace(clientID),
	)[:appFingerprintLength]
}

type AccountingAppCredential struct {
	keel.Table `keel:"accounting_app_credentials"`

	ID                        pulid.ID                          `json:"id"              keel:"id,pk,autoid:acctapp_"`
	BusinessUnitID            pulid.ID                          `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	OrganizationID            pulid.ID                          `json:"organizationId"  keel:"organization_id,pk,tenant"`
	IntegrationType           integration.Type                  `json:"integrationType" keel:"integration_type"`
	Environment               accountingprovider.AppEnvironment `json:"environment"     keel:"environment"`
	ClientID                  string                            `json:"clientId"        keel:"client_id"`
	ClientSecretCiphertext    string                            `json:"-"               keel:"client_secret_ciphertext"`
	WebhookVerifierCiphertext string                            `json:"-"               keel:"webhook_verifier_ciphertext,nullzero"`
	Fingerprint               string                            `json:"fingerprint"     keel:"fingerprint"`
	UpdatedByID               pulid.ID                          `json:"updatedById"     keel:"updated_by_id,nullzero"`
	Version                   int64                             `json:"version"         keel:"version,version"`
	CreatedAt                 int64                             `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt                 int64                             `json:"updatedAt"       keel:"updated_at,autoupdate"`

	Organization *tenant.Organization `json:"organization,omitempty"`
	BusinessUnit *tenant.BusinessUnit `json:"businessUnit,omitempty"`
	UpdatedBy    *tenant.User         `json:"updatedBy,omitempty"    keel:"belongs:updated_by_id"`
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
				if !accountingprovider.SupportsAccountingSync(a.IntegrationType) {
					return validation.NewError(
						"invalid",
						"Integration type is not an accounting system",
					)
				}
				return nil
			}),
		),
		validation.Field(
			&a.Environment,
			validation.Required.Error("Environment is required"),
			domainvalidation.ValidEnum[accountingprovider.AppEnvironment](
				"Environment must be Sandbox or Production",
			),
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

func (AccountingAppCredential) Constraints() []keel.Constraint {
	return []keel.Constraint{
		{
			Name:    "uq_accounting_app_credentials_tenant_type",
			Message: "An accounting app is already registered for this integration",
		},
	}
}
