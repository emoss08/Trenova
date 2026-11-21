package tenantpurgerepository

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/cloud/ports/cloudtenantports"
	"github.com/emoss08/trenova/internal/infrastructure/postgres/dbtx"
	"github.com/emoss08/trenova/pkg/dbscope"
	"github.com/jackc/pgx/v5"
)

const (
	crossTenantReason = "find rows of other tenants that refer to a removed shared-server tenant's records, before any of them is deleted, nulled or cascaded"
	crossTenantBatch  = 40

	organizationsTable = "organizations"
	businessUnitsTable = "business_units"
	idColumn           = "id"
)

type inboundReference struct {
	fk     *foreignKey
	child  tenantTable
	parent string
}

func (t tenantTable) belongs(alias string) string {
	col := func(name string) string { return alias + "." + quoteIdent(name) }
	switch {
	case t.HasBU && t.HasOrg:
		return "(" + col(businessUnitColumn) + " = $2 OR (" + col(businessUnitColumn) +
			" IS NULL AND " + col(organizationColumn) + " = $1))"
	case t.HasBU:
		return col(businessUnitColumn) + " = $2"
	case t.HasOrg:
		return col(organizationColumn) + " IN (SELECT o." + quoteIdent(idColumn) +
			" FROM " + quoteIdent(organizationsTable) + " o WHERE o." +
			quoteIdent(businessUnitColumn) + " = $2)"
	case t.PartyBU != "":
		return col(t.PartyBU) + " = $2"
	default:
		return ""
	}
}

func tenantColumn(name string) bool {
	return name == organizationColumn || name == businessUnitColumn ||
		strings.HasSuffix(name, "_"+businessUnitColumn)
}

func (p *purgePlan) parentWhere(table string) (string, bool) {
	switch table {
	case organizationsTable:
		return "p." + quoteIdent(idColumn) + " = $1", true
	case businessUnitsTable:
		return "p." + quoteIdent(idColumn) + " = $2", true
	}
	target, ok := p.byTable[table]
	if !ok {
		return "", false
	}
	param := "$2"
	if target.Column == organizationColumn {
		param = "$1"
	}
	return "p." + quoteIdent(target.Column) + " = " + param, true
}

func (p *purgePlan) inboundReferences() []inboundReference {
	out := make([]inboundReference, 0)
	for _, fk := range p.foreignKeys {
		child, ok := p.tables[fk.ChildTable]
		if !ok || child.belongs("c") == "" {
			continue
		}
		if len(fk.ChildColumns) == 0 || len(fk.ChildColumns) != len(fk.ParentColumns) {
			continue
		}
		if slices.ContainsFunc(fk.ChildColumns, tenantColumn) {
			continue
		}
		if _, known := p.parentWhere(fk.ParentTable); !known {
			continue
		}
		out = append(out, inboundReference{fk: fk, child: child, parent: fk.ParentTable})
	}
	slices.SortFunc(out, func(a, b inboundReference) int {
		return strings.Compare(a.fk.Name, b.fk.Name)
	})
	return out
}

func (p *purgePlan) CrossTenantQueries() []string {
	refs := p.inboundReferences()
	out := make([]string, 0, len(refs)/crossTenantBatch+1)
	parts := make([]string, 0, crossTenantBatch)
	flush := func() {
		if len(parts) > 0 {
			parts = append(parts, "SELECT NULL::text, NULL::text, NULL::text, 0::bigint "+
				"WHERE $1::text IS NULL AND $2::text IS NULL")
			out = append(out, strings.Join(parts, "\nUNION ALL\n"))
			parts = parts[:0]
		}
	}
	for _, ref := range refs {
		where, _ := p.parentWhere(ref.parent)
		parts = append(parts, "SELECT "+literal(ref.fk.Name)+"::text AS constraint_name, "+
			literal(ref.fk.ChildTable)+"::text AS table_name, "+
			literal(strings.Join(ref.fk.ChildColumns, ", "))+"::text AS column_names, "+
			"count(*)::bigint AS row_count FROM "+quoteIdent(ref.parent)+" p JOIN "+
			quoteIdent(ref.fk.ChildTable)+" c ON "+fkMatch(ref.fk)+" WHERE "+where+
			" AND NOT ("+ref.child.belongs("c")+")")
		if len(parts) >= crossTenantBatch {
			flush()
		}
	}
	flush()
	return out
}

func fkMatch(fk *foreignKey) string {
	conds := make([]string, 0, len(fk.ChildColumns))
	for i, c := range fk.ChildColumns {
		conds = append(conds, "c."+quoteIdent(c)+" = p."+quoteIdent(fk.ParentColumns[i]))
	}
	return strings.Join(conds, " AND ")
}

func (r *repository) crossTenantReferences(
	ctx context.Context,
	plan *purgePlan,
	tenant dbscope.Tenant,
) ([]cloudtenantports.CrossTenantReference, error) {
	ctx = dbscope.WithSystem(ctx, crossTenantReason)
	queries := plan.CrossTenantQueries()
	return dbtx.Read(
		ctx,
		r.db,
		func(ctx context.Context) ([]cloudtenantports.CrossTenantReference, error) {
			db := r.db.Keel(ctx)
			if err := bound(ctx, db); err != nil {
				return nil, err
			}
			out := make([]cloudtenantports.CrossTenantReference, 0)
			for _, query := range queries {
				rows, err := keel.QueryAll(
					ctx,
					db,
					pgx.RowToStructByName[cloudtenantports.CrossTenantReference],
					query,
					tenant.OrganizationID.String(),
					tenant.BusinessUnitID.String(),
				)
				if err != nil {
					return nil, fmt.Errorf(
						"look for other tenants' rows that refer to the tenant: %w",
						err,
					)
				}
				for _, row := range rows {
					if row.Rows > 0 {
						out = append(out, row)
					}
				}
			}
			return out, nil
		},
	)
}

func (r *repository) refuseCrossTenantReferences(
	ctx context.Context,
	plan *purgePlan,
	tenant dbscope.Tenant,
) error {
	refs, err := r.crossTenantReferences(ctx, plan, tenant)
	if err != nil {
		return err
	}
	if len(refs) > 0 {
		return &cloudtenantports.CrossTenantReferencesError{References: refs}
	}
	return nil
}
