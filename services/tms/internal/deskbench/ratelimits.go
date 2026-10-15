package deskbench

import (
	"context"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const matchRateMarker = "}:" + agentports.AgentMatchRateScope + ":"

type unthrottledMatches struct {
	kernelports.RateLimitStore
}

func (u unthrottledMatches) Check(
	ctx context.Context,
	requests []kernelports.RateLimitRequest,
) ([]kernelports.RateLimitDecision, error) {
	decisions := make([]kernelports.RateLimitDecision, len(requests))
	others := make([]kernelports.RateLimitRequest, 0, len(requests))
	positions := make([]int, 0, len(requests))
	for idx, request := range requests {
		if strings.Contains(request.Key, matchRateMarker) {
			decisions[idx] = kernelports.RateLimitDecision{Allowed: true}
			continue
		}
		others = append(others, request)
		positions = append(positions, idx)
	}
	if len(others) == 0 {
		return decisions, nil
	}

	checked, err := u.RateLimitStore.Check(ctx, others)
	if err != nil {
		return nil, err
	}
	for idx, decision := range checked {
		if idx < len(positions) {
			decisions[positions[idx]] = decision
		}
	}

	return decisions, nil
}
