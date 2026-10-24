package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	aiAuditLedgerUp = "20261231006750_ai_audit_ledger"
)

func TestAIAuditLedgerMigration_KeysEachTenantsChain(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, aiAuditLedgerUp))

	for _, fragment := range []string{
		`CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_audit_events_source_key" ON "ai_audit_events"("organization_id", "business_unit_id", "source_key");`,
		`CREATE UNIQUE INDEX IF NOT EXISTS "uq_ai_audit_events_seq" ON "ai_audit_events"("organization_id", "business_unit_id", "seq");`,
		`("organization_id", "business_unit_id", "occurred_at" DESC, "id" DESC)`,
		`WHERE "decided_by_user_id" IS NOT NULL`,
		`CONSTRAINT "pk_ai_audit_chain_heads" PRIMARY KEY ("organization_id", "business_unit_id")`,
		`CONSTRAINT "pk_ai_audit_seals" PRIMARY KEY ("organization_id", "business_unit_id", "to_seq")`,
		`CONSTRAINT "pk_ai_audit_projector_state" PRIMARY KEY ("source")`,
		`CHECK (("hash_version" = 1) = ("hash_key_id" IS NOT NULL))`,
		`"cost_usd" NUMERIC(14, 6)`,
	} {
		assert.Contains(t, up, fragment)
	}
}

func TestAIAuditLedgerMigration_RowsCanOnlyBePrunedBySetting(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, aiAuditLedgerUp))

	assert.Contains(t, up, `BEFORE UPDATE OR DELETE ON "ai_audit_events" FOR EACH ROW`)
	assert.Contains(t, up, `BEFORE TRUNCATE ON "ai_audit_events" FOR EACH STATEMENT`)
	assert.Contains(t, up, `current_setting('trenova.ai_audit_prune', TRUE)`)
	assert.Contains(t, up, `IF TG_OP = 'UPDATE' THEN RAISE EXCEPTION`)
}

func TestAIAuditLedgerMigration_SealsAreNotAnchoredExternally(t *testing.T) {
	t.Parallel()

	up := readMigration(t, aiAuditLedgerUp)

	assert.NotContains(t, up, "anchored_at")
	assert.NotContains(t, up, "anchor_ref")
}
