package seedaccountrepository

import (
	"slices"

	"github.com/emoss08/trenova/pkg/buncolgen"
)

const (
	idColumn    = "id"
	ownerColumn = "user_id"

	catalogReferencesQuery = `SELECT child.relname AS child_table,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.conkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.conrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS child_columns,
       ARRAY(
           SELECT att.attname::text
           FROM unnest(con.confkey) WITH ORDINALITY AS k(attnum, ord)
           JOIN pg_catalog.pg_attribute att ON att.attrelid = con.confrelid AND att.attnum = k.attnum
           ORDER BY k.ord
       ) AS parent_columns
FROM pg_catalog.pg_constraint con
JOIN pg_catalog.pg_class child ON child.oid = con.conrelid
JOIN pg_catalog.pg_class parent ON parent.oid = con.confrelid
JOIN pg_catalog.pg_namespace n ON n.oid = child.relnamespace
JOIN pg_catalog.pg_namespace pn ON pn.oid = parent.relnamespace
WHERE con.contype = 'f'
  AND n.nspname = 'public'
  AND pn.nspname = 'public'
  AND parent.relname = ?
  AND NOT child.relispartition
ORDER BY child.relname, con.conname`

	cappedCountQuery         = "SELECT count(*) FROM (SELECT 1 FROM ? WHERE ? = ? LIMIT ?) AS refs"
	cappedCountExcludingSelf = "SELECT count(*) FROM (SELECT 1 FROM ? WHERE ? = ? AND ? <> ? LIMIT ?) AS refs"
	cappedCountExcludingOwns = "SELECT count(*) FROM (SELECT 1 FROM ? WHERE ? = ? AND ? NOT IN (?) LIMIT ?) AS refs"
)

type catalogReference struct {
	ChildTable    string   `bun:"child_table"`
	ChildColumns  []string `bun:"child_columns,array"`
	ParentColumns []string `bun:"parent_columns,array"`
}

func (c *catalogReference) columnFor(parentColumn string) (string, bool) {
	index := slices.Index(c.ParentColumns, parentColumn)
	if index < 0 || index >= len(c.ChildColumns) {
		return "", false
	}

	return c.ChildColumns[index], true
}

type referenceColumn struct {
	Table  string
	Column string
}

func referenceColumns(refs []catalogReference) []referenceColumn {
	out := make([]referenceColumn, 0, len(refs))
	seen := make(map[referenceColumn]struct{}, len(refs))
	for i := range refs {
		column, ok := refs[i].columnFor(idColumn)
		if !ok {
			continue
		}
		key := referenceColumn{Table: refs[i].ChildTable, Column: column}
		if _, dup := seen[key]; dup {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, key)
	}

	return out
}

func ownedByUserTables() map[string]struct{} {
	return map[string]struct{}{
		buncolgen.OrganizationMembershipTable.Name: {},
		buncolgen.UserRoleAssignmentTable.Name:     {},
		buncolgen.MFAAuthenticatorTable.Name:       {},
		buncolgen.PasswordResetTokenTable.Name:     {},
	}
}

func membershipTables() map[string]struct{} {
	return map[string]struct{}{
		buncolgen.OrganizationMembershipTable.Name: {},
		buncolgen.UserRoleAssignmentTable.Name:     {},
	}
}

func historyTables() map[string]struct{} {
	return map[string]struct{}{
		buncolgen.EntryTable.Name:            {},
		buncolgen.AuthEventTable.Name:        {},
		buncolgen.AIAuditEventTable.Name:     {},
		buncolgen.AIAuditChainHeadTable.Name: {},
		buncolgen.AIAuditSealTable.Name:      {},
		buncolgen.AIAuditExportTable.Name:    {},
		"ai_logs":                            {},
	}
}

func seedDefaultTables() map[string]struct{} {
	return map[string]struct{}{
		buncolgen.AccountingControlTable.Name:        {},
		buncolgen.BillingControlTable.Name:           {},
		buncolgen.InvoiceAdjustmentControlTable.Name: {},
		buncolgen.DispatchControlTable.Name:          {},
		buncolgen.ShipmentControlTable.Name:          {},
		buncolgen.DocumentControlTable.Name:          {},
		buncolgen.DataEntryControlTable.Name:         {},
		buncolgen.CostingControlTable.Name:           {},
		buncolgen.CostCategoryTable.Name:             {},
		buncolgen.CostCategoryGLAccountTable.Name:    {},
		buncolgen.SequenceTable.Name:                 {},
		buncolgen.RoleTable.Name:                     {},
		buncolgen.ResourcePermissionTable.Name:       {},
		buncolgen.GLAccountTable.Name:                {},
		buncolgen.AccountTypeTable.Name:              {},
		buncolgen.DocumentTypeTable.Name:             {},
		buncolgen.DocumentTemplateTable.Name:         {},
		buncolgen.DocumentTemplateVersionTable.Name:  {},
		buncolgen.ReasonCodeTable.Name:               {},
		buncolgen.TCAAllowlistedTableTable.Name:      {},
		buncolgen.DefinitionTable.Name:               {},
		buncolgen.JurisdictionRuleTable.Name:         {},
		buncolgen.JurisdictionTable.Name:             {},
	}
}
