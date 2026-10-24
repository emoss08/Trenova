package iam

import (
	"context"
	"fmt"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
)

var (
	_ validationframework.TenantedEntity = (*AccessPolicy)(nil)
	_ domaintypes.PostgresSearchable     = (*AuthEvent)(nil)
	_ domaintypes.PostgresSearchable     = (*RiskDecision)(nil)
	_ domaintypes.PostgresSearchable     = (*ExternalIdentity)(nil)
	_ domaintypes.PostgresSearchable     = (*MFAAuthenticator)(nil)
)

type IdentityProvider struct {
	keel.Table `keel:"identity_providers"`

	ID                  pulid.ID                 `json:"id" keel:"id,pk,autoid:idp_"`
	OrganizationID      pulid.ID                 `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID      pulid.ID                 `json:"businessUnitId" keel:"business_unit_id,tenant"`
	Name                string                   `json:"name" keel:"name"`
	Slug                string                   `json:"slug" keel:"slug"`
	Protocol            IdentityProviderProtocol `json:"protocol" keel:"protocol"`
	Enabled             bool                     `json:"enabled" keel:"enabled"`
	EnforceSSO          bool                     `json:"enforceSso" keel:"enforce_sso"`
	AutoProvision       bool                     `json:"autoProvision" keel:"auto_provision"`
	AllowFederatedMFA   bool                     `json:"allowFederatedMfa" keel:"allow_federated_mfa"`
	FederatedMFAACRs    []string                 `json:"federatedMfaAcrValues" keel:"federated_mfa_acr_values"`
	AllowedDomains      []string                 `json:"allowedDomains" keel:"allowed_domains"`
	AttributeMap        map[string]string        `json:"attributeMap" keel:"attribute_map,jsonb"`
	OIDCIssuerURL       string                   `json:"oidcIssuerUrl" keel:"oidc_issuer_url,nullzero"`
	OIDCClientID        string                   `json:"oidcClientId" keel:"oidc_client_id,nullzero"`
	OIDCClientSecret    string                   `json:"-" keel:"oidc_client_secret,nullzero"`
	OIDCRedirectURL     string                   `json:"oidcRedirectUrl" keel:"oidc_redirect_url,nullzero"`
	OIDCScopes          []string                 `json:"oidcScopes" keel:"oidc_scopes"`
	SAMLEntityID        string                   `json:"samlEntityId" keel:"saml_entity_id,nullzero"`
	SAMLSSOURL          string                   `json:"samlSsoUrl" keel:"saml_sso_url,nullzero"`
	SAMLX509Certificate string                   `json:"-" keel:"saml_x509_certificate,nullzero"`
	SAMLMetadataXML     string                   `json:"-" keel:"saml_metadata_xml,nullzero"`
	Version             int64                    `json:"version" keel:"version,version"`
	CreatedAt           int64                    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt           int64                    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (p *IdentityProvider) BeforeUpdate(_ context.Context) error {
	p.setCollectionDefaults()

	return nil
}

func (p *IdentityProvider) BeforeCreate(_ context.Context) error {
	p.setCollectionDefaults()

	return nil
}

func (p *IdentityProvider) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		p,
		validation.Field(&p.OrganizationID, validation.Required),
		validation.Field(&p.BusinessUnitID, validation.Required),
		validation.Field(&p.Name, validation.Required, validation.Length(1, 120)),
		validation.Field(&p.Slug, validation.Required, validation.Length(1, 80)),
		validation.Field(&p.Protocol, validation.Required, validation.In(
			IdentityProviderProtocolOIDC,
			IdentityProviderProtocolSAML,
		)),
	)
	multiErr.AddOzzoError(err)
	if !p.Protocol.IsValid() {
		multiErr.Add("protocol", errortypes.ErrInvalid, "Protocol must be OIDC or SAML")
	}
	if len(p.FederatedMFAACRs) > maxFederatedMFAACRs {
		multiErr.Add(
			"federatedMfaAcrValues",
			errortypes.ErrInvalid,
			"No more than 20 trusted ACR values can be listed",
		)
	}
	for i, value := range p.FederatedMFAACRs {
		if value == "" || len(value) > maxFederatedMFAACRLength {
			multiErr.Add(
				fmt.Sprintf("federatedMfaAcrValues[%d]", i),
				errortypes.ErrInvalid,
				"Each trusted ACR value must be between 1 and 255 characters",
			)
		}
	}
}

const (
	maxFederatedMFAACRs      = 20
	maxFederatedMFAACRLength = 255
)

func (p *IdentityProvider) BeforeSave(_ context.Context) error {
	p.setCollectionDefaults()
	return nil
}

func (p *IdentityProvider) setCollectionDefaults() {
	if p.AllowedDomains == nil {
		p.AllowedDomains = []string{}
	}
	if p.FederatedMFAACRs == nil {
		p.FederatedMFAACRs = []string{}
	}
	if p.AttributeMap == nil {
		p.AttributeMap = map[string]string{}
	}
	if p.OIDCScopes == nil {
		p.OIDCScopes = defaultIdentityProviderOIDCScopes()
	}
}

func defaultIdentityProviderOIDCScopes() []string {
	return []string{"openid", "email", "profile"}
}

type ExternalIdentity struct {
	keel.Table `keel:"external_identities"`

	ID                 pulid.ID          `json:"id" keel:"id,pk,autoid:ext_"`
	UserID             pulid.ID          `json:"userId" keel:"user_id"`
	OrganizationID     pulid.ID          `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID     pulid.ID          `json:"businessUnitId" keel:"business_unit_id,tenant"`
	IdentityProviderID pulid.ID          `json:"identityProviderId" keel:"identity_provider_id"`
	ExternalSubject    string            `json:"externalSubject" keel:"external_subject"`
	ExternalUsername   string            `json:"externalUsername" keel:"external_username,nullzero"`
	ExternalEmail      string            `json:"externalEmail" keel:"external_email,nullzero"`
	RawClaims          map[string]string `json:"rawClaims" keel:"raw_claims,jsonb"`
	LastLoginAt        int64             `json:"lastLoginAt" keel:"last_login_at,nullzero"`
	CreatedAt          int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt          int64             `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (e *ExternalIdentity) BeforeUpdate(_ context.Context) error {
	if e.RawClaims == nil {
		e.RawClaims = map[string]string{}
	}

	return nil
}

func (e *ExternalIdentity) BeforeCreate(_ context.Context) error {
	if e.RawClaims == nil {
		e.RawClaims = map[string]string{}
	}

	return nil
}

type MFAAuthenticator struct {
	keel.Table `keel:"mfa_authenticators"`

	ID             pulid.ID             `json:"id" keel:"id,pk,autoid:mfa_"`
	UserID         pulid.ID             `json:"userId" keel:"user_id"`
	OrganizationID pulid.ID             `json:"organizationId" keel:"organization_id,tenant"`
	Type           MFAAuthenticatorType `json:"type" keel:"type"`
	Name           string               `json:"name" keel:"name"`
	CredentialID   string               `json:"credentialId" keel:"credential_id,nullzero"`
	SecretCipher   string               `json:"-" keel:"secret_cipher,nullzero"`
	Enabled        bool                 `json:"enabled" keel:"enabled"`
	VerifiedAt     int64                `json:"verifiedAt" keel:"verified_at,nullzero"`
	LastUsedAt     int64                `json:"lastUsedAt" keel:"last_used_at,nullzero"`
	LastUsedStep   int64                `json:"-" keel:"last_used_step"`
	CreatedAt      int64                `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64                `json:"updatedAt" keel:"updated_at,autoupdate"`
}

