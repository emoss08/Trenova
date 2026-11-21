//go:build integration

package cloud_test

import (
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/tenantkeys"
	"github.com/emoss08/trenova/internal/modules/kernel/domain/platformcatalog"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type inboundKey struct {
	Constraint    string   `db:"constraint_name"`
	Child         string   `db:"child_table"`
	Parent        string   `db:"parent_table"`
	ChildColumns  []string `db:"child_columns"`
	ParentColumns []string `db:"parent_columns"`
	ChildHasBU    bool     `db:"child_has_bu"`
	ParentHasOrg  bool     `db:"parent_has_org"`
	ParentHasBU   bool     `db:"parent_has_bu"`
}

const inboundKeysQuery = `
WITH tenant_tables AS (
    SELECT c.oid, c.relname,
           bool_or(a.attname = 'organization_id') AS has_org,
           bool_or(a.attname = 'business_unit_id') AS has_bu
    FROM pg_catalog.pg_class c
    JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
    JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
    WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
      AND a.attname IN ('organization_id', 'business_unit_id')
    GROUP BY c.oid, c.relname
)
SELECT con.conname AS constraint_name,
       child.relname AS child_table,
       parent.relname AS parent_table,
       ARRAY(SELECT att.attname::text FROM unnest(con.conkey) WITH ORDINALITY k(n, o)
             JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.n
             ORDER BY k.o) AS child_columns,
       ARRAY(SELECT att.attname::text FROM unnest(con.confkey) WITH ORDINALITY k(n, o)
             JOIN pg_catalog.pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = k.n
             ORDER BY k.o) AS parent_columns,
       child.has_bu AS child_has_bu,
       COALESCE(parent.has_org, false) AS parent_has_org,
       COALESCE(parent.has_bu, false) AS parent_has_bu
FROM pg_catalog.pg_constraint con
JOIN tenant_tables child ON child.oid = con.conrelid
LEFT JOIN tenant_tables parent ON parent.oid = con.confrelid
WHERE con.contype = 'f'
  AND (parent.oid IS NOT NULL OR con.confrelid::regclass::text IN ('organizations', 'business_units'))
  AND NOT EXISTS (
      SELECT 1 FROM unnest(con.conkey) k(n)
      JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.n
      WHERE att.attname IN ('organization_id', 'business_unit_id') OR att.attname LIKE '%\_business\_unit\_id'
  )
ORDER BY con.conname`

func ident(name string) string { return `"` + strings.ReplaceAll(name, `"`, `""`) + `"` }

func identList(names []string, alias string) string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, alias+"."+ident(n))
	}
	return strings.Join(out, ", ")
}

func (k *inboundKey) parentOf(removed dbscope.Tenant) (string, []any) {
	switch {
	case k.Parent == "organizations":
		return "p.id = $1", []any{removed.OrganizationID.String()}
	case k.Parent == "business_units":
		return "p.id = $1", []any{removed.BusinessUnitID.String()}
	case k.ParentHasOrg:
		return "p.organization_id = $1", []any{removed.OrganizationID.String()}
	default:
		return "p.business_unit_id = $1", []any{removed.BusinessUnitID.String()}
	}
}

func (k *inboundKey) childOf(other dbscope.Tenant) string {
	if k.ChildHasBU {
		return "c.business_unit_id = " + literalOf(other.BusinessUnitID.String())
	}
	return "c.organization_id IN (SELECT id FROM organizations WHERE business_unit_id = " +
		literalOf(other.BusinessUnitID.String()) + ")"
}

func literalOf(v string) string { return "'" + strings.ReplaceAll(v, "'", "''") + "'" }

func (r *rig) plantInbound(t *testing.T, removed, other dbscope.Tenant) []string {
	t.Helper()
	keys, err := keel.QueryAll(
		r.ctx,
		r.admin,
		pgx.RowToAddrOfStructByName[inboundKey],
		inboundKeysQuery,
	)
	require.NoError(t, err)
	planted := make([]string, 0)
	for _, k := range keys {
		parentWhere, args := k.parentOf(removed)
		sql := fmt.Sprintf(
			`UPDATE %s c SET (%s) = (SELECT %s FROM %s p WHERE %s LIMIT 1)
             WHERE c.ctid = (SELECT c.ctid FROM %s c WHERE %s LIMIT 1)
               AND EXISTS (SELECT 1 FROM %s p WHERE %s)`,
			ident(k.Child), strings.Join(quoted(k.ChildColumns), ", "),
			identList(k.ParentColumns, "p"), ident(k.Parent), parentWhere,
			ident(k.Child), k.childOf(other),
			ident(k.Parent), parentWhere,
		)
		if len(k.ChildColumns) == 1 {
			sql = strings.Replace(sql, "SET ("+quoted(k.ChildColumns)[0]+") = (", "SET "+
				quoted(k.ChildColumns)[0]+" = (", 1)
		}
		n, execErr := keel.Exec(r.ctx, r.admin, sql, args...)
		if execErr == nil && n > 0 {
			planted = append(planted, k.Constraint)
		}
	}
	return planted
}

