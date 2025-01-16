package queryfilters

import (
	"fmt"

	"github.com/trenova-app/transport/internal/core/ports"
	"github.com/uptrace/bun"
)

type TenantFilterQueryOptions struct {
	Query      *bun.SelectQuery
	Filter     *ports.LimitOffsetQueryOptions
	TableAlias string
}

func TenantFilterQuery(opts *TenantFilterQueryOptions) *bun.SelectQuery {
	return opts.Query.
		Where(fmt.Sprintf("%s.business_unit_id = ?", opts.TableAlias), opts.Filter.TenantOpts.BuID).
		Where(fmt.Sprintf("%s.organization_id = ?", opts.TableAlias), opts.Filter.TenantOpts.OrgID)
}
