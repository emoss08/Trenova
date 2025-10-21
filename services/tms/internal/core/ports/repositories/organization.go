package repositories

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/pkg/pulid"
)

type GetOrganizationByIDRequest struct {
	OrgID        pulid.ID
	BuID         pulid.ID
	IncludeState bool
	IncludeBu    bool
}

type OrganizationRepository interface {
	// List(
	// 	ctx context.Context,
	// 	opts *pagination.QueryOptions,
	// ) (*pagination.ListResult[*tenant.Organization], error)
	GetByID(ctx context.Context, opts GetOrganizationByIDRequest) (*tenant.Organization, error)
	GetUserOrganizations(
		ctx context.Context,
		opts *pagination.QueryOptions,
	) (*pagination.ListResult[*tenant.Organization], error)
	// Create(ctx context.Context, org *tenant.Organization) (*tenant.Organization, error)
	Update(ctx context.Context, org *tenant.Organization) (*tenant.Organization, error)
	// SetLogo(ctx context.Context, org *tenant.Organization) (*tenant.Organization, error)
	// ClearLogo(
	// 	ctx context.Context,
	// 	org *tenant.Organization,
	// ) (*tenant.Organization, error)

	// GetOrganizationBucketName(ctx context.Context, orgID pulid.ID) (string, error)
}

type OrganizationCacheRepository interface {
	GetByID(ctx context.Context, orgID pulid.ID) (*tenant.Organization, error)
	GetUserOrganizations(ctx context.Context, userID pulid.ID) ([]*tenant.Organization, error)
	SetUserOrganizations(
		ctx context.Context,
		userID pulid.ID,
		orgs []*tenant.Organization,
	) error
	Set(ctx context.Context, org *tenant.Organization) error
	Invalidate(ctx context.Context, orgID pulid.ID) error
	InvalidateUserOrganizations(ctx context.Context, userID pulid.ID) error
	InvalidateOrganizationForAllUsers(ctx context.Context, orgID pulid.ID) error
}
