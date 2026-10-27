package kernelports

import (
	"context"

	"github.com/emoss08/trenova/pkg/pagination"
)

type TenantLister interface {
	ListTenants(ctx context.Context) ([]pagination.TenantInfo, error)
}