type AuthEvent struct {
	keel.Table `keel:"auth_events"`

	ID                 pulid.ID         `json:"id" keel:"id,pk,autoid:aue_"`
	UserID             pulid.ID         `json:"userId" keel:"user_id,nullzero"`
	OrganizationID     pulid.ID         `json:"organizationId" keel:"organization_id,nullzero"`
	BusinessUnitID     pulid.ID         `json:"businessUnitId" keel:"business_unit_id,nullzero"`
	IdentityProviderID pulid.ID         `json:"identityProviderId" keel:"identity_provider_id,nullzero"`
	Provider           string           `json:"provider" keel:"provider"`
	Outcome            AuthEventOutcome `json:"outcome" keel:"outcome"`
	IPAddress          string           `json:"ipAddress" keel:"ip_address,nullzero"`
	UserAgent          string           `json:"userAgent" keel:"user_agent,nullzero"`
	AuthenticatorAAL   int              `json:"authenticatorAal" keel:"authenticator_aal"`
	FederationFAL      int              `json:"federationFal" keel:"federation_fal"`
	MFAState           string           `json:"mfaState" keel:"mfa_state,nullzero"`
	RiskOutcome        RiskOutcome      `json:"riskOutcome" keel:"risk_outcome"`
	RiskSignals        []string         `json:"riskSignals" keel:"risk_signals"`
	ErrorCode          string           `json:"errorCode" keel:"error_code,nullzero"`
	OccurredAt         int64            `json:"occurredAt" keel:"occurred_at"`
	CreatedAt          int64            `json:"createdAt" keel:"created_at,autocreate"`
}

