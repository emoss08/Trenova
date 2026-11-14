package tenantports

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type UploadBrandImageRequest struct {
	TenantInfo     pagination.TenantInfo
	OrganizationID pulid.ID
	Image          tenant.BrandImage
	File           *multipart.FileHeader
}

type GetBrandingRequest struct {
	TenantInfo     pagination.TenantInfo
	OrganizationID pulid.ID
}

type Branding struct {
	LogoURL string `json:"logoUrl"`
	IconURL string `json:"iconUrl"`
}

type DeleteBrandImageRequest struct {
	TenantInfo     pagination.TenantInfo
	OrganizationID pulid.ID
	Image          tenant.BrandImage
}

type GetOrganizationsByIDsRequest struct {
	TenantInfo      pagination.TenantInfo
	OrganizationIDs []pulid.ID
	IncludeState    bool
	IncludeBU       bool
}

type MicrosoftSSOConfig struct {
	OrganizationID   string   `json:"organizationId"`
	Enabled          bool     `json:"enabled"`
	EnforceSSO       bool     `json:"enforceSso"`
	TenantID         string   `json:"tenantId"`
	ClientID         string   `json:"clientId"`
	ClientSecret     string   `json:"clientSecret,omitempty"`
	RedirectURL      string   `json:"redirectUrl"`
	AllowedDomains   []string `json:"allowedDomains"`
	SecretConfigured bool     `json:"secretConfigured"`
}

type OktaSSOConfig struct {
	OrganizationID   string   `json:"organizationId"`
	Enabled          bool     `json:"enabled"`
	EnforceSSO       bool     `json:"enforceSso"`
	IssuerURL        string   `json:"issuerUrl"`
	ClientID         string   `json:"clientId"`
	ClientSecret     string   `json:"clientSecret,omitempty"`
	RedirectURL      string   `json:"redirectUrl"`
	Scopes           []string `json:"scopes"`
	AllowedDomains   []string `json:"allowedDomains"`
	SecretConfigured bool     `json:"secretConfigured"`
}

type OrganizationService interface {
	GetByID(
		ctx context.Context,
		req GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
	GetByIDs(
		ctx context.Context,
		req GetOrganizationsByIDsRequest,
	) ([]*tenant.Organization, error)
	SelectOptions(
		ctx context.Context,
		req *SelectOrganizationOptionsRequest,
	) (*pagination.ListResult[*tenant.Organization], error)
	Update(ctx context.Context, entity *tenant.Organization) (*tenant.Organization, error)
	UploadBrandImage(
		ctx context.Context,
		req *UploadBrandImageRequest,
		userID pulid.ID,
	) (*tenant.Organization, error)
	GetBranding(ctx context.Context, req GetBrandingRequest) (*Branding, error)
	DeleteBrandImage(ctx context.Context, req DeleteBrandImageRequest) (*tenant.Organization, error)
	GetMicrosoftSSOConfig(
		ctx context.Context,
		organizationID pulid.ID,
	) (*MicrosoftSSOConfig, error)
	UpsertMicrosoftSSOConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		config *MicrosoftSSOConfig,
	) (*MicrosoftSSOConfig, error)
	GetOktaSSOConfig(
		ctx context.Context,
		organizationID pulid.ID,
	) (*OktaSSOConfig, error)
	UpsertOktaSSOConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		config *OktaSSOConfig,
	) (*OktaSSOConfig, error)
}
