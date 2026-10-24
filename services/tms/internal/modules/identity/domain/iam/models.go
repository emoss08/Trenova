package iam

import (
	"context"
	"fmt"
	"regexp"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/pkg/domaintypes"
	"github.com/emoss08/trenova/pkg/domainvalidation"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/validationframework"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/go-ozzo/ozzo-validation/v4/is"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook          = (*IdentityProvider)(nil)
	_ bun.BeforeAppendModelHook          = (*ExternalIdentity)(nil)
	_ bun.BeforeAppendModelHook          = (*MFAAuthenticator)(nil)
	_ bun.BeforeAppendModelHook          = (*AuthEvent)(nil)
	_ bun.BeforeAppendModelHook          = (*RiskDecision)(nil)
	_ bun.BeforeAppendModelHook          = (*SCIMDirectory)(nil)
	_ bun.BeforeAppendModelHook          = (*SCIMToken)(nil)
	_ bun.BeforeAppendModelHook          = (*SCIMGroupRoleMapping)(nil)
	_ bun.BeforeAppendModelHook          = (*ProvisioningAuditRecord)(nil)
	_ bun.BeforeAppendModelHook          = (*AccessPolicy)(nil)
	_ validationframework.TenantedEntity = (*AccessPolicy)(nil)
	_ domaintypes.PostgresSearchable     = (*AuthEvent)(nil)
	_ domaintypes.PostgresSearchable     = (*RiskDecision)(nil)
	_ domaintypes.PostgresSearchable     = (*ExternalIdentity)(nil)
	_ domaintypes.PostgresSearchable     = (*MFAAuthenticator)(nil)
)

type IdentityProvider struct {
	bun.BaseModel `bun:"table:identity_providers,alias:idp" json:"-"`
	keel.Table    `keel:"identity_providers"`

	ID                  pulid.ID                 `json:"id"                    bun:"id,pk,type:VARCHAR(100)" keel:"id,pk,autoid:idp_"`
	OrganizationID      pulid.ID                 `json:"organizationId"        bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	BusinessUnitID      pulid.ID                 `json:"businessUnitId"        bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	Name                string                   `json:"name"                  bun:"name,type:VARCHAR(120),notnull" keel:"name"`
	Slug                string                   `json:"slug"                  bun:"slug,type:VARCHAR(80),notnull" keel:"slug"`
	Protocol            IdentityProviderProtocol `json:"protocol"              bun:"protocol,type:iam_identity_provider_protocol_enum,notnull" keel:"protocol"`
	Enabled             bool                     `json:"enabled"               bun:"enabled,notnull" keel:"enabled"`
	EnforceSSO          bool                     `json:"enforceSso"            bun:"enforce_sso,notnull" keel:"enforce_sso"`
	AutoProvision       bool                     `json:"autoProvision"         bun:"auto_provision,notnull" keel:"auto_provision"`
	AllowFederatedMFA   bool                     `json:"allowFederatedMfa"     bun:"allow_federated_mfa,notnull" keel:"allow_federated_mfa"`
	FederatedMFAACRs    []string                 `json:"federatedMfaAcrValues" bun:"federated_mfa_acr_values,array,notnull" keel:"federated_mfa_acr_values"`
	AllowedDomains      []string                 `json:"allowedDomains"        bun:"allowed_domains,array" keel:"allowed_domains"`
	AttributeMap        map[string]string        `json:"attributeMap"          bun:"attribute_map,type:JSONB,notnull" keel:"attribute_map,jsonb"`
	OIDCIssuerURL       string                   `json:"oidcIssuerUrl"         bun:"oidc_issuer_url,type:VARCHAR(500)" keel:"oidc_issuer_url,nullzero"`
	OIDCClientID        string                   `json:"oidcClientId"          bun:"oidc_client_id,type:VARCHAR(255)" keel:"oidc_client_id,nullzero"`
	OIDCClientSecret    string                   `json:"-"                     bun:"oidc_client_secret,type:TEXT" keel:"oidc_client_secret,nullzero"`
	OIDCRedirectURL     string                   `json:"oidcRedirectUrl"       bun:"oidc_redirect_url,type:VARCHAR(500)" keel:"oidc_redirect_url,nullzero"`
	OIDCScopes          []string                 `json:"oidcScopes"            bun:"oidc_scopes,array" keel:"oidc_scopes"`
	SAMLEntityID        string                   `json:"samlEntityId"          bun:"saml_entity_id,type:VARCHAR(500)" keel:"saml_entity_id,nullzero"`
	SAMLSSOURL          string                   `json:"samlSsoUrl"            bun:"saml_sso_url,type:VARCHAR(500)" keel:"saml_sso_url,nullzero"`
	SAMLX509Certificate string                   `json:"-"                     bun:"saml_x509_certificate,type:TEXT" keel:"saml_x509_certificate,nullzero"`
	SAMLMetadataXML     string                   `json:"-"                     bun:"saml_metadata_xml,type:TEXT" keel:"saml_metadata_xml,nullzero"`
	Version             int64                    `json:"version"               bun:"version,type:BIGINT,notnull" keel:"version,version"`
	CreatedAt           int64                    `json:"createdAt"             bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at,autocreate"`
	UpdatedAt           int64                    `json:"updatedAt"             bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at,autoupdate"`
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

func (p *IdentityProvider) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()
	switch q.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			p.ID = pulid.MustNew("idp_")
		}
		p.setCollectionDefaults()
		p.CreatedAt = now
	case *bun.UpdateQuery:
		p.setCollectionDefaults()
		p.UpdatedAt = now
	}
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
	bun.BaseModel `bun:"table:external_identities,alias:extid" json:"-"`
	keel.Table    `keel:"external_identities"`

	pagination.CursorValueSet `bun:",embed" json:"-"`

	ID                 pulid.ID          `json:"id"                 bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	UserID             pulid.ID          `json:"userId"             bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	OrganizationID     pulid.ID          `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	BusinessUnitID     pulid.ID          `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	IdentityProviderID pulid.ID          `json:"identityProviderId" bun:"identity_provider_id,type:VARCHAR(100),notnull" keel:"identity_provider_id"`
	ExternalSubject    string            `json:"externalSubject"    bun:"external_subject,type:VARCHAR(255),notnull" keel:"external_subject"`
	ExternalUsername   string            `json:"externalUsername"   bun:"external_username,type:VARCHAR(255)" keel:"external_username,nullzero"`
	ExternalEmail      string            `json:"externalEmail"      bun:"external_email,type:VARCHAR(255)" keel:"external_email,nullzero"`
	RawClaims          map[string]string `json:"rawClaims"          bun:"raw_claims,type:JSONB,notnull" keel:"raw_claims,jsonb"`
	LastLoginAt        int64             `json:"lastLoginAt"        bun:"last_login_at" keel:"last_login_at,nullzero"`
	CreatedAt          int64             `json:"createdAt"          bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt          int64             `json:"updatedAt"          bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

