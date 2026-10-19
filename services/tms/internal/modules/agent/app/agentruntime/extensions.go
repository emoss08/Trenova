package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agent"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"go.uber.org/zap"
)

const externalContentNote = " It was not run automatically because this turn read content written " +
	"outside the organization (a web page, or a trading partner's text or guide it was given), " +
	"and a change made after reading outside content always waits for a person."

var outsideTextTools = []string{"edi_inspect_x12", "edi_draft_connection"}

func (s *Service) activeExtensions(
	ctx context.Context,
	actor *permissionports.RequestActor,
) agentextension.Active {
	if s.extensions == nil || actor == nil {
		return agentextension.Active{}
	}

	active, err := s.extensions.ActiveCapabilities(ctx, actor.TenantInfo())
	if err != nil {
		s.logger.Warn("could not read the organization's extensions; withholding their tools",
			zap.String("organization", actor.OrganizationID.String()), zap.Error(err))
		return agentextension.Active{}
	}

	return active
}

func withGrants(held, granted []string) []string {
	for _, name := range granted {
		if !slices.Contains(held, name) {
			held = append(held, name)
		}
	}

	return held
}

func onlyOffered(names []string, active agentextension.Active) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		if active.Offers(name) {
			kept = append(kept, name)
		}
	}

	return kept
}

func touchesExtensions(names []string) bool {
	return slices.ContainsFunc(names, agentextension.IsExtensionTool)
}

func (s *Service) holdsFor(ctx context.Context, req *agentports.RunRequest, name string) bool {
	if s.holds(req.Definition, name) {
		return true
	}
	if !agentextension.IsExtensionTool(name) {
		return false
	}

	return s.activeExtensions(ctx, req.Actor).GrantsEveryAgent(name)
}

func (s *Service) extensionRefusal(
	ctx context.Context,
	req *agentports.RunRequest,
	name string,
) (toolOutcome, bool) {
	capability, isExtension := agentextension.CapabilityForTool(name)
	if !isExtension {
		return toolOutcome{}, false
	}
	if s.activeExtensions(ctx, req.Actor).Offers(name) {
		return toolOutcome{}, false
	}
	spec, _ := agentextension.CapabilitySpecFor(capability)

	return failedOutcome("Tool %q comes with an extension this organization has turned off "+
		"or not set up, so it was not run. Answer without it, and say %s.", name,
		spec.Unchecked), true
}

func ReadsExternalContent(name string) bool {
	return slices.Contains(outsideTextTools, name) ||
		agentextension.ReturnsExternalContent(name)
}

// decisionTaint is the taint a call is decided with: what the turn had read,
// and, when it has read the web, a web mark even where a turn from before
// taint was kept could not say so. A nil taint stays nil, which Decide already
// reads as tainted.
func decisionTaint(taint *agent.RunTaint, external bool) *agent.RunTaint {
	if !external || taint == nil {
		return taint
	}

	decided := taint.Clone()
	decided.Add(
		agent.TaintMark{Source: agent.TaintSourceWeb, ToolName: agentextension.ToolWebSearch},
	)

	return decided
}

func afterExternalContent(tier agent.AutonomyTier, external bool) (agent.AutonomyTier, bool) {
	if !external || !tier.Above(agent.TierPropose) {
		return tier, false
	}

	return agent.TierPropose, true
}

func withSummaries(
	summaries []agentdefinition.ToolSummary,
	extra []agentdefinition.ToolSummary,
) []agentdefinition.ToolSummary {
	for _, summary := range extra {
		if !slices.ContainsFunc(summaries, func(existing agentdefinition.ToolSummary) bool {
			return existing.Name == summary.Name
		}) {
			summaries = append(summaries, summary)
		}
	}

	return summaries
}
