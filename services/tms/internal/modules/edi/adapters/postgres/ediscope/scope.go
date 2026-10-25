package ediscope

import (
	"context"

	"github.com/emoss08/keel"
	"github.com/emoss08/keel/schema"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

const (
	organizationIDColumn = "organization_id"
	businessUnitIDColumn = "business_unit_id"
)

func Tenant[T any](
	ctx context.Context,
	q *keel.Query[*T],
	tenant pagination.TenantInfo,
) *keel.Query[*T] {
	if !dbscope.IsSystem(ctx) {
		return q
	}

	ref := q.TableRef()
	if !ref.Table.HasTenant() {
		return q
	}

	q.AllTenants()
	for _, column := range []struct {
		name string
		id   pulid.ID
	}{
		{name: organizationIDColumn, id: tenant.OrgID},
		{name: businessUnitIDColumn, id: tenant.BuID},
	} {
		index := ref.Table.Column(column.name)
		if index < 0 || !ref.Table.Columns[index].Flags.Has(schema.FlagTenant) {
			continue
		}
		if column.id.IsNil() {
			return q.Where(keel.Or())
		}
		q.Where(keel.StringCol{Col: keel.Col{Ref: ref, Index: index}}.Eq(column.id.String()))
	}

	return q
}
