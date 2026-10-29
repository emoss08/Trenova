package toolaccess

import (
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/aiusage"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentruntimeports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/aiusageports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// ToolSetState is a turn's tool set as data. Which tools are loaded follows
// from Specs.
type ToolSetState struct {
	Specs      []kernelports.ToolSpec `json:"specs"`
	Deferred   []string               `json:"deferred,omitempty"`
	Loaded     []string               `json:"loaded,omitempty"`
	Pending    []string               `json:"pending,omitempty"`
	Allowed    []string               `json:"allowed"`
	Disclosed  bool                   `json:"disclosed"`
	Unattended bool                   `json:"unattended"`
	FindCalls  int                    `json:"findCalls"`
	Beyond     []string               `json:"beyond,omitempty"`
	Query      *QueryVectorState      `json:"query,omitempty"`
}

func RestoreToolSet(state *ToolSetState) *ToolSet {
	set := &ToolSet{
		Loaded:     make(map[string]struct{}, len(state.Specs)),
		Allowed:    state.Allowed,
		Disclosed:  state.Disclosed,
		unattended: state.Unattended,
		FindCalls:  state.FindCalls,
		Beyond:     state.Beyond,
		Query:      state.Query.unpack(),
	}
	for _, spec := range state.Specs {
		set.add(spec)
	}
	set.Deferred = state.Deferred
	for _, name := range state.Loaded {
		set.Loaded[name] = struct{}{}
	}
	set.Pending = slices.Clone(state.Pending)
	set.frozen = true

	return set
}

func TurnAttribution(req *agentruntimeports.RunRequest) aiusageports.AIUsageAttribution {
	definition := req.Definition
	version := definition.Version
	attribution := aiusageports.AIUsageAttribution{
		UserID:            req.Actor.PersonUserID(),
		AgentDefinitionID: definition.ID,
		ThreadID:          req.ThreadID,
		RunID:             req.RunID,
		Purpose:           req.AttributedPurpose(),
		Feature:           aiusage.FeatureAgentTurn,
		DefinitionVersion: &version,
	}
	if req.StepOwner.ID.IsNotNil() {
		attribution.OwnerKind = req.StepOwner.Kind
		attribution.OwnerID = req.StepOwner.ID
	}
	if req.Delegation != nil {
		attribution.DelegateCallID = req.Delegation.CallID
	}

	return attribution
}