func (e *ExternalIdentity) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if e.RawClaims == nil {
		e.RawClaims = map[string]string{}
	}
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &e.ID,
		IDPrefix:  "ext_",
		CreatedAt: &e.CreatedAt,
		UpdatedAt: &e.UpdatedAt,
	})
	return nil
}

type MFAAuthenticator struct {
	bun.BaseModel `bun:"table:mfa_authenticators,alias:mfa" json:"-"`
	keel.Table    `keel:"mfa_authenticators"`

	pagination.CursorValueSet `bun:",embed" json:"-"`

	ID             pulid.ID             `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	UserID         pulid.ID             `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull" keel:"user_id"`
	OrganizationID pulid.ID             `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	Type           MFAAuthenticatorType `json:"type"           bun:"type,type:iam_mfa_authenticator_type_enum,notnull" keel:"type"`
	Name           string               `json:"name"           bun:"name,type:VARCHAR(120),notnull" keel:"name"`
	CredentialID   string               `json:"credentialId"   bun:"credential_id,type:VARCHAR(512)" keel:"credential_id,nullzero"`
	SecretCipher   string               `json:"-"              bun:"secret_cipher,type:TEXT" keel:"secret_cipher,nullzero"`
	Enabled        bool                 `json:"enabled"        bun:"enabled,notnull" keel:"enabled"`
	VerifiedAt     int64                `json:"verifiedAt"     bun:"verified_at" keel:"verified_at,nullzero"`
	LastUsedAt     int64                `json:"lastUsedAt"     bun:"last_used_at" keel:"last_used_at,nullzero"`
	LastUsedStep   int64                `json:"-"              bun:"last_used_step,notnull,default:0" keel:"last_used_step"`
	CreatedAt      int64                `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt      int64                `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

func (a *MFAAuthenticator) BeforeAppendModel(_ context.Context, q bun.Query) error {
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &a.ID,
		IDPrefix:  "mfa_",
		CreatedAt: &a.CreatedAt,
		UpdatedAt: &a.UpdatedAt,
	})
	return nil
}

type AuthEvent struct {
	bun.BaseModel `bun:"table:auth_events,alias:ae" json:"-"`
	keel.Table    `keel:"auth_events"`

	pagination.CursorValueSet `bun:",embed" json:"-"`

	ID                 pulid.ID         `json:"id"                 bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	UserID             pulid.ID         `json:"userId"             bun:"user_id,type:VARCHAR(100)" keel:"user_id,nullzero"`
	OrganizationID     pulid.ID         `json:"organizationId"     bun:"organization_id,type:VARCHAR(100)" keel:"organization_id,nullzero"`
	BusinessUnitID     pulid.ID         `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100)" keel:"business_unit_id,nullzero"`
	IdentityProviderID pulid.ID         `json:"identityProviderId" bun:"identity_provider_id,type:VARCHAR(100)" keel:"identity_provider_id,nullzero"`
	Provider           string           `json:"provider"           bun:"provider,type:VARCHAR(120),notnull" keel:"provider"`
	Outcome            AuthEventOutcome `json:"outcome"            bun:"outcome,type:iam_auth_event_outcome_enum,notnull" keel:"outcome"`
	IPAddress          string           `json:"ipAddress"          bun:"ip_address,type:INET,nullzero" keel:"ip_address,nullzero"`
	UserAgent          string           `json:"userAgent"          bun:"user_agent,type:TEXT,nullzero" keel:"user_agent,nullzero"`
	AuthenticatorAAL   int              `json:"authenticatorAal"   bun:"authenticator_aal,notnull,default:1" keel:"authenticator_aal"`
	FederationFAL      int              `json:"federationFal"      bun:"federation_fal,notnull,default:1" keel:"federation_fal"`
	MFAState           string           `json:"mfaState"           bun:"mfa_state,type:VARCHAR(80)" keel:"mfa_state,nullzero"`
	RiskOutcome        RiskOutcome      `json:"riskOutcome"        bun:"risk_outcome,type:iam_risk_outcome_enum,notnull,default:'allow'" keel:"risk_outcome"`
	RiskSignals        []string         `json:"riskSignals"        bun:"risk_signals,array" keel:"risk_signals"`
	ErrorCode          string           `json:"errorCode"          bun:"error_code,type:VARCHAR(120)" keel:"error_code,nullzero"`
	OccurredAt         int64            `json:"occurredAt"         bun:"occurred_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"occurred_at"`
	CreatedAt          int64            `json:"createdAt"          bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
}

func (e *AuthEvent) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if e.RiskSignals == nil {
		e.RiskSignals = []string{}
	}
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &e.ID,
		IDPrefix:  "aue_",
		CreatedAt: &e.CreatedAt,
	})
	if e.OccurredAt == 0 {
		e.OccurredAt = e.CreatedAt
	}
	return nil
}

type RiskDecision struct {
	bun.BaseModel `bun:"table:risk_decisions,alias:rd" json:"-"`
	keel.Table    `keel:"risk_decisions"`

	pagination.CursorValueSet `bun:",embed" json:"-"`

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	UserID         pulid.ID    `json:"userId"         bun:"user_id,type:VARCHAR(100)" keel:"user_id,nullzero"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,type:VARCHAR(100)" keel:"organization_id,nullzero"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100)" keel:"business_unit_id,nullzero"`
	Outcome        RiskOutcome `json:"outcome"        bun:"outcome,type:iam_risk_outcome_enum,notnull" keel:"outcome"`
	Signals        []string    `json:"signals"        bun:"signals,array" keel:"signals"`
	Reason         string      `json:"reason"         bun:"reason,type:TEXT" keel:"reason,nullzero"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
}

