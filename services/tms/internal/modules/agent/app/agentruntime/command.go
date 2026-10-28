package agentruntime

import (
	"context"
	"slices"

	"github.com/emoss08/trenova/internal/modules/agent/domain/agentdefinition"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentextension"
	"github.com/emoss08/trenova/internal/modules/agent/ports/assistantports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/llmports"
	"github.com/emoss08/trenova/internal/modules/identity/ports/permissionports"
	"github.com/emoss08/trenova/internal/modules/kernel/ports/kernelports"
)

// commandHold is how a turn opened with an extension command is held to it:
// the directive the model is sent in place of the command and the correction a
// reply that ended without the extension is sent back with. Every command in
// agentextension.AllCommands has one.
type commandHold struct {
	directive   string
	correction  string
	retryReason string
}

var commandHolds = map[agentextension.Command]commandHold{
	agentextension.CommandBrowser: {
		directive: "The person used /browser: they want this done in the browser, on the " +
			"organization's portal sites. Work it with the portal tools you have: " +
			"web_list_portal_sites to find the site that fits, web_read_portal to look something up " +
			"on it, and web_use_portal to change something on it, which a person approves " +
			"before it runs. Do not answer from Trenova's own records or from memory in place " +
			"of the browser. When no site fits, say so and name the sites there are.\n\n" +
			"What to do in the browser: ",
		correction: "You answered without the browser, but the person used /browser to have " +
			"this done there. Call web_list_portal_sites to find the site that fits, then " +
			"web_read_portal or web_use_portal on it, and answer from what the portal says. When " +
			"no site fits, say so and name the sites there are.",
		retryReason: "The reply did not use the browser the person asked for, so it is being " +
			"worked again in the browser.",
	},
	agentextension.CommandSearch: {
		directive: "The person used /search: they want this answered from the public web. " +
			"Search with web_search, read the pages that matter with web_read, and answer " +
			"from what they say, linking to each page you use. Do not answer from Trenova's " +
			"own records or from memory in place of the web. The query leaves Trenova, so " +
			"keep record IDs and private details about a customer, carrier or driver out of " +
			"it. When the web has nothing on it, say so.\n\nWhat to search the web for: ",
		correction: "You answered without searching the web, but the person used /search to " +
			"have this answered from it. Call web_search, read what fits with web_read, and " +
			"answer from what the pages say, linking to each one you use. When the web has " +
			"nothing on it, say so.",
		retryReason: "The reply did not search the web the person asked for, so it is being " +
			"worked again with a web search.",
	},
	agentextension.CommandWatch: {
		directive: "The person used /watch: they want to be told when something new about " +
			"this appears on the web. Set a web watch with web_watch: the purpose in their " +
			"words, and a search written as a description of the pages wanted. Set how often " +
			"and for how long only when they said; otherwise leave the defaults. Use " +
			"web_search first only when you need it to write a good search. To change or end " +
			"a watch already running, find it with web_list_watches and stop it with " +
			"web_stop_watch. Do not answer from memory in place of a watch. The search leaves " +
			"Trenova, so keep record IDs and private details about a customer, carrier or " +
			"driver out of it.\n\nWhat to keep watching the web for: ",
		correction: "You answered without setting a web watch, but the person used /watch to " +
			"be told when something new appears. Call web_watch with what they want to hear " +
			"about and a search for it, or web_list_watches and web_stop_watch when they asked " +
			"about a watch already running.",
		retryReason: "The reply did not set the web watch the person asked for, so it is being " +
			"worked again with one.",
	},
}

// AvailableCommands are the extension commands the person can send this agent:
// the capability is on, the agent holds a tool the command requires (or the
// extension gives it to every agent), and the person may use it. It is decided
// from the same held tools, grants and permission checks a turn's tool set is
// built from, reading the organization's extensions once.
func (s *Service) AvailableCommands(
	ctx context.Context,
	actor *permissionports.RequestActor,
	definition *agentdefinition.Definition,
) []agentextension.Command {
	all := agentextension.AllCommands()
	available := make([]agentextension.Command, 0, len(all))
	active := s.activeExtensions(ctx, actor)
	if len(active) == 0 {
		return available
	}

	held := withGrants(s.heldTools(definition), active.Grants())
	usable := s.permittedWith(ctx, actor, commandToolsIn(held), &active)
	for _, command := range all {
		spec, _ := agentextension.CommandSpecFor(command)
		if spec.HonouredBy(usable) {
			available = append(available, command)
		}
	}

	return available
}

func commandToolsIn(names []string) []string {
	tools := make([]string, 0, len(names))
	for _, command := range agentextension.AllCommands() {
		spec, _ := agentextension.CommandSpecFor(command)
		for _, name := range spec.ToolsIn(names) {
			if !slices.Contains(tools, name) {
				tools = append(tools, name)
			}
		}
	}

	return tools
}

// pinCommand loads the command's tools the person may use, even on a turn
// whose tools are narrowed. It reports false when none of them honours the
// command, and the turn is then answered as asked.
func (s *Service) pinCommand(set *toolSet, command agentextension.Command) bool {
	spec, ok := agentextension.CommandSpecFor(command)
	if !ok {
		return false
	}
	tools := spec.ToolsIn(set.allowed)
	if !spec.HonouredBy(tools) {
		return false
	}
	for _, name := range tools {
		s.load(set, name)
	}

	return true
}

func commandInput(command agentextension.Command, input string) string {
	_, task, _ := agentextension.ParseCommand(input)

	return commandHolds[command].directive + task
}

func (t *Turn) honoured(spec agentextension.CommandSpec) bool {
	for idx := range t.result.Messages {
		for _, call := range t.result.Messages[idx].ToolCalls {
			if spec.HonouredByCall(call.Name) {
				return true
			}
		}
	}

	return false
}

// holdToCommand sends a reply that would end a command's turn without calling
// any tool that honours it back once, with the command's correction. A second
// such reply stands, so a request the extension cannot serve still ends in an
// answer.
func (s *Service) holdToCommand(
	t *Turn,
	fx TurnEffects,
	completion *llmports.ChatCompletionResult,
	held *bool,
) bool {
	if t.command == "" || *held || completion == nil {
		return false
	}
	spec, specified := agentextension.CommandSpecFor(t.command)
	hold, holds := commandHolds[t.command]
	if !specified || !holds || t.honoured(spec) {
		return false
	}

	*held = true
	fx.Emit(kernelports.StreamEvent{
		Event: assistantports.AssistantEventRetrying,
		Data: assistantports.AssistantRetryingEvent{
			Attempt: 1,
			Reason:  hold.retryReason,
			Kind:    llmports.RetryKindRestart,
		},
	})
	t.messages = append(t.messages,
		llmports.Message{
			Role:          llmports.RoleAssistant,
			Content:       completion.Text,
			ReplyID:       completion.ProviderMessageID,
			ReplyProvider: completion.ProviderID,
		},
		llmports.NoticeMessage(hold.correction),
	)

	return true
}