func quoted(names []string) []string {
	out := make([]string, 0, len(names))
	for _, n := range names {
		out = append(out, ident(n))
	}
	return out
}

func (r *rig) fingerprint(t *testing.T, tenant dbscope.Tenant) map[string]string {
	t.Helper()
	tables, err := keel.QueryAll(r.ctx, r.admin, pgx.RowToStructByName[struct {
		Name  string `db:"relname"`
		HasBU bool   `db:"has_bu"`
	}], `
SELECT c.relname, bool_or(a.attname = 'business_unit_id') AS has_bu
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
  AND a.attname IN ('organization_id', 'business_unit_id')
GROUP BY c.relname ORDER BY 1`)
	require.NoError(t, err)
	out := make(map[string]string, len(tables))
	for _, table := range tables {
		where := "t.organization_id IN (SELECT id FROM organizations WHERE business_unit_id = $1)"
		if table.HasBU {
			where = "t.business_unit_id = $1"
		}
		sum, qerr := keel.QueryValue[string](
			r.ctx,
			r.admin,
			"SELECT count(*)::text || ':' || COALESCE(md5(string_agg(t::text, '|' ORDER BY t::text)), '') FROM "+
				ident(
					table.Name,
				)+" t WHERE "+where,
			tenant.BusinessUnitID.String(),
		)
		require.NoError(t, qerr, table.Name)
		out[table.Name] = sum
	}
	return out
}

func TestRemovalStopsOnAnotherTenantsReferenceAndChangesNothing(t *testing.T) {
	r := newRigOn(t, true)
	other := r.organizationNamed(t, "Trenova Logistics")
	r.apply(t, entry(instanceA, "a@a.test", 1, platformcatalog.FeatureAccounting))
	removed := scopeOf(r.tenant(t, instanceA))

	planted := r.plantInbound(t, removed, other)
	require.NotEmpty(t, planted, "the declared exceptions admit at least one reference to plant")
	t.Logf("planted %d cross-tenant references: %s", len(planted), strings.Join(planted, ", "))
	declared := tenantkeys.Exceptions()
	for _, constraint := range planted {
		assert.Contains(t, declared, constraint, "only a declared exception admits another "+
			"tenant's record; every other key carries the tenant")
	}

	otherBefore := r.fingerprint(t, other)
	removedBefore := r.fingerprint(t, removed)

	_, err := r.purge.PurgeRows(r.ctx, &cloudtenantports.PurgeRowsRequest{Tenant: removed})
	var cross *cloudtenantports.CrossTenantReferencesError
	require.True(t, errors.As(err, &cross), "the purge refuses: %v", err)
	found := make([]string, 0, len(cross.References))
	for _, ref := range cross.References {
		assert.Positive(t, ref.Rows)
		assert.NotEmpty(t, ref.Table)
		assert.NotEmpty(t, ref.Columns)
		found = append(found, ref.Constraint)
	}
	for _, constraint := range planted {
		assert.True(t, slices.Contains(found, constraint), "%s is reported", constraint)
	}
	assert.NotContains(t, err.Error(), other.OrganizationID.String(), "the reason names no tenant")

	_, err = r.purge.DeleteOrganization(r.ctx, removed)
	require.True(t, errors.As(err, &cross), "the organization's cascade is refused too: %v", err)
	_, err = r.purge.DeleteBusinessUnit(r.ctx, removed)
	require.True(t, errors.As(err, &cross), "the business unit's cascade is refused too: %v", err)

	assert.Equal(t, otherBefore, r.fingerprint(t, other), "nothing of the other tenant changed")
	assert.Equal(t, removedBefore, r.fingerprint(t, removed), "nothing was removed on a refusal")
}

func TestRemovalOfATenantNobodyRefersToProceeds(t *testing.T) {
	r := newRigOn(t, true)
	other := r.organizationNamed(t, "Trenova Logistics")
	r.apply(t, entry(instanceA, "a@a.test", 1, platformcatalog.FeatureAccounting))
	removed := scopeOf(r.tenant(t, instanceA))
	otherBefore := r.fingerprint(t, other)

	r.purgeAll(t, removed, true)

	assert.Equal(t, otherBefore, r.fingerprint(t, other))
	for table, n := range r.remainingOf(t, removed) {
		assert.Zerof(t, n, "%s still holds %d rows of the removed tenant", table, n)
	}
}

func (r *rig) remainingOf(t *testing.T, tenant dbscope.Tenant) map[string]int64 {
	t.Helper()
	counts, err := r.purge.Remaining(r.ctx, tenant)
	require.NoError(t, err)
	out := make(map[string]int64, len(counts))
	for _, c := range counts {
		out[c.Table] = c.Rows
	}
	return out
}
