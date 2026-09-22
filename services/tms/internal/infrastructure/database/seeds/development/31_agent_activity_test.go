package development

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/domain/shipment"
	"github.com/emoss08/trenova/internal/core/domain/tenant"
	"github.com/emoss08/trenova/internal/core/domain/worker"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A seed is only ever exercised against a real database, which is the one place
these builders cannot be run here. So the contract is checked directly: every
row a seed hands to an insert has to satisfy the domain's own validation,
because the alternative is finding out at `task db-seed` with a constraint
violation and no line number.
*/

// orgStub and definitionStub stand in for the rows loadRefs would have read.
// The builders only ever read ids and names off them, so a full record would
// be noise that hides which fields the seed actually depends on.
func orgStub(orgID, buID pulid.ID) *tenant.Organization {
	return &tenant.Organization{ID: orgID, BusinessUnitID: buID}
}

func definitionStub() *agentdefinition.Definition {
	return &agentdefinition.Definition{ID: pulid.MustNew("adef_")}
}

func userStub() *tenant.User {
	return &tenant.User{ID: pulid.MustNew("usr_")}
}

func activityRefs(t *testing.T) *activitySeedRefs {
	t.Helper()

	orgID := pulid.MustNew("org_")
	buID := pulid.MustNew("bu_")
	shipments := make([]*shipment.Shipment, 0, 4)
	for i := range 4 {
		shipments = append(shipments, &shipment.Shipment{
			ID:             pulid.MustNew("shp_"),
			OrganizationID: orgID,
			BusinessUnitID: buID,
			ProNumber:      "S0000" + string(rune('1'+i)),
			Status:         shipment.StatusInTransit,
			Version:        3,
		})
	}

	return &activitySeedRefs{
		org:       orgStub(orgID, buID),
		shipments: shipments,
		customers: []*customer.Customer{
			{ID: pulid.MustNew("cus_")},
			{ID: pulid.MustNew("cus_")},
		},
		worker: &worker.Worker{ID: pulid.MustNew("wrk_")},
		now:    1_800_000_000,
	}
}

func TestAgentActivitySeed_EveryProposalIsValid(t *testing.T) {
	refs := activityRefs(t)
	refs.dispatch = definitionStub()
	refs.billing = definitionStub()

	for _, spec := range decisionSpecs(refs) {
		for _, proposal := range spec.proposals {
			proposal.OrganizationID = refs.org.ID
			proposal.BusinessUnitID = refs.org.BusinessUnitID
			proposal.RunID = pulid.MustNew("arun_")

			multiErr := errortypes.NewMultiError()
			proposal.Validate(multiErr)
			assert.Falsef(t, multiErr.HasErrors(),
				"proposal %q is not valid: %v", proposal.ToolName, multiErr)
		}
	}
}

// A decision the queue cannot batch is a decision that teaches nothing about
// batching, so at least two rows have to share a tool.
func TestAgentActivitySeed_TwoDecisionsShareATool(t *testing.T) {
	refs := activityRefs(t)
	refs.dispatch = definitionStub()
	refs.billing = definitionStub()

	byTool := map[string]int{}
	for _, spec := range decisionSpecs(refs) {
		for _, proposal := range spec.proposals {
			if proposal.Status == agent.ProposalStatusPending {
				byTool[proposal.ToolName]++
			}
		}
	}

	shared := 0
	for _, count := range byTool {
		if count > 1 {
			shared++
		}
	}
	assert.Positivef(t, shared,
		"no tool has two pending proposals, so the queue cannot show a batch decision")
}

// The queue is only a queue if something is waiting, and only a history if
// something is not.
func TestAgentActivitySeed_HasBothWaitingAndDecided(t *testing.T) {
	refs := activityRefs(t)
	refs.dispatch = definitionStub()
	refs.billing = definitionStub()

	var pending, settled int
	for _, spec := range decisionSpecs(refs) {
		for _, proposal := range spec.proposals {
			if proposal.Status == agent.ProposalStatusPending {
				pending++
			} else {
				settled++
			}
		}
	}

	assert.GreaterOrEqual(t, pending, 3, "the decisions queue needs rows to decide")
	assert.Positive(t, settled, "the queue needs a decided row so the history is not empty")
}

func TestAgentActivitySeed_TrustLedgerShowsAPromotionAndADemotion(t *testing.T) {
	refs := activityRefs(t)
	refs.dispatch = definitionStub()
	refs.billing = definitionStub()

	rows := trustRows(refs)
	require.NotEmpty(t, rows)

	var promoted, demoted int
	for _, row := range rows {
		multiErr := errortypes.NewMultiError()
		row.Validate(multiErr)
		assert.Falsef(t, multiErr.HasErrors(),
			"trust row %q is not valid: %v", row.ToolName, multiErr)

		if row.PromotedAt != nil {
			promoted++
		}
		if row.DemotedAt != nil {
			demoted++
		}
	}

	// The panel's whole point is the record behind a tier. A ledger where
	// nothing ever moved has no promotion to explain.
	assert.Positive(t, promoted, "no tool was ever promoted, so there is nothing to explain")
	assert.Positive(t, demoted, "no tool was ever demoted, so a demotion cannot be seen")
}
