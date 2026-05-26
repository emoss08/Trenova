package iam

import (
	"context"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	validation "github.com/go-ozzo/ozzo-validation/v4"
	"github.com/uptrace/bun"
)

var (
	_ bun.BeforeAppendModelHook = (*IdentityProvider)(nil)
	_ bun.BeforeAppendModelHook = (*ExternalIdentity)(nil)
	_ bun.BeforeAppendModelHook = (*MFAAuthenticator)(nil)
	_ bun.BeforeAppendModelHook = (*AuthEvent)(nil)
	_ bun.BeforeAppendModelHook = (*RiskDecision)(nil)
	_ bun.BeforeAppendModelHook = (*SCIMDirectory)(nil)
	_ bun.BeforeAppendModelHook = (*SCIMToken)(nil)
	_ bun.BeforeAppendModelHook = (*SCIMGroupRoleMapping)(nil)
	_ bun.BeforeAppendModelHook = (*ProvisioningAuditRecord)(nil)
	_ bun.BeforeAppendModelHook = (*AccessPolicy)(nil)
)

type IdentityProvider struct {
	bun.BaseModel `bun:"table:identity_providers,alias:idp" json:"-"`

	ID                  pulid.ID                 `json:"id"                bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID      pulid.ID                 `json:"organizationId"    bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID      pulid.ID                 `json:"businessUnitId"    bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Name                string                   `json:"name"              bun:"name,type:VARCHAR(120),notnull"`
	Slug                string                   `json:"slug"              bun:"slug,type:VARCHAR(80),notnull"`
	Protocol            IdentityProviderProtocol `json:"protocol"          bun:"protocol,type:iam_identity_provider_protocol_enum,notnull"`
	Enabled             bool                     `json:"enabled"           bun:"enabled,notnull"`
	EnforceSSO          bool                     `json:"enforceSso"        bun:"enforce_sso,notnull"`
	AutoProvision       bool                     `json:"autoProvision"     bun:"auto_provision,notnull"`
	AllowFederatedMFA   bool                     `json:"allowFederatedMfa" bun:"allow_federated_mfa,notnull"`
	AllowedDomains      []string                 `json:"allowedDomains"    bun:"allowed_domains,array"`
	AttributeMap        map[string]string        `json:"attributeMap"      bun:"attribute_map,type:JSONB,notnull"`
	OIDCIssuerURL       string                   `json:"oidcIssuerUrl"     bun:"oidc_issuer_url,type:VARCHAR(500)"`
	OIDCClientID        string                   `json:"oidcClientId"      bun:"oidc_client_id,type:VARCHAR(255)"`
	OIDCClientSecret    string                   `json:"-"                 bun:"oidc_client_secret,type:TEXT"`
	OIDCRedirectURL     string                   `json:"oidcRedirectUrl"   bun:"oidc_redirect_url,type:VARCHAR(500)"`
	OIDCScopes          []string                 `json:"oidcScopes"        bun:"oidc_scopes,array"`
	SAMLEntityID        string                   `json:"samlEntityId"      bun:"saml_entity_id,type:VARCHAR(500)"`
	SAMLSSOURL          string                   `json:"samlSsoUrl"        bun:"saml_sso_url,type:VARCHAR(500)"`
	SAMLX509Certificate string                   `json:"-"                 bun:"saml_x509_certificate,type:TEXT"`
	SAMLMetadataXML     string                   `json:"-"                 bun:"saml_metadata_xml,type:TEXT"`
	Version             int64                    `json:"version"           bun:"version,type:BIGINT,notnull,default:0"`
	CreatedAt           int64                    `json:"createdAt"         bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt           int64                    `json:"updatedAt"         bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
}

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

	ID                 pulid.ID          `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	UserID             pulid.ID          `json:"userId"             bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID     pulid.ID          `json:"organizationId"     bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID     pulid.ID          `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100),notnull"`
	IdentityProviderID pulid.ID          `json:"identityProviderId" bun:"identity_provider_id,type:VARCHAR(100),notnull"`
	ExternalSubject    string            `json:"externalSubject"    bun:"external_subject,type:VARCHAR(255),notnull"`
	ExternalUsername   string            `json:"externalUsername"   bun:"external_username,type:VARCHAR(255)"`
	ExternalEmail      string            `json:"externalEmail"      bun:"external_email,type:VARCHAR(255)"`
	RawClaims          map[string]string `json:"rawClaims"          bun:"raw_claims,type:JSONB,notnull"`
	LastLoginAt        int64             `json:"lastLoginAt"        bun:"last_login_at"`
	CreatedAt          int64             `json:"createdAt"          bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt          int64             `json:"updatedAt"          bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID             pulid.ID             `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID             `json:"userId"         bun:"user_id,type:VARCHAR(100),notnull"`
	OrganizationID pulid.ID             `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	Type           MFAAuthenticatorType `json:"type"           bun:"type,type:iam_mfa_authenticator_type_enum,notnull"`
	Name           string               `json:"name"           bun:"name,type:VARCHAR(120),notnull"`
	CredentialID   string               `json:"credentialId"   bun:"credential_id,type:VARCHAR(512)"`
	SecretCipher   string               `json:"-"              bun:"secret_cipher,type:TEXT"`
	Enabled        bool                 `json:"enabled"        bun:"enabled,notnull"`
	VerifiedAt     int64                `json:"verifiedAt"     bun:"verified_at"`
	LastUsedAt     int64                `json:"lastUsedAt"     bun:"last_used_at"`
	CreatedAt      int64                `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64                `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID                 pulid.ID         `json:"id"                 bun:"id,pk,type:VARCHAR(100)"`
	UserID             pulid.ID         `json:"userId"             bun:"user_id,type:VARCHAR(100)"`
	OrganizationID     pulid.ID         `json:"organizationId"     bun:"organization_id,type:VARCHAR(100)"`
	BusinessUnitID     pulid.ID         `json:"businessUnitId"     bun:"business_unit_id,type:VARCHAR(100)"`
	IdentityProviderID pulid.ID         `json:"identityProviderId" bun:"identity_provider_id,type:VARCHAR(100)"`
	Provider           string           `json:"provider"           bun:"provider,type:VARCHAR(120),notnull"`
	Outcome            AuthEventOutcome `json:"outcome"            bun:"outcome,type:iam_auth_event_outcome_enum,notnull"`
	IPAddress          string           `json:"ipAddress"          bun:"ip_address,type:INET"`
	UserAgent          string           `json:"userAgent"          bun:"user_agent,type:TEXT"`
	AuthenticatorAAL   int              `json:"authenticatorAal"   bun:"authenticator_aal,notnull,default:1"`
	FederationFAL      int              `json:"federationFal"      bun:"federation_fal,notnull,default:1"`
	MFAState           string           `json:"mfaState"           bun:"mfa_state,type:VARCHAR(80)"`
	RiskOutcome        RiskOutcome      `json:"riskOutcome"        bun:"risk_outcome,type:iam_risk_outcome_enum,notnull,default:'allow'"`
	RiskSignals        []string         `json:"riskSignals"        bun:"risk_signals,array"`
	ErrorCode          string           `json:"errorCode"          bun:"error_code,type:VARCHAR(120)"`
	OccurredAt         int64            `json:"occurredAt"         bun:"occurred_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	CreatedAt          int64            `json:"createdAt"          bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID             pulid.ID    `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	UserID         pulid.ID    `json:"userId"         bun:"user_id,type:VARCHAR(100)"`
	OrganizationID pulid.ID    `json:"organizationId" bun:"organization_id,type:VARCHAR(100)"`
	BusinessUnitID pulid.ID    `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100)"`
	Outcome        RiskOutcome `json:"outcome"        bun:"outcome,type:iam_risk_outcome_enum,notnull"`
	Signals        []string    `json:"signals"        bun:"signals,array"`
	Reason         string      `json:"reason"         bun:"reason,type:TEXT"`
	CreatedAt      int64       `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID             pulid.ID `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	TenantSlug     string   `json:"tenantSlug"     bun:"tenant_slug,type:VARCHAR(100),notnull"`
	Enabled        bool     `json:"enabled"        bun:"enabled,notnull"`
	CreatedAt      int64    `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64    `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID             pulid.ID        `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID        `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	DirectoryID    pulid.ID        `json:"directoryId"    bun:"directory_id,type:VARCHAR(100),notnull"`
	Name           string          `json:"name"           bun:"name,type:VARCHAR(120),notnull"`
	Prefix         string          `json:"prefix"         bun:"prefix,type:VARCHAR(24),notnull"`
	TokenHash      string          `json:"-"              bun:"token_hash,type:VARCHAR(128),notnull"`
	Status         SCIMTokenStatus `json:"status"         bun:"status,type:iam_scim_token_status_enum,notnull"`
	LastUsedAt     int64           `json:"lastUsedAt"     bun:"last_used_at"`
	ExpiresAt      int64           `json:"expiresAt"      bun:"expires_at"`
	CreatedAt      int64           `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64           `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