func (e *AuthEvent) BeforeUpdate(_ context.Context) error {
	if e.RiskSignals == nil {
		e.RiskSignals = []string{}
	}
	if e.OccurredAt == 0 {
		e.OccurredAt = e.CreatedAt
	}

	return nil
}

func (e *AuthEvent) BeforeCreate(_ context.Context) error {
	if e.RiskSignals == nil {
		e.RiskSignals = []string{}
	}
	if e.AuthenticatorAAL == 0 {
		e.AuthenticatorAAL = 1
	}
	if e.FederationFAL == 0 {
		e.FederationFAL = 1
	}
	if e.RiskOutcome == "" {
		e.RiskOutcome = RiskOutcomeAllow
	}
	if e.OccurredAt == 0 {
		e.OccurredAt = timeutils.NowUnix()
	}

	return nil
}

type RiskDecision struct {
	keel.Table `keel:"risk_decisions"`

	ID             pulid.ID    `json:"id" keel:"id,pk,autoid:rsk_"`
	UserID         pulid.ID    `json:"userId" keel:"user_id,nullzero"`
	OrganizationID pulid.ID    `json:"organizationId" keel:"organization_id,nullzero"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" keel:"business_unit_id,nullzero"`
	Outcome        RiskOutcome `json:"outcome" keel:"outcome"`
	Signals        []string    `json:"signals" keel:"signals"`
	Reason         string      `json:"reason" keel:"reason,nullzero"`
	CreatedAt      int64       `json:"createdAt" keel:"created_at,autocreate"`
}

func (d *RiskDecision) BeforeUpdate(_ context.Context) error {
	if d.Signals == nil {
		d.Signals = []string{}
	}

	return nil
}

func (d *RiskDecision) BeforeCreate(_ context.Context) error {
	if d.Signals == nil {
		d.Signals = []string{}
	}

	return nil
}

