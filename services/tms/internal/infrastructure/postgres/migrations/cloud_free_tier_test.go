package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	cloudFreeTierUp   = "20261231008150_cloud_free_tier.tx.up.sql"
	cloudFreeTierDown = "20261231008150_cloud_free_tier.tx.down.sql"
)

func TestCloudFreeTierMigration_KeepsOneSubscriptionAndOnboardingPerOrganization(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, cloudFreeTierUp))

	assert.Contains(t, up, `CREATE UNIQUE INDEX IF NOT EXISTS "uq_organization_subscriptions_organization" ON "organization_subscriptions"("organization_id");`)
	assert.Contains(t, up, `CREATE UNIQUE INDEX IF NOT EXISTS "uq_organization_onboarding_organization" ON "organization_onboarding"("organization_id");`)
}

func TestCloudFreeTierMigration_AllowsOnePendingSignupPerAddress(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, cloudFreeTierUp))

	assert.Contains(t, up, `CREATE UNIQUE INDEX IF NOT EXISTS "uq_cloud_signups_pending_email" ON "cloud_signups"("email_normalized") WHERE "status" = 'pending';`)
	assert.Contains(t, up, `CREATE UNIQUE INDEX IF NOT EXISTS "uq_cloud_signups_token_hash" ON "cloud_signups"("token_hash");`)
}

func TestCloudFreeTierMigration_ProtectsEveryTableWithRowLevelSecurity(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, cloudFreeTierUp))

	assert.Contains(t, up, `SELECT trenova_rls.apply_policy( 'public.cloud_signups',`)
	assert.Contains(t, up, `trenova_rls.org_id()`)
	assert.Contains(t, up, `SELECT trenova_rls.reconcile();`)
	assert.NotContains(t, up, "global_tables",
		"signups hold password hashes and must never be readable by every tenant")
}

func TestCloudFreeTierMigration_CommentsEveryTable(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, cloudFreeTierUp))

	for _, table := range []string{"organization_subscriptions", "organization_onboarding", "cloud_signups"} {
		assert.Contains(t, up, `COMMENT ON TABLE "`+table+`" IS`)
	}
}

func TestCloudFreeTierMigration_DownRemovesWhatUpAdded(t *testing.T) {
	t.Parallel()

	down := compactSQL(readMigration(t, cloudFreeTierDown))

	for _, table := range []string{"cloud_signups", "organization_onboarding", "organization_subscriptions"} {
		assert.Contains(t, down, `DROP TABLE IF EXISTS "`+table+`";`)
	}
}