type SCIMGroupRoleMapping struct {
	bun.BaseModel `bun:"table:scim_group_role_mappings,alias:sgrm" json:"-"`

	ID              pulid.ID `json:"id"              bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID  pulid.ID `json:"organizationId"  bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID  pulid.ID `json:"businessUnitId"  bun:"business_unit_id,type:VARCHAR(100),notnull"`
	DirectoryID     pulid.ID `json:"directoryId"     bun:"directory_id,type:VARCHAR(100),notnull"`
	ExternalGroupID string   `json:"externalGroupId" bun:"external_group_id,type:VARCHAR(255),notnull"`
	DisplayName     string   `json:"displayName"     bun:"display_name,type:VARCHAR(255),notnull"`
	RoleID          pulid.ID `json:"roleId"          bun:"role_id,type:VARCHAR(100),notnull"`
	CreatedAt       int64    `json:"createdAt"       bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt       int64    `json:"updatedAt"       bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
}

func (m *SCIMGroupRoleMapping) BeforeAppendModel(_ context.Context, q bun.Query) error {
	setIAMTimestamps(iamTimestampParams{
		Query:     q,
		ID:        &m.ID,
		IDPrefix:  "sgr_",
		CreatedAt: &m.CreatedAt,
		UpdatedAt: &m.UpdatedAt,
	})
	return nil
}

