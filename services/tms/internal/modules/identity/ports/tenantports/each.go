package tenantports

import (
	"context"
	"errors"

	"github.com/emoss08/trenova/internal/modules/kernel/domain/entitlement"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const eachTenantPageSize = 100

type TenantPager interface {
	ListTenantsPage(
		ctx context.Context,
		need entitlement.Need,
		after pulid.ID,
		limit int,
	) ([]pagination.TenantInfo, error)
}

func EachTenant(
	ctx context.Context,
	tenants TenantPager,
	need entitlement.Need,
	fn func(context.Context, pagination.TenantInfo) error,
) error {
	if err := need.Validate(); err != nil {
		return err
	}
	var errs []error
	after := pulid.Nil
	for {
		page, err := tenants.ListTenantsPage(ctx, need, after, eachTenantPageSize)
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