type SCIMDirectory struct {
	keel.Table `keel:"scim_directories"`

	ID             pulid.ID `json:"id" keel:"id,pk,autoid:scd_"`
	OrganizationID pulid.ID `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" keel:"business_unit_id,tenant"`
	TenantSlug     string   `json:"tenantSlug" keel:"tenant_slug"`
	Enabled        bool     `json:"enabled" keel:"enabled"`
	CreatedAt      int64    `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64    `json:"updatedAt" keel:"updated_at,autoupdate"`
}

type SCIMToken struct {
	keel.Table `keel:"scim_tokens"`

	ID             pulid.ID        `json:"id" keel:"id,pk,autoid:sct_"`
	OrganizationID pulid.ID        `json:"organizationId" keel:"organization_id,tenant"`
	DirectoryID    pulid.ID        `json:"directoryId" keel:"directory_id"`
	Name           string          `json:"name" keel:"name"`
	Prefix         string          `json:"prefix" keel:"prefix"`
	TokenHash      string          `json:"-" keel:"token_hash"`
	Status         SCIMTokenStatus `json:"status" keel:"status"`
	LastUsedAt     int64           `json:"lastUsedAt" keel:"last_used_at,nullzero"`
	ExpiresAt      int64           `json:"expiresAt" keel:"expires_at,nullzero"`
	CreatedAt      int64           `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64           `json:"updatedAt" keel:"updated_at,autoupdate"`
}

type ProvisioningAuditRecord struct {
	keel.Table `keel:"provisioning_audit_records"`

	ID             pulid.ID           `json:"id" keel:"id,pk,autoid:par_"`
	OrganizationID pulid.ID           `json:"organizationId" keel:"organization_id,tenant"`
	DirectoryID    pulid.ID           `json:"directoryId" keel:"directory_id"`
	Action         ProvisioningAction `json:"action" keel:"action"`
	ResourceType   string             `json:"resourceType" keel:"resource_type"`
	ExternalID     string             `json:"externalId" keel:"external_id,nullzero"`
	ResourceID     pulid.ID           `json:"resourceId" keel:"resource_id,nullzero"`
	Status         string             `json:"status" keel:"status"`
	ErrorMessage   string             `json:"errorMessage" keel:"error_message,nullzero"`
	CreatedAt      int64              `json:"createdAt" keel:"created_at,autocreate"`
}

type AccessPolicy struct {
	keel.Table `keel:"access_policies"`

	ID             pulid.ID          `json:"id" keel:"id,pk,autoid:pol_"`
	OrganizationID pulid.ID          `json:"organizationId" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" keel:"business_unit_id,tenant"`
	Name           string            `json:"name" keel:"name"`
	Resource       string            `json:"resource" keel:"resource"`
	Operation      string            `json:"operation" keel:"operation"`
	Effect         PolicyEffect      `json:"effect" keel:"effect"`
	Priority       int               `json:"priority" keel:"priority"`
	Conditions     map[string]string `json:"conditions" keel:"conditions,jsonb"`
	Enabled        bool              `json:"enabled" keel:"enabled"`
	CreatedAt      int64             `json:"createdAt" keel:"created_at,autocreate"`
	UpdatedAt      int64             `json:"updatedAt" keel:"updated_at,autoupdate"`
}

func (p *AccessPolicy) BeforeUpdate(_ context.Context) error {
	if p.Conditions == nil {
		p.Conditions = map[string]string{}
	}

	return nil
}

func (p *AccessPolicy) BeforeCreate(_ context.Context) error {
	if p.Conditions == nil {
		p.Conditions = map[string]string{}
	}

	return nil
}

func (p *AccessPolicy) Validate(multiErr *errortypes.MultiError) {
	err := validation.ValidateStruct(
		p,
		validation.Field(&p.OrganizationID, validation.Required),
		validation.Field(&p.BusinessUnitID, validation.Required),
		validation.Field(&p.Name, validation.Required, validation.Length(1, 120)),
		validation.Field(&p.Resource, validation.Required),
		validation.Field(&p.Operation, validation.Required),
		validation.Field(
			&p.Effect,
			validation.Required,
			validation.In(PolicyEffectAllow, PolicyEffectDeny),
		),
	)
	multiErr.AddOzzoError(err)
}

func (p *AccessPolicy) GetID() pulid.ID {
	return p.ID
}

func (p *AccessPolicy) GetOrganizationID() pulid.ID {
	return p.OrganizationID
}

func (p *AccessPolicy) GetBusinessUnitID() pulid.ID {
	return p.BusinessUnitID
}

func (p *AccessPolicy) GetTableName() string {
	return "access_policies"
}