func (d *RiskDecision) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if d.Signals == nil {
		d.Signals = []string{}
	}
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &d.ID,
		IDPrefix:  "rsk_",
		CreatedAt: &d.CreatedAt,
	})
	return nil
}

type SCIMDirectory struct {
	bun.BaseModel `bun:"table:scim_directories,alias:sd" json:"-"`
	keel.Table    `keel:"scim_directories"`

	pagination.CursorValueSet `bun:",embed" json:"-"`

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	TenantSlug     string   `json:"tenantSlug"     bun:"tenant_slug,type:VARCHAR(100),notnull" keel:"tenant_slug"`
	Enabled        bool     `json:"enabled"        bun:"enabled,notnull" keel:"enabled"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

func (d *SCIMDirectory) BeforeAppendModel(_ context.Context, q bun.Query) error {
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &d.ID,
		IDPrefix:  "scd_",
		CreatedAt: &d.CreatedAt,
		UpdatedAt: &d.UpdatedAt,
	})
	return nil
}

type SCIMToken struct {
	bun.BaseModel `bun:"table:scim_tokens,alias:st" json:"-"`
	keel.Table    `keel:"scim_tokens"`

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	DirectoryID    pulid.ID        `json:"directoryId"    bun:"directory_id,type:VARCHAR(100),notnull" keel:"directory_id"`
	Name           string          `json:"name"           bun:"name,type:VARCHAR(120),notnull" keel:"name"`
	Prefix         string          `json:"prefix"         bun:"prefix,type:VARCHAR(24),notnull" keel:"prefix"`
	TokenHash      string          `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull" keel:"token_hash"`
	Status         SCIMTokenStatus `json:"status"         bun:"status,type:iam_scim_token_status_enum,notnull" keel:"status"`
	LastUsedAt     int64           `json:"lastUsedAt"     bun:"last_used_at" keel:"last_used_at,nullzero"`
	ExpiresAt      int64           `json:"expiresAt"      bun:"expires_at" keel:"expires_at,nullzero"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
}

func (t *SCIMToken) BeforeAppendModel(_ context.Context, q bun.Query) error {
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &t.ID,
		IDPrefix:  "sct_",
		CreatedAt: &t.CreatedAt,
		UpdatedAt: &t.UpdatedAt,
	})
	return nil
}

type ProvisioningAuditRecord struct {
	bun.BaseModel `bun:"table:provisioning_audit_records,alias:par" json:"-"`
	keel.Table    `keel:"provisioning_audit_records"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	DirectoryID    pulid.ID           `json:"directoryId"    bun:"directory_id,type:VARCHAR(100),notnull" keel:"directory_id"`
	Action         ProvisioningAction `json:"action"         bun:"action,type:iam_provisioning_action_enum,notnull" keel:"action"`
	ResourceType   string             `json:"resourceType"   bun:"resource_type,type:VARCHAR(80),notnull" keel:"resource_type"`
	ExternalID     string             `json:"externalId"     bun:"external_id,type:VARCHAR(255)" keel:"external_id,nullzero"`
	ResourceID     pulid.ID           `json:"resourceId"     bun:"resource_id,type:VARCHAR(100)" keel:"resource_id,nullzero"`
	Status         string             `json:"status"         bun:"status,type:VARCHAR(80),notnull" keel:"status"`
	ErrorMessage   string             `json:"errorMessage"   bun:"error_message,type:TEXT" keel:"error_message,nullzero"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
}

func (r *ProvisioningAuditRecord) BeforeAppendModel(_ context.Context, q bun.Query) error {
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &r.ID,
		IDPrefix:  "par_",
		CreatedAt: &r.CreatedAt,
	})
	return nil
}

type AccessPolicy struct {
	bun.BaseModel `bun:"table:access_policies,alias:ap" json:"-"`
	keel.Table    `keel:"access_policies"`

	ID             pulid.ID          `json:"id"             bun:"id,pk,type:VARCHAR(100)" keel:"id,pk"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull" keel:"organization_id,tenant"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull" keel:"business_unit_id,tenant"`
	Name           string            `json:"name"           bun:"name,type:VARCHAR(120),notnull" keel:"name"`
	Resource       string            `json:"resource"       bun:"resource,type:VARCHAR(120),notnull" keel:"resource"`
	Operation      string            `json:"operation"      bun:"operation,type:VARCHAR(80),notnull" keel:"operation"`
	Effect         PolicyEffect      `json:"effect"         bun:"effect,type:iam_policy_effect_enum,notnull" keel:"effect"`
	Priority       int               `json:"priority"       bun:"priority,notnull" keel:"priority"`
	Conditions     map[string]string `json:"conditions"     bun:"conditions,type:JSONB,notnull" keel:"conditions,jsonb"`
	Enabled        bool              `json:"enabled"        bun:"enabled,notnull" keel:"enabled"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"created_at"`
	UpdatedAt      int64             `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint" keel:"updated_at"`
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

func (p *AccessPolicy) BeforeAppendModel(_ context.Context, q bun.Query) error {
	if p.Conditions == nil {
		p.Conditions = map[string]string{}
	}
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &p.ID,
		IDPrefix:  "pol_",
		CreatedAt: &p.CreatedAt,
		UpdatedAt: &p.UpdatedAt,
	})
	return nil
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

type iamTimestampParams struct {
	Query     bun.Query
	ID        *pulid.ID
	IDPrefix  string
	CreatedAt *int64
	UpdatedAt *int64
}

func setIAMTimestamps(p iamTimestampParams) {
	now := timeutils.NowUnix()
	switch p.Query.(type) {
	case *bun.InsertQuery:
		if p.ID.IsNil() {
			*p.ID = pulid.MustNew(p.IDPrefix)
		}
		*p.CreatedAt = now
	case *bun.UpdateQuery:
		if p.UpdatedAt != nil {
			*p.UpdatedAt = now
		}
	}
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
