package services

import (
	"context"
	"mime/multipart"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type UploadLogoRequest struct {
	TenantInfo     pagination.TenantInfo
	OrganizationID pulid.ID
	File           *multipart.FileHeader
}

type GetLogoURLRequest struct {
	TenantInfo     pagination.TenantInfo
	OrganizationID pulid.ID
}

type GetLogoURLResponse struct {
	URL string `json:"url,omitempty"`
}

type DeleteLogoRequest struct {
	TenantInfo     pagination.TenantInfo
	OrganizationID pulid.ID
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

type OrganizationService interface {
	GetByID(
		ctx context.Context,
		req repositories.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
	Update(ctx context.Context, entity *tenant.Organization) (*tenant.Organization, error)
	UploadLogo(
		ctx context.Context,
		req *UploadLogoRequest,
		userID pulid.ID,
	) (*tenant.Organization, error)
	GetLogoURL(ctx context.Context, req GetLogoURLRequest) (*GetLogoURLResponse, error)
	DeleteLogo(ctx context.Context, req DeleteLogoRequest) (*tenant.Organization, error)
	GetMicrosoftSSOConfig(
		ctx context.Context,
		organizationID pulid.ID,
	) (*MicrosoftSSOConfig, error)
	UpsertMicrosoftSSOConfig(
		ctx context.Context,
		tenantInfo pagination.TenantInfo,
		config *MicrosoftSSOConfig,
	) (*MicrosoftSSOConfig, error)
}
