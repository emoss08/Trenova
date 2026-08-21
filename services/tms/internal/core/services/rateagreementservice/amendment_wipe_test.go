package rateagreementservice

import (
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/rateagreement"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A save that carries no lanes at all is indistinguishable, at this layer, from
// a user who deleted every one of them: each existing lane is closed out and
// nothing replaces it. The panel is what has to guarantee the payload was ever
// seated from the record, because from here the two look the same.
func TestPlanRuleAmendment_ASaveWithNoLanesClosesOutEveryLane(t *testing.T) {
	t.Parallel()

	existing := []*rateagreement.RateAgreementRule{
		{ID: pulid.MustNew("ragr_"), Label: "Dallas to Chicago"},
		{ID: pulid.MustNew("ragr_"), Label: "Chicago to Dallas"},
	}

	plan, multiErr := planRuleAmendment(existing, nil, 1700000000)

	require.Nil(t, multiErr)
	require.NotNil(t, plan)
	assert.Len(t, plan.SupersededIDs, 2)
	assert.Empty(t, plan.Inserts,
		"nothing replaces them, so the contract is left with no open lanes at all")
}