type ProvisioningAuditRecord struct {
	bun.BaseModel `bun:"table:provisioning_audit_records,alias:par" json:"-"`

	ID             pulid.ID           `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID           `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	DirectoryID    pulid.ID           `json:"directoryId"    bun:"directory_id,type:VARCHAR(100),notnull"`
	Action         ProvisioningAction `json:"action"         bun:"action,type:iam_provisioning_action_enum,notnull"`
	ResourceType   string             `json:"resourceType"   bun:"resource_type,type:VARCHAR(80),notnull"`
	ExternalID     string             `json:"externalId"     bun:"external_id,type:VARCHAR(255)"`
	ResourceID     pulid.ID           `json:"resourceId"     bun:"resource_id,type:VARCHAR(100)"`
	Status         string             `json:"status"         bun:"status,type:VARCHAR(80),notnull"`
	ErrorMessage   string             `json:"errorMessage"   bun:"error_message,type:TEXT"`
	CreatedAt      int64              `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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

	ID             pulid.ID          `json:"id"             bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID pulid.ID          `json:"organizationId" bun:"organization_id,type:VARCHAR(100),notnull"`
	BusinessUnitID pulid.ID          `json:"businessUnitId" bun:"business_unit_id,type:VARCHAR(100),notnull"`
	Name           string            `json:"name"           bun:"name,type:VARCHAR(120),notnull"`
	Resource       string            `json:"resource"       bun:"resource,type:VARCHAR(120),notnull"`
	Operation      string            `json:"operation"      bun:"operation,type:VARCHAR(80),notnull"`
	Effect         PolicyEffect      `json:"effect"         bun:"effect,type:iam_policy_effect_enum,notnull"`
	Priority       int               `json:"priority"       bun:"priority,notnull,default:0"`
	Conditions     map[string]string `json:"conditions"     bun:"conditions,type:JSONB,notnull"`
	Enabled        bool              `json:"enabled"        bun:"enabled,notnull"`
	CreatedAt      int64             `json:"createdAt"      bun:"created_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
	UpdatedAt      int64             `json:"updatedAt"      bun:"updated_at,notnull,default:extract(epoch from current_timestamp)::bigint"`
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
