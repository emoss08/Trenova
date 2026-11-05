package driversettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/settlementcontrol"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/driversettlement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanSeparateApprover_HoldsTheSubmitterToTheOrganizationsSetting(t *testing.T) {
	t.Parallel()

	submitter := pulid.MustNew("usr_")
	settlement := &driversettlement.Settlement{SubmittedByID: submitter}
	required := &settlementcontrol.SettlementControl{RequireSeparateApprover: true}
	optional := &settlementcontrol.SettlementControl{}

	err := PlanSeparateApprover(required, settlement, submitter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "someone else must approve it")

	require.NoError(t, PlanSeparateApprover(required, settlement, pulid.MustNew("usr_")))
	require.NoError(t, PlanSeparateApprover(optional, settlement, submitter))
}

func TestPlanSeparateApprover_ReachesTheOneWhoSubmitsFromDraft(t *testing.T) {
	t.Parallel()

	userID := pulid.MustNew("usr_")
	settlement := &driversettlement.Settlement{Status: driversettlement.StatusDraft}
	required := &settlementcontrol.SettlementControl{RequireSeparateApprover: true}

	require.NoError(t, PlanSubmit(settlement, userID, 100))

	require.Error(t, PlanSeparateApprover(required, settlement, userID))
}
