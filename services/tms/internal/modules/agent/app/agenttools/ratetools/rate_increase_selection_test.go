package ratetools

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/agenttoolstest"
	"github.com/emoss08/trenova/internal/modules/agent/app/agenttools/toolparams"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agenttoolports"
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

	agreements := &fakeAgreements{guard: &agenttoolstest.WriteGuard{}}
	tool := NewApplyRateIncreaseTool(agreements)
	named := pulid.MustNew("rag_")
	params := map[string]any{
		toolparams.ParamEffectiveFrom: "2027-01-01",
		paramAgreementIDs:             []any{named.String()},
		toolparams.ParamCustomerID:    pulid.MustNew("cus_").String(),
		toolparams.ParamCarrierID:     "the carrier on the page",
		paramPartyType:                "Customer",
		paramPercentChange:            "3.5",
	}

	resolved, err := tool.(agenttoolports.ToolSelectionResolver).ResolveSelection(
		t.Context(), agenttoolstest.ExecuteParams(params),
	)
	require.NoError(t, err)
	assert.Equal(t, map[string]any{
		toolparams.ParamEffectiveFrom: "2027-01-01",
		paramAgreementIDs:             []any{named.String()},
		paramPercentChange:            "3.5",
	}, resolved)
	assert.Len(t, params, 6, "the model's own call is left as it sent it")

	require.NoError(
		t,
		tool.(agenttoolports.ToolValidator).Validate(
			t.Context(),
			agenttoolstest.ExecuteParams(params),
		),
		"a scope beside named agreements is never read, so it cannot refuse the call",
	)

	approved := agenttoolstest.ExecuteParams(params)
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

	tool := NewApplyRateIncreaseTool(&fakeAgreements{guard: &agenttoolstest.WriteGuard{}})
	params := map[string]any{
		toolparams.ParamEffectiveFrom: "2027-01-01",
		toolparams.ParamCustomerID:    pulid.MustNew("cus_").String(),
		paramPercentChange:            "3.5",
	}

	resolved, err := tool.(agenttoolports.ToolSelectionResolver).ResolveSelection(
		t.Context(), agenttoolstest.ExecuteParams(params),
	)
	require.NoError(t, err)
	assert.Equal(t, params, resolved)

	params[toolparams.ParamCustomerID] = "acme"
	require.Error(
		t,
		tool.(agenttoolports.ToolValidator).Validate(
			t.Context(),
			agenttoolstest.ExecuteParams(params),
		),
		"without named agreements the scope is read, and a bad one is refused",
	)
}