func (i *ExternalIdentity) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(i,
		validation.Field(&i.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&i.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		validation.Field(&i.UserID, validation.Required.Error("User is required")),
		validation.Field(&i.IdentityProviderID,
			validation.Required.Error("Identity provider is required"),
		),
		// The external subject is what a login is matched on. A blank one would
		// match the next unidentified assertion from the same provider.
		validation.Field(&i.ExternalSubject,
			validation.Required.Error("External subject is required"),
			validation.Length(1, maxIAMIdentifierLength).
				Error("External subject cannot be longer than 255 characters"),
		),
		validation.Field(&i.ExternalUsername,
			validation.Length(0, maxIAMIdentifierLength).
				Error("External username cannot be longer than 255 characters"),
		),
		validation.Field(&i.ExternalEmail,
			is.EmailFormat.Error("External email must be a valid email address"),
			validation.Length(0, maxIAMIdentifierLength).
				Error("External email cannot be longer than 255 characters"),
		),
	))
}

func (a *MFAAuthenticator) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(a,
		validation.Field(&a.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&a.UserID, validation.Required.Error("User is required")),
		validation.Field(&a.Type,
			validation.Required.Error("Type is required"),
			domainvalidation.ValidEnum[MFAAuthenticatorType]("Type is invalid"),
		),
		validation.Field(&a.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxIAMNameLength).
				Error("Name cannot be longer than 120 characters"),
		),
		// A WebAuthn authenticator is identified by its credential; a TOTP one
		// by the secret it derives codes from. Either kind missing its own
		// material would accept no second factor at all.
		validation.Field(&a.CredentialID,
			validation.When(
				a.Type == MFAAuthenticatorTypeWebAuthn,
				validation.Required.Error("A WebAuthn authenticator must record its credential"),
			),
			validation.Length(0, maxCredentialIDLength).
				Error("Credential cannot be longer than 512 characters"),
		),
		validation.Field(&a.SecretCipher,
			validation.When(
				a.Type == MFAAuthenticatorTypeTOTP,
				validation.Required.Error("A TOTP authenticator must record its secret"),
			),
		),
		// Enabling an unverified authenticator would let an unproven factor
		// satisfy a challenge.
		validation.Field(&a.VerifiedAt,
			validation.When(
				a.Enabled,
				validation.Required.Error("An enabled authenticator must be verified first"),
			),
		),
	))
}

func (e *AuthEvent) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(e,
		validation.Field(&e.Provider,
			validation.Required.Error("Provider is required"),
			validation.Length(1, maxIAMNameLength).
				Error("Provider cannot be longer than 120 characters"),
		),
		validation.Field(&e.Outcome,
			validation.Required.Error("Outcome is required"),
			domainvalidation.ValidEnum[AuthEventOutcome]("Outcome is invalid"),
		),
		validation.Field(&e.IPAddress, is.IP.Error("IP address must be a valid address")),
		// The assurance levels are what a policy compares against; the scale
		// starts at one, so a zero would read as a weaker login than any real
		// one and pass checks it should fail.
		validation.Field(&e.AuthenticatorAAL,
			validation.Required.Error("Authenticator assurance level is required"),
			validation.Min(minAssuranceLevel).
				Error("Authenticator assurance level must be between 1 and 3"),
			validation.Max(maxAssuranceLevel).
				Error("Authenticator assurance level must be between 1 and 3"),
		),
		validation.Field(&e.FederationFAL,
			validation.Required.Error("Federation assurance level is required"),
			validation.Min(minAssuranceLevel).
				Error("Federation assurance level must be between 1 and 3"),
			validation.Max(maxAssuranceLevel).
				Error("Federation assurance level must be between 1 and 3"),
		),
		validation.Field(&e.RiskOutcome,
			validation.Required.Error("Risk outcome is required"),
			domainvalidation.ValidEnum[RiskOutcome]("Risk outcome is invalid"),
		),
		validation.Field(&e.ErrorCode,
			validation.Length(0, maxIAMNameLength).
				Error("Error code cannot be longer than 120 characters"),
		),
		validation.Field(&e.OccurredAt,
			validation.Required.Error("Occurred at is required"),
			validation.Min(int64(1)).Error("Occurred at must be a valid timestamp"),
		),
	))
}

