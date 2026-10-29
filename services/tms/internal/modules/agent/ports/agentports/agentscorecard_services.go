package agentports

import (
	"context"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentevaluation"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agenttooltrust"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
)

type AgentScorecardRequest struct {
	TenantInfo        pagination.TenantInfo
	AgentDefinitionID pulid.ID
	Window            agentevaluation.ScorecardWindow
}

// AgentScorecardResult is the counted record and the trust ladder behind it.
//
// The ladder is not windowed. A tier is earned over the whole life of the
// agent and taken back the same way, so showing only the last thirty days of
// it would describe a different ladder from the one deciding what runs
// without asking.
type AgentScorecardResult struct {
	Scorecard *agentevaluation.Scorecard
	ToolTrust []*agenttooltrust.ToolTrust
}

type AgentScorecardService interface {
	Get(
		ctx context.Context,
		req AgentScorecardRequest,
		actor *permissionports.RequestActor,
	) (*AgentScorecardResult, error)
}
