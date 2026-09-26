package writecoverage

import (
	"fmt"
	"slices"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const toolList = "docs/engineering/ai-tool-safety.md"

func Check(
	writes []Write,
	mapping Mapping,
	tools map[string]serviceports.ToolPolicy,
) []string {
	problems := make([]string, 0)
	known := make(map[string]struct{}, len(writes))
	twins := make(map[string]*Write)

	for idx := range writes {
		write := &writes[idx]
		known[write.Key] = struct{}{}
		for _, twin := range write.Twins {
			for _, route := range twin.Routes {
				twins[route.String()] = write
			}
		}

		decision, ok := mapping.Writes[write.Key]
		if !ok {
			problems = append(problems, missingEntry(write))
			continue
		}
		problems = append(problems, checkDecision(write.Key, decision, tools)...)
	}

	for key := range mapping.Writes {
		if _, ok := known[key]; ok {
			continue
		}
		if owner, merged := twins[key]; merged {
			problems = append(problems, fmt.Sprintf(
				"%q is now merged into %q as its REST twin (both reach %s): remove its "+
					"entry from %s; the mutation's entry covers both",
				key, owner.Key, strings.Join(owner.Calls, ", "), MappingDisplayPath,
			))
			continue
		}
		problems = append(problems, fmt.Sprintf(
			"%q in %s names a write the app no longer exposes: remove the entry, or rename "+
				"it to the write's new key if it moved",
			key, MappingDisplayPath,
		))
	}

	slices.Sort(problems)

	return problems
}

func missingEntry(write *Write) string {
	return fmt.Sprintf(
		"%q (%s, handled by %s) has no entry in %s: add it under writes: with "+
			"tools: [<agent tool that performs it>], or exempt: <category> with a reason: "+
			"saying why no agent should, or pending: <what the missing tool would do>",
		write.Key, write.Domain, handlerOrUnknown(write.Handler), MappingDisplayPath,
	)
}

func checkDecision(
	key string,
	decision Decision,
	tools map[string]serviceports.ToolPolicy,
) []string {
	problems := make([]string, 0)
	fail := func(format string, args ...any) {
		problems = append(problems, fmt.Sprintf("%q in %s: ", key, MappingDisplayPath)+
			fmt.Sprintf(format, args...))
	}

	set := 0
	if len(decision.Tools) > 0 {
		set++
	}
	if decision.Exempt != "" {
		set++
	}
	if strings.TrimSpace(decision.Pending) != "" {
		set++
	}
	switch {
	case set == 0:
		fail("give it tools:, exempt: with a reason:, or pending: with what the tool would do")
	case set > 1:
		fail("set exactly one of tools:, exempt: or pending:")
	}

	if decision.Exempt != "" {
		if _, ok := categoryInfo(decision.Exempt); !ok {
			fail("exempt: %q is not a category; use one of %s",
				decision.Exempt, categoryNames())
		}
		if strings.TrimSpace(decision.Reason) == "" {
			fail("exempt: %s needs a reason: saying why no agent should perform this write",
				decision.Exempt)
		}
	} else if decision.Reason != "" {
		fail("reason: belongs to an exemption; put a pending write's description in pending:")
	}

	seen := make(map[string]struct{}, len(decision.Tools))
	for _, name := range decision.Tools {
		if _, dup := seen[name]; dup {
			fail("tool %q is listed twice", name)
		}
		seen[name] = struct{}{}
		if _, ok := tools[name]; !ok {
			fail("tool %q is not a registered agent tool: fix the name (every tool is "+
				"listed in %s) or mark the write pending: until the tool exists", name, toolList)
		}
	}

	return problems
}

func categoryNames() string {
	names := make([]string, 0, len(categories))
	for _, info := range categories {
		names = append(names, string(info.Category))
	}

	return strings.Join(names, ", ")
}

func handlerOrUnknown(handler string) string {
	if handler == "" {
		return "no resolver found"
	}

	return handler
}