func (d *RiskDecision) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.Outcome,
			validation.Required.Error("Outcome is required"),
			domainvalidation.ValidEnum[RiskOutcome]("Outcome is invalid"),
		),
		// Denying or challenging a sign-in without recording why leaves the user
		// locked out and support with nothing to look at.
		validation.Field(&d.Reason,
			validation.When(
				d.Outcome != RiskOutcomeAllow,
				validation.Required.Error("A challenge or denial must record its reason"),
			),
		),
	))
}

func (d *SCIMDirectory) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(d,
		validation.Field(&d.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&d.BusinessUnitID,
			validation.Required.Error("Business unit is required"),
		),
		// The slug appears in the SCIM endpoint the identity provider posts to,
		// so it is held to what is safe in a URL path.
		validation.Field(&d.TenantSlug,
			validation.Required.Error("Tenant slug is required"),
			validation.Length(1, maxTenantSlugLength).
				Error("Tenant slug cannot be longer than 100 characters"),
			validation.Match(tenantSlugPattern).
				Error("Tenant slug may contain only lowercase letters, numbers and hyphens"),
		),
	))
}

func (t *SCIMToken) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(t,
		validation.Field(&t.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&t.DirectoryID, validation.Required.Error("Directory is required")),
		validation.Field(&t.Name,
			validation.Required.Error("Name is required"),
			validation.Length(1, maxIAMNameLength).
				Error("Name cannot be longer than 120 characters"),
		),
		// The prefix is how a presented token is located before its hash is
		// compared, and the hash is the only stored form of the secret.
		validation.Field(&t.Prefix,
			validation.Required.Error("Prefix is required"),
			validation.Length(1, maxTokenPrefixLength).
				Error("Prefix cannot be longer than 24 characters"),
		),
		validation.Field(&t.TokenHash,
			validation.Required.Error("Token hash is required"),
			validation.Length(1, maxTokenHashLength).
				Error("Token hash cannot be longer than 128 characters"),
		),
		validation.Field(&t.Status,
			validation.Required.Error("Status is required"),
			domainvalidation.ValidEnum[SCIMTokenStatus]("Status is invalid"),
		),
	))
}

func (r *ProvisioningAuditRecord) Validate(multiErr *errortypes.MultiError) {
	multiErr.AddOzzoError(validation.ValidateStruct(r,
		validation.Field(&r.OrganizationID,
			validation.Required.Error("Organization is required"),
		),
		validation.Field(&r.DirectoryID, validation.Required.Error("Directory is required")),
		validation.Field(&r.Action,
			validation.Required.Error("Action is required"),
			domainvalidation.ValidEnum[ProvisioningAction]("Action is invalid"),
		),
		validation.Field(&r.ResourceType,
			validation.Required.Error("Resource type is required"),
			validation.Length(1, maxResourceTypeLength).
				Error("Resource type cannot be longer than 80 characters"),
		),
		validation.Field(&r.ExternalID,
			validation.Length(0, maxIAMIdentifierLength).
				Error("External id cannot be longer than 255 characters"),
		),
		validation.Field(&r.Status,
			validation.Required.Error("Status is required"),
			validation.Length(1, maxResourceTypeLength).
				Error("Status cannot be longer than 80 characters"),
		),
	))
}

var tenantSlugPattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

const (
	maxIAMIdentifierLength = 255
	maxIAMNameLength       = 120
	maxCredentialIDLength  = 512
	maxTenantSlugLength    = 100
	maxTokenPrefixLength   = 24
	maxTokenHashLength     = 128
	maxResourceTypeLength  = 80
	minAssuranceLevel      = 1
	maxAssuranceLevel      = 3
)
