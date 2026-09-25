package agentproposalbaselinerepository

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/uptrace/bun"
	"github.com/uptrace/bun/dialect/pgdialect"
)

func TestPurgeQuery_DeletesOnlyOldBaselinesNoProposalNames(t *testing.T) {
	t.Parallel()

	sql := purgeQuery(bun.NewDB(nil, pgdialect.New()), 1767225600, 250).String()

	assert.Contains(t, sql, `DELETE FROM "agent_proposal_baselines" AS "apb"`)
	assert.Contains(t, sql,
		"(apb.proposal_id, apb.organization_id, apb.business_unit_id) IN (SELECT")
	assert.Contains(t, sql, "apb.created_at < 1767225600")
	assert.Contains(t, sql, `NOT EXISTS (SELECT 1 FROM "agent_proposals" AS "ap"`)
	assert.Contains(t, sql, "ap.id = apb.proposal_id")
	assert.Contains(t, sql, "ap.organization_id = apb.organization_id")
	assert.Contains(t, sql, "ap.business_unit_id = apb.business_unit_id")
	assert.Contains(t, sql, "LIMIT 250")
}

func TestCreate_KeepsTheFirstBaselineForAProposal(t *testing.T) {
	t.Parallel()

	baseline := &agent.ProposalBaseline{
		ProposalID:     pulid.MustNew("ap_"),
		OrganizationID: pulid.MustNew("org_"),
		BusinessUnitID: pulid.MustNew("bu_"),
		ToolName:       "cancel_shipment",
		Preview:        &agent.ToolPreview{Summary: "Would cancel."},
	}

	sql := bun.NewDB(nil, pgdialect.New()).NewInsert().
		Model(baseline).
		On(conflictTarget).
		String()

	assert.Contains(t, sql,
		"ON CONFLICT (proposal_id, organization_id, business_unit_id) DO NOTHING")
}
