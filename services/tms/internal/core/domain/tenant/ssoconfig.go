package tenant

import (
	"context"

	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/uptrace/bun"
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

var _ bun.BeforeAppendModelHook = (*SSOConfig)(nil)

type SSOConfig struct {
	bun.BaseModel `bun:"table:sso_configs,alias:ssoc" json:"-"`

	ID               pulid.ID          `json:"id"               bun:"id,pk,type:VARCHAR(100)"`
	OrganizationID   pulid.ID          `json:"organizationId"   bun:"organization_id,pk,type:VARCHAR(100),notnull"`
	BusinessUnitID   pulid.ID          `json:"businessUnitId"   bun:"business_unit_id,pk,type:VARCHAR(100),notnull"`
	Name             string            `json:"name"             bun:"name,type:VARCHAR(100),notnull"`
	Provider         SSOProvider       `json:"provider"         bun:"provider,type:sso_provider_enum,notnull"`
	Protocol         SSOProtocol       `json:"protocol"         bun:"protocol,type:sso_protocol_enum,notnull"`
	Enabled          bool              `json:"enabled"          bun:"enabled,notnull"`
	EnforceSSO       bool              `json:"enforceSso"       bun:"enforce_sso,notnull"`
	AutoProvision    bool              `json:"autoProvision"    bun:"auto_provision,notnull"`
	DefaultRole      string            `json:"defaultRole"      bun:"default_role,type:VARCHAR(50)"`
	AllowedDomains   []string          `json:"allowedDomains"   bun:"allowed_domains,array"`
	AttributeMap     map[string]string `json:"attributeMap"     bun:"attribute_map,type:JSONB"`
	OIDCIssuerURL    string            `json:"oidcIssuerUrl"    bun:"oidc_issuer_url,type:VARCHAR(500),notnull"`
	OIDCClientID     string            `json:"oidcClientId"     bun:"oidc_client_id,type:VARCHAR(255),notnull"`
	OIDCClientSecret string            `json:"-"                bun:"oidc_client_secret,type:TEXT,notnull"`
	OIDCRedirectURL  string            `json:"oidcRedirectUrl"  bun:"oidc_redirect_url,type:VARCHAR(500),notnull"`
	OIDCScopes       []string          `json:"oidcScopes"       bun:"oidc_scopes,array,notnull"`
	Version          int64             `json:"version"          bun:"version,type:BIGINT,notnull"`
	CreatedAt        int64             `json:"createdAt"        bun:"created_at,notnull"`
	UpdatedAt        int64             `json:"updatedAt"        bun:"updated_at,notnull"`
}

func (c *SSOConfig) BeforeAppendModel(_ context.Context, q bun.Query) error {
	now := timeutils.NowUnix()

	switch q.(type) {
	case *bun.InsertQuery:
		if c.ID.IsNil() {
			c.ID = pulid.MustNew("sso_")
		}
		c.CreatedAt = now
	case *bun.UpdateQuery:
		c.UpdatedAt = now
	}

	return nil
}
