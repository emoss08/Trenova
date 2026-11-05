package carriersettlementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/identity/domain/carriersettlementcontrol"
	"github.com/emoss08/trenova/internal/modules/settlement/domain/carriersettlement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPlanSeparateApprover_HoldsTheSubmitterToTheOrganizationsSetting(t *testing.T) {
	t.Parallel()

	submitter := pulid.MustNew("usr_")
	settlement := &carriersettlement.CarrierSettlement{SubmittedByID: submitter}
	required := &carriersettlementcontrol.CarrierSettlementControl{RequireSeparateApprover: true}
	optional := &carriersettlementcontrol.CarrierSettlementControl{}

	err := PlanSeparateApprover(required, settlement, submitter)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "someone else must approve it")

	require.NoError(t, PlanSeparateApprover(required, settlement, pulid.MustNew("usr_")))
	require.NoError(t, PlanSeparateApprover(optional, settlement, submitter))
}
