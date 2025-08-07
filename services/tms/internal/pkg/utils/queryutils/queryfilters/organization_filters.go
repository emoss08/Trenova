/*
 * Copyright 2023-2025 Eric Moss
 * Licensed under FSL-1.1-ALv2 (Functional Source License 1.1, Apache 2.0 Future)
 * Full license: https://github.com/emoss08/Trenova/blob/master/LICENSE.md */

package queryfilters

import (
	"fmt"

	"github.com/emoss08/trenova/internal/core/ports"
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
