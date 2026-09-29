package tenanttimezone

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type organizationReader interface {
	GetByID(
		ctx context.Context,
		req repositories.GetOrganizationByIDRequest,
	) (*tenant.Organization, error)
}

type userReader interface {
	GetByID(ctx context.Context, req repositories.GetUserByIDRequest) (*tenant.User, error)
}

type Params struct {
	fx.In

	Organizations repositories.OrganizationRepository
	Users         repositories.UserRepository
}

type Reader struct {
	organizations organizationReader
	users         userReader
}

func New(p Params) services.TenantTimezoneReader {
	return &Reader{organizations: p.Organizations, users: p.Users}
}

func (r *Reader) TenantTimezone(ctx context.Context, info pagination.TenantInfo) (string, error) {
	org, err := r.organizations.GetByID(ctx, repositories.GetOrganizationByIDRequest{
		TenantInfo: info,
	})
	if err != nil {
		return "", fmt.Errorf("read the organization's timezone: %w", err)
	}
	if timeutils.IsValidLocation(org.Timezone) {
		return org.Timezone, nil
	}
	if info.UserID.IsNil() {
		return "", nil
	}

	user, err := r.users.GetByID(ctx, repositories.GetUserByIDRequest{
		TenantInfo:   info,
		LookupUserID: info.UserID,
	})
	if err != nil {
		return "", fmt.Errorf("read the person's timezone: %w", err)
	}
	if timeutils.IsValidLocation(user.Timezone) {
		return user.Timezone, nil
	}

	return "", nil
}
