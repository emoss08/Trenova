package tenanttimezone

import (
	"context"
	"fmt"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/app/orgzone"
	"github.com/emoss08/trenova/internal/modules/identity/domain/tenant"
	"github.com/emoss08/trenova/internal/modules/identity/ports/tenantports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/timeutils"
	"go.uber.org/fx"
)

type userReader interface {
	GetByID(ctx context.Context, req tenantports.GetUserByIDRequest) (*tenant.User, error)
}

type Params struct {
	fx.In

	Organizations tenantports.OrganizationRepository
	Users         tenantports.UserRepository
}

type Reader struct {
	organizations orgzone.OrganizationReader
	users         userReader
}

func New(p Params) agentports.TenantTimezoneReader {
	return &Reader{organizations: p.Organizations, users: p.Users}
}

func (r *Reader) TenantTimezone(ctx context.Context, info pagination.TenantInfo) (string, error) {
	org, err := r.organizations.GetByID(ctx, tenantports.GetOrganizationByIDRequest{
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

	user, err := r.users.GetByID(ctx, tenantports.GetUserByIDRequest{
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
