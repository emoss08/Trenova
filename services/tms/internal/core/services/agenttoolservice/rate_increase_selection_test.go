package agenttoolservice

import (
	"testing"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

/*
A model that listed one customer's agreements sent the customerId back beside
their ids, with a carrierId as well, and the increase was refused for
targeting a customer and a carrier at once. The ids are all the call can
move, so the scopes are set aside and the proposal holds only the agreements.
*/
func TestApplyRateIncrease_SetsAsideTheScopesNamedAgreementsWereFoundBy(t *testing.T) {
	t.Parallel()

	agreements := &fakeAgreements{guard: &writeGuard{}}
	tool := newApplyRateIncreaseTool(agreements)
	named := pulid.MustNew("rag_")
	params := map[string]any{
		paramEffectiveFrom: "2027-01-01",
		paramAgreementIDs:  []any{named.String()},
		paramCustomerID:    pulid.MustNew("cus_").String(),
		paramCarrierID:     "the carrier on the page",
		paramPartyType:     "Customer",
		paramPercentChange: "3.5",
	}

	resolved, err := tool.(serviceports.ToolSelectionResolver).ResolveSelection(
		t.Context(), executeParams(params),
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		paramEffectiveFrom: "2027-01-01",
		paramAgreementIDs:  []any{named.String()},
		paramPercentChange: "3.5",
	}, resolved)
	assert.Len(t, params, 6, "the model's own call is left as it sent it")

	require.NoError(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(params)),
		"a scope beside named agreements is never read, so it cannot refuse the call")

	approved := executeParams(params)
	approved.ProposalID = pulid.MustNew("ap_")
	require.NoError(t, tool.Execute(t.Context(), approved))
	require.NotNil(t, agreements.increased)
	assert.Equal(t, []pulid.ID{named}, agreements.increased.AgreementIDs)
	assert.Nil(t, agreements.increased.CustomerID)
	assert.Nil(t, agreements.increased.CarrierID)
	assert.Empty(t, agreements.increased.PartyType)
}

func TestApplyRateIncrease_LeavesAScopedCallAsItIs(t *testing.T) {
	t.Parallel()

	tool := newApplyRateIncreaseTool(&fakeAgreements{guard: &writeGuard{}})
	params := map[string]any{
		paramEffectiveFrom: "2027-01-01",
		paramCustomerID:    pulid.MustNew("cus_").String(),
		paramPercentChange: "3.5",
	}

	resolved, err := tool.(serviceports.ToolSelectionResolver).ResolveSelection(
		t.Context(), executeParams(params),
	)
	require.NoError(t, err)
	assert.Equal(t, params, resolved)

	params[paramCustomerID] = "acme"
	require.Error(t, tool.(serviceports.ToolValidator).Validate(t.Context(), executeParams(params)),
		"without named agreements the scope is read, and a bad one is refused")
}
