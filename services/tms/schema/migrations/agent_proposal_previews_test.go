package migrations

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

const (
	agentProposalPreviewsUp = "20261231006790_agent_proposal_previews"
)

func TestAgentProposalPreviewsMigration_KeysBaselinesByProposalAndTenant(t *testing.T) {
	t.Parallel()

	up := compactSQL(readMigration(t, agentProposalPreviewsUp))

	for _, fragment := range []string{
		`CONSTRAINT "pk_agent_proposal_baselines" PRIMARY KEY ("proposal_id", "organization_id", "business_unit_id")`,
		`REFERENCES "organizations"("id") ON UPDATE NO ACTION ON DELETE CASCADE`,
		`REFERENCES "business_units"("id") ON UPDATE NO ACTION ON DELETE CASCADE`,
		`CREATE INDEX IF NOT EXISTS "idx_agent_proposal_baselines_created_at"`,
		`ADD COLUMN IF NOT EXISTS "preview" jsonb`,
		`ADD COLUMN IF NOT EXISTS "preview_digest" varchar(64)`,
		`ADD COLUMN IF NOT EXISTS "preview_reviewed" boolean NOT NULL DEFAULT false`,
		`ADD COLUMN IF NOT EXISTS "preview_target_version" bigint`,
		`"preview_digest" IS NULL OR "preview_digest" ~ '^[0-9a-f]{64}$'`,
		`NOT "preview_reviewed" OR "preview_digest" IS NOT NULL`,
	} {
		assert.Contains(t, up, fragment)
	}

	assert.NotContains(t, up, `REFERENCES "agent_proposals"`,
		"a baseline is written before its proposal row exists")
}
