//go:build integration

package cloud_test

import (
	"testing"

	"github.com/emoss08/keel"
	"github.com/emoss08/trenova/internal/modules/edi/domain/editransfer"
	"github.com/emoss08/trenova/internal/modules/edi/editest"
	"github.com/emoss08/trenova/internal/testutil/seedtest"
	"github.com/emoss08/trenova/pkg/dberror"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/jackc/pgx/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNoRowOfOneTenantCanNameAnotherTenant(t *testing.T) {
	tender := editest.SeedInternalTender(t)
	transfer := tender.NewTransfer(t, editransfer.TransferStatusPendingApproval)
	recipient := tender.NewRecipient(t, transfer)
	elsewhere := seedtest.SeedAdditionalTenant(t, tender.Ctx, tender.Admin, "XC")
	otherOrg, otherBU := elsewhere.Organization.ID.String(), elsewhere.BusinessUnit.ID.String()
	source := tender.Source.Info

	refused := func(t *testing.T, constraint, sql string, args ...any) {
		t.Helper()
		_, err := keel.Exec(tender.Ctx, tender.Admin, sql, args...)
		require.Error(t, err, constraint)
		assert.Equal(t, constraint, dberror.ExtractConstraintName(err), err.Error())
	}

	t.Run("in-network EDI stays inside one business unit", func(t *testing.T) {
		refused(t, "fk_edi_partners_internal_org",
			`UPDATE edi_partners SET internal_organization_id = $1 WHERE id = $2`,
			otherOrg, tender.SourcePartner.ID.String())
		refused(
			t,
			"ck_edi_load_tender_transfers_one_business_unit",
			`UPDATE edi_load_tender_transfers SET target_organization_id = $1, target_business_unit_id = $2
             WHERE id = $3`,
			otherOrg,
			otherBU,
			transfer.ID.String(),
		)
		refused(t, "fk_edi_tender_recipients_recipient_org",
			`UPDATE edi_tender_recipients SET recipient_organization_id = $1 WHERE id = $2`,
			otherOrg, recipient.ID.String())
		refused(
			t,
			"fk_edi_connections_target_org",
			`INSERT INTO edi_connections (id, business_unit_id, source_organization_id, target_organization_id, method)
             VALUES ($1, $2, $3, $4, 'Internal')`,
			pulid.MustNew("edic_").String(),
			source.BuID.String(),
			source.OrgID.String(),
			otherOrg,
		)
	})

	t.Run(
		"a membership names a person and an organization of its own business unit",
		func(t *testing.T) {
			refused(
				t,
				"fk_user_organization_memberships_organization",
				`INSERT INTO user_organization_memberships (id, user_id, business_unit_id, organization_id)
             VALUES ($1, $2, $3, $4)`,
				pulid.MustNew("mem_").String(),
				source.UserID.String(),
				source.BuID.String(),
				otherOrg,
			)
			refused(
				t,
				"fk_user_organization_memberships_user",
				`INSERT INTO user_organization_memberships (id, user_id, business_unit_id, organization_id)
             VALUES ($1, $2, $3, $4)`,
				pulid.MustNew("mem_").String(),
				source.UserID.String(),
				otherBU,
				otherOrg,
			)
		},
	)

	t.Run("every reference to another organization carries its business unit", func(t *testing.T) {
		loose, err := keel.QueryAll(tender.Ctx, tender.Admin, pgx.RowTo[string], `
SELECT c.relname || '.' || a.attname
FROM pg_catalog.pg_class c
JOIN pg_catalog.pg_namespace n ON n.oid = c.relnamespace
JOIN pg_catalog.pg_attribute a ON a.attrelid = c.oid AND a.attnum > 0 AND NOT a.attisdropped
WHERE n.nspname = 'public' AND c.relkind IN ('r', 'p') AND NOT c.relispartition
  AND a.attname LIKE '%organization_id' AND a.attname <> 'organization_id'
  AND NOT EXISTS (SELECT 1 FROM trenova_rls.global_tables g WHERE g.table_name = c.relname)
  AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_constraint pc
      WHERE pc.conrelid = c.oid AND pc.contype IN ('f', 'c')
        AND a.attnum = ANY(pc.conkey)
        AND (
            pc.contype = 'c'
            OR EXISTS (
                SELECT 1 FROM pg_catalog.pg_attribute b
                WHERE b.attrelid = c.oid AND b.attnum = ANY(pc.conkey)
                  AND b.attname LIKE '%business_unit_id'
            )
        )
  )
  AND NOT EXISTS (
      SELECT 1 FROM pg_catalog.pg_constraint pc
      WHERE pc.conrelid = c.oid AND pc.contype = 'c'
        AND pg_get_constraintdef(pc.oid) LIKE '%business_unit_id%'
  )
ORDER BY 1`)
		require.NoError(t, err)
		assert.Equal(t, []string{"users.current_organization_id"}, loose,
			"a user's current organization is held to the user's business unit by "+
				"trigger_check_user_organization_business_unit; every other column naming an "+
				"organization is held by a composite key or a check")
	})
}
