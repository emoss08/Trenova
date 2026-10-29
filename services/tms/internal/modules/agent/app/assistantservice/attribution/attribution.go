package attribution

import (
	"github.com/emoss08/trenova/internal/modules/agent/domain/assistantartifact"
	"github.com/emoss08/trenova/shared/pulid"
)

// proposer is the agent behind a proposal or a plan.
type proposer struct {
	Id   pulid.ID
	Name string
}

// AgentIdentity is how an agent is named and drawn where its work is shown.
type AgentIdentity struct {
	Name   string
	Icon   string
	Accent string
}

func NamesItsReader(kind assistantartifact.Kind) bool {
	//nolint:exhaustive // only the kinds whose card names the agent
	switch kind {
	case assistantartifact.KindTableView,
		assistantartifact.KindReportRun,
		assistantartifact.KindReportPreview:
		return true
	default:
		return false
	}
}

type ArtifactReader struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Icon   string `json:"icon"`
	Accent string `json:"accent"`
}
