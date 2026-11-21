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
}
