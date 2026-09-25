package agentruntime

import (
	"context"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/pulid"
)

// baselineCall is one write whose target is pinned as it is decided.
type baselineCall struct {
	req        *serviceports.RunRequest
	tool       serviceports.AgentTool
	call       serviceports.ToolCall
	proposalID pulid.ID
	persist    bool
}

// fileBaseline pins the record a write would change and, where the tool can
// say, what the write would do to it, both read from one snapshot. The
// preview is kept beside the proposal, keyed by the id minted for it here,
// never on the action: the action travels through every later activity of
// the turn, and a preview there would grow the workflow's history with each
// proposal. An evaluation keeps nothing. Nothing here can fail the call: a
// baseline that cannot be taken is simply not kept.
func (s *Service) fileBaseline(
	ctx context.Context,
	b *baselineCall,
) (*serviceports.ProposalTarget, *serviceports.ProposalBaselineResult) {
	if s.previews == nil {
		return s.snapshotTarget(ctx, b.req, b.tool, b.call), nil
	}

	result := s.previews.Baseline(ctx, &serviceports.ProposalBaselineRequest{
		ProposalID: b.proposalID,
		Tool:       b.tool,
		Params: serviceports.ToolExecuteParams{
			OrganizationID: b.req.Actor.OrganizationID,
			BusinessUnitID: b.req.Actor.BusinessUnitID,
			Actor:          b.req.Actor,
			IdempotencyKey: b.call.ID,
			RunID:          b.req.RunID,
			Params:         b.call.Arguments,
		},
		Persist: b.persist,
	})
	if result == nil {
		return s.snapshotTarget(ctx, b.req, b.tool, b.call), nil
	}

	return result.Target, result
}
