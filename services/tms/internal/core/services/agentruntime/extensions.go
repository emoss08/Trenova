package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/core/domain/agent"
	"github.com/emoss08/trenova/internal/core/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/core/domain/agentextension"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"go.uber.org/zap"
)

const externalContentNote = " It was not run automatically because this turn read content from " +
	"the web, and a change made after reading outside content always waits for a person."

type activeExtensions map[agentextension.Type]agentextension.Availability

func (s *Service) activeExtensions(
	ctx context.Context,
	actor *serviceports.RequestActor,
) activeExtensions {
	if s.extensions == nil || actor == nil {
		return activeExtensions{}
	}

	active, err := s.extensions.ActiveExtensions(ctx, actor.TenantInfo())
	if err != nil {
		s.logger.Warn("could not read the organization's extensions; withholding their tools",
			zap.String("organization", actor.OrganizationID.String()), zap.Error(err))
		return activeExtensions{}
	}

	return active
}

func (a activeExtensions) offers(name string) bool {
	typ, isExtension := agentextension.ExtensionForTool(name)
	if !isExtension {
		return true
	}
	_, active := a[typ]

	return active
}

func (a activeExtensions) grants() []string {
	granted := make([]string, 0, len(a)*2)
	for _, typ := range agentextension.AllTypes() {
		if a[typ] != agentextension.AvailabilityAllAgents {
			continue
		}
		spec, ok := agentextension.SpecFor(typ)
		if !ok {
			continue
		}
		granted = append(granted, spec.Tools...)
	}

	return granted
}

func withGrants(held, granted []string) []string {
	for _, name := range granted {
		if !slices.Contains(held, name) {
			held = append(held, name)
		}
	}

	return held
}

func onlyOffered(names []string, active activeExtensions) []string {
	kept := make([]string, 0, len(names))
	for _, name := range names {
		if active.offers(name) {
			kept = append(kept, name)
		}
	}

	return kept
}

func touchesExtensions(names []string) bool {
	return slices.ContainsFunc(names, func(name string) bool {
		_, isExtension := agentextension.ExtensionForTool(name)
		return isExtension
	})
}

func (s *Service) holdsFor(ctx context.Context, req *serviceports.RunRequest, name string) bool {
	if s.holds(req.Definition, name) {
		return true
	}
	if _, isExtension := agentextension.ExtensionForTool(name); !isExtension {
		return false
	}

	return slices.Contains(s.activeExtensions(ctx, req.Actor).grants(), name)
}

func (s *Service) extensionRefusal(
	ctx context.Context,
	req *serviceports.RunRequest,
	name string,
) (toolOutcome, bool) {
	if _, isExtension := agentextension.ExtensionForTool(name); !isExtension {
		return toolOutcome{}, false
	}
	if s.activeExtensions(ctx, req.Actor).offers(name) {
		return toolOutcome{}, false
	}

	return failedOutcome("Tool %q comes with an extension this organization has turned off "+
		"or not set up, so it was not run. Answer without it, and say the web could not be "+
		"checked.", name), true
}

func ReadsExternalContent(name string) bool {
	typ, isExtension := agentextension.ExtensionForTool(name)
	if !isExtension {
		return false
	}
	spec, ok := agentextension.SpecFor(typ)

	return ok && spec.ReturnsExternalContent
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
