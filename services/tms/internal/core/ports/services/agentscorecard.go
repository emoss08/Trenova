package services

import (
	"context"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentScorecardRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Window            agent.ScorecardWindow
}

// AgentScorecardResult is the counted record and the trust ladder behind it.
//
// The ladder is not windowed. A tier is earned over the whole life of the
// agent and taken back the same way, so showing only the last thirty days of
// it would describe a different ladder from the one deciding what runs
// without asking.
type AgentScorecardResult struct {
	Scorecard *agent.Scorecard
	ToolTrust []*agent.ToolTrust
}

type AgentScorecardService interface {
	Get(
		ctx context.Context,
		req AgentScorecardRequest,
		actor *RequestActor,
	) (*AgentScorecardResult, error)
}
