package tenant

import (
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/httpsafe"
	"github.com/emoss08/trenova/shared/pulid"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

type SSOProtocol string

const (
	SSOProtocolOIDC SSOProtocol = "OIDC"
)

type SSOProvider string

const (
	SSOProviderAzureAD SSOProvider = "AzureAD"
	SSOProviderOkta    SSOProvider = "Okta"
)

type SSOConfig struct {
	keel.Table `keel:"sso_configs"`

	ID                pulid.ID          `json:"id"              keel:"id,pk,autoid:sso_"`
	OrganizationID    pulid.ID          `json:"organizationId"  keel:"organization_id,pk,tenant"`
	BusinessUnitID    pulid.ID          `json:"businessUnitId"  keel:"business_unit_id,pk,tenant"`
	Name              string            `json:"name"            keel:"name"`
	Provider          SSOProvider       `json:"provider"        keel:"provider"`
	Protocol          SSOProtocol       `json:"protocol"        keel:"protocol"`
	Enabled           bool              `json:"enabled"         keel:"enabled"`
	EnforceSSO        bool              `json:"enforceSso"      keel:"enforce_sso"`
	AutoProvision     bool              `json:"autoProvision"   keel:"auto_provision"`
	DefaultRole       string            `json:"defaultRole"     keel:"default_role,nullzero"`
	AllowedDomains    []string          `json:"allowedDomains"  keel:"allowed_domains"`
	AttributeMap      map[string]string `json:"attributeMap"    keel:"attribute_map,jsonb"`
	OIDCIssuerURL     string            `json:"oidcIssuerUrl"   keel:"oidc_issuer_url"`
	OIDCClientID      string            `json:"oidcClientId"    keel:"oidc_client_id"`
	OIDCClientSecret  string            `json:"-"               keel:"oidc_client_secret"`
	OIDCRedirectURL   string            `json:"oidcRedirectUrl" keel:"oidc_redirect_url"`
	OIDCScopes        []string          `json:"oidcScopes"      keel:"oidc_scopes"`
	AllowFederatedMFA bool              `json:"-"`
	FederatedMFAACRs  []string          `json:"-"`
	Version           int64             `json:"version"         keel:"version,version"`
	CreatedAt         int64             `json:"createdAt"       keel:"created_at,autocreate"`
	UpdatedAt         int64             `json:"updatedAt"       keel:"updated_at,autoupdate"`
}

func (v SSOProtocol) IsValid() bool {
	switch v {
	case SSOProtocolOIDC:
		return true
	default:
		return false
	}
}

func (v SSOProvider) IsValid() bool {
	switch v {
	case SSOProviderAzureAD,
		SSOProviderOkta:
		return true
	default:
		return false
	}
}

func (c *SSOConfig) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(c,
		validation.Field(&c.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&c.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&c.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxSSONameLength).
				Error("Name cannot be longer than 100 characters"),
		),
		validation.Field(&c.Provider,
			validation.Required.Error("Provider is required"),
			domainvalidation.ValidEnum[SSOProvider]("Provider is invalid"),
		),
		validation.Field(&c.Protocol,
			validation.Required.Error("Protocol is required"),
			domainvalidation.ValidEnum[SSOProtocol]("Protocol is invalid"),
		),
		// These four are the whole OIDC handshake. A configuration missing one
		// fails at the redirect, which is the worst place to find out — the
		// user is already out of the application by then.
		validation.Field(&c.OIDCIssuerURL,
			validation.Required.Error("Issuer URL is required"),
			is.URL.Error("Issuer URL must be a valid URL"),
			validation.Length(1, maxSSOURLLength).
				Error("Issuer URL cannot be longer than 500 characters"),
		),
		validation.Field(&c.OIDCClientID,
			validation.Required.Error("Client id is required"),
			validation.Length(1, maxSSOClientIDLength).
				Error("Client id cannot be longer than 255 characters"),
		),
		validation.Field(&c.OIDCClientSecret,
			validation.Required.Error("Client secret is required"),
		),
		validation.Field(&c.OIDCRedirectURL,
			validation.Required.Error("Redirect URL is required"),
			is.URL.Error("Redirect URL must be a valid URL"),
			validation.Length(1, maxSSOURLLength).
				Error("Redirect URL cannot be longer than 500 characters"),
		),
		validation.Field(&c.OIDCScopes,
			validation.Required.Error("At least one scope is required"),
		),
		validation.Field(&c.DefaultRole,
			validation.When(
				c.AutoProvision,
				validation.Required.Error("Auto provisioning needs a default role to assign"),
			),
			validation.Length(0, maxSSODefaultRoleLength).
				Error("Default role cannot be longer than 50 characters"),
		),
		// Enforcing SSO locks everyone out of password sign-in, so it cannot be
		// turned on while the configuration itself is disabled.
		validation.Field(&c.EnforceSSO,
			validation.When(!c.Enabled, validation.Empty.Error(
				"SSO must be enabled before it can be enforced",
			)),
		),
	))
}

const (
	maxSSONameLength        = 100
	maxSSOURLLength         = 500
	maxSSOClientIDLength    = 255
	maxSSODefaultRoleLength = 50
)

func ValidateOIDCIssuerURL(
	multiErr *errortypes.MultiError,
	field string,
	issuerURL string,
	policy httpsafe.Policy,
) {
	if strings.TrimSpace(issuerURL) == "" {
		return
	}
	parsed, err := httpsafe.ValidateHTTPSURL(issuerURL, policy)
	if err != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		multiErr.Add(
			field,
			errortypes.ErrInvalid,
			"Issuer URL must be an https address on a public network, without credentials, a query or a fragment",
		)
	}
}
