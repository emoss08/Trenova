package supportaccessservice

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/identity/ports/supportaccessports"
)

type noTenants struct{}

func NewNoTenants() supportaccessports.Tenants {
	return noTenants{}
}

func (noTenants) Tenant(context.Context, string) (*supportaccessports.SupportTenant, error) {
	return nil, supportaccessports.ErrTenantNotHeld
}
