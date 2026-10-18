package agentextension

import (
	"slices"
	"strings"
	"unicode"
)

type Command string

const (
	CommandBrowser = Command("browser")
	CommandSearch  = Command("search")
)

func AllCommands() []Command {
	return []Command{CommandBrowser, CommandSearch}
}

func (c Command) String() string { return string(c) }

func (c Command) Text() string { return "/" + string(c) }

func (c Command) IsValid() bool {
	_, ok := commands[c]
	return ok
}

type CommandSpec struct {
	Capability        Capability
	Requires          []string
	MissingTaskReason string
	DirectedReason    string
	UnavailableReason string
}

func (s CommandSpec) HonouredBy(tools []string) bool {
	return slices.ContainsFunc(s.Requires, func(name string) bool {
		return slices.Contains(tools, name)
	})
}

func (s CommandSpec) ToolsIn(names []string) []string {
	spec := capabilities[s.Capability]
	tools := make([]string, 0, len(spec.Tools))
	for _, name := range spec.Tools {
		if slices.Contains(names, name) {
			tools = append(tools, name)
		}
	}

	return tools
}

var commands = map[Command]CommandSpec{
	CommandBrowser: {
		Capability:        CapabilityPortalAutomation,
		Requires:          []string{ToolReadPortal, ToolActOnPortal},
		MissingTaskReason: "Say what to do in the browser after /browser",
		DirectedReason: "Work in the browser is done by this conversation's agent; " +
			"send it without naming another",
		UnavailableReason: "{0} cannot work in the browser. Portal automation has to be turned " +
			"on, the agent has to hold its portal tools, and you need permission to use them.",
	},
	CommandSearch: {
		Capability:        CapabilityWebResearch,
		Requires:          []string{ToolWebSearch},
		MissingTaskReason: "Say what to search the web for after /search",
		DirectedReason: "A web search is done by this conversation's agent; " +
			"send it without naming another",
		UnavailableReason: "{0} cannot search the web. Web research has to be turned on, the " +
			"agent has to hold its web tools, and you need permission to use them.",
	},
}

func CommandSpecFor(c Command) (CommandSpec, bool) {
	spec, ok := commands[c]
	return spec, ok
}

func ParseCommand(content string) (Command, string, bool) {
	trimmed := strings.TrimSpace(content)
	body, slashed := strings.CutPrefix(trimmed, "/")
	if !slashed {
		return "", "", false
	}

	name, rest := body, ""
	if end := strings.IndexFunc(body, unicode.IsSpace); end >= 0 {
		name, rest = body[:end], body[end:]
	}
	command := Command(strings.ToLower(name))
	if !command.IsValid() {
		return "", "", false
	}

	return command, strings.TrimSpace(rest), true
}
