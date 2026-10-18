package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

const changeBrowserGuard = "agent-loop-browser-guard"

const (
	browserDirective = "The person used /browser: they want this done in the browser, on the " +
		"organization's portal sites. Work it with the portal tools you have: list_portal_sites " +
		"to find the site that fits, read_portal to look something up on it, and act_on_portal " +
		"to change something on it, which a person approves before it runs. Do not answer from " +
		"Trenova's own records or from memory in place of the browser. When no site fits, say " +
		"so and name the sites there are.\n\nWhat to do in the browser: "
	browserCorrection = "You answered without the browser, but the person used /browser to " +
		"have this done there. Call list_portal_sites to find the site that fits, then " +
		"read_portal or act_on_portal on it, and answer from what the portal says. When no " +
		"site fits, say so and name the sites there are."
	browserRetryReason = "The reply did not use the browser the person asked for, so it is " +
		"being worked again in the browser."
)

func (s *Service) BrowserTools(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) []string {
	active := s.activeExtensions(ctx, actor)
	if _, on := active[agentextension.CapabilityPortalAutomation]; !on {
		return nil
	}

	held := withGrants(s.heldTools(definition), active.Grants())

	return s.permittedWith(ctx, actor, browserToolsIn(held), &active)
}

func browserToolsIn(names []string) []string {
	spec, _ := agentextension.CapabilitySpecFor(agentextension.CapabilityPortalAutomation)
	tools := make([]string, 0, len(spec.Tools))
	for _, name := range spec.Tools {
		if slices.Contains(names, name) {
			tools = append(tools, name)
		}
	}

	return tools
}

func (s *Service) pinBrowser(set *toolSet) bool {
	tools := browserToolsIn(set.allowed)
	if !agentextension.WorksInBrowser(tools) {
		return false
	}
	for _, name := range tools {
		s.load(set, name)
	}

	return true
}

func browserInput(input string) string {
	task, _ := agentextension.BrowserTask(input)

	return browserDirective + task
}

func (t *Turn) usedBrowser() bool {
	for idx := range t.result.Messages {
		for _, call := range t.result.Messages[idx].ToolCalls {
			if capability, ok := agentextension.CapabilityForTool(call.Name); ok &&
				capability == agentextension.CapabilityPortalAutomation {
				return true
			}
		}
	}

	return false
}

func (s *Service) holdToBrowser(
	t *Turn,
	fx TurnEffects,
	completion *agentports.ChatCompletionResult,
	held *bool,
) bool {
	if !t.browser || *held || completion == nil || t.usedBrowser() ||
		!fx.Supports(changeBrowserGuard) {
		return false
	}

	*held = true
	fx.Emit(kernelports.StreamEvent{
		Event: agentports.AssistantEventRetrying,
		Data: agentports.AssistantRetryingEvent{
			Attempt: 1,
			Reason:  browserRetryReason,
			Kind:    agentports.RetryKindRestart,
		},
	})
	t.messages = append(t.messages,
		agentports.Message{
			Role:          agentports.RoleAssistant,
			Content:       completion.Text,
			ReplyID:       completion.ProviderMessageID,
			ReplyProvider: completion.ProviderID,
		},
		agentports.Message{Role: agentports.RoleUser, Content: browserCorrection},
	)

	return true
}
