package tenantlifecycle

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/cloud/domain/cloudtenant"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
)

type SupportTenants struct {
	tenants cloudtenantports.Repository
}

var _ supportaccessports.Tenants = (*SupportTenants)(nil)

func NewSupportTenants(tenants cloudtenantports.Repository) *SupportTenants {
	return &SupportTenants{tenants: tenants}
}

func (s *SupportTenants) Tenant(
	ctx context.Context,
	instanceID string,
) (*supportaccessports.SupportTenant, error) {
	if !cloudtenant.ValidInstanceID(instanceID) {
		return nil, supportaccessports.ErrTenantNotHeld
	}
	t, err := s.tenants.Get(ctx, instanceID)
	if errors.Is(err, cloudtenantports.ErrNotFound) {
		return nil, supportaccessports.ErrTenantNotHeld
	}
	if err != nil {
		return nil, err
	}
	if !t.Made() {
		return nil, supportaccessports.ErrTenantNotHeld
	}
	return &supportaccessports.SupportTenant{
		OrganizationID: t.OrganizationID,
		BusinessUnitID: t.BusinessUnitID,
		Active:         t.State == cloudtenant.StateActive,
	}, nil
}
