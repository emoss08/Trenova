package deskbench

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type refusingRates struct {
	keys []string
}

func (r *refusingRates) Check(
	_ context.Context,
	requests []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	decisions := make([]kernelports.RateLimitDecision, 0, len(requests))
	for _, request := range requests {
		r.keys = append(r.keys, request.Key)
		decisions = append(decisions, kernelports.RateLimitDecision{Allowed: false})
	}

	return decisions, nil
}

func TestUnthrottledMatches_LetsEveryMatchThroughAndAsksAboutTheRest(t *testing.T) {
	t.Parallel()

	inner := &refusingRates{}
	store := unthrottledMatches{RateLimitStore: inner}

	decisions, err := store.Check(t.Context(), []kernelports.RateLimitRequest{
		{Key: "{org_1}:agent-match:usr_1"},
		{Key: "{org_1}:carrier-lookup:usr_1"},
		{Key: "{org_1}:agent-match:usr_2"},
	})

	require.NoError(t, err)
	assert.Equal(t, []bool{true, false, true}, []bool{
		decisions[0].Allowed,
		decisions[1].Allowed,
		decisions[2].Allowed,
	})
	assert.Equal(t, []string{"{org_1}:carrier-lookup:usr_1"}, inner.keys,
		"only the bench's matches are exempt; every other limit still holds")
}
