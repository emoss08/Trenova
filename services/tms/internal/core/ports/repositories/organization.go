package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type GetOrganizationByIDRequest struct {
	TenantInfo   pagination.TenantInfo
	IncludeState bool `json:"includeState"`
	IncludeBU    bool `json:"includeBu"`
}

type OrganizationRepository interface {
	GetByID(ctx context.Context, req GetOrganizationByIDRequest) (*tenant.Organization, error)
	Update(ctx context.Context, entity *tenant.Organization) (*tenant.Organization, error)
	ClearLogoURL(ctx context.Context, orgID pulid.ID, version int64) (*tenant.Organization, error)
}

type OrganizationCacheRepository interface {
	GetByID(ctx context.Context, orgID pulid.ID) (*tenant.Organization, error)
}
