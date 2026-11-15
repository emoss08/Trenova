package tenantports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const eachTenantPageSize = 100

type TenantPager interface {
	ListTenantsPage(ctx context.Context, after pulid.ID, limit int) ([]pagination.TenantInfo, error)
}

func EachTenant(
	ctx context.Context,
	tenants TenantPager,
	fn func(context.Context, pagination.TenantInfo) error,
) error {
	var errs []error
	after := pulid.Nil
	for {
		page, err := tenants.ListTenantsPage(ctx, after, eachTenantPageSize)
		if err != nil {
			return errors.Join(append(errs, err)...)
		}
		for _, tenant := range page {
			if err = fn(dbscope.WithTenant(ctx, tenant.DBTenant()), tenant); err != nil {
				errs = append(errs, err)
			}
		}
		if len(page) < eachTenantPageSize {
			return errors.Join(errs...)
		}
		after = page[len(page)-1].OrgID
	}
}
