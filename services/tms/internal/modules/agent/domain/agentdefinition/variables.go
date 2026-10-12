package agentdefinition

import (
	"regexp"
	"strings"
	"time"

	"github.com/emoss08/trenova/shared/cronutils"
)

var instructionVariablePattern = regexp.MustCompile(`\{\{[^{}]*[^{}\s][^{}]*\}\}`)

func InstructionVariables(text string) []string {
	matches := instructionVariablePattern.FindAllString(text, -1)
	if len(matches) == 0 {
		return nil
	}

	out := make([]string, 0, len(matches))
	seen := make(map[string]struct{}, len(matches))
	for _, match := range matches {
		if _, ok := seen[match]; ok {
			continue
		}
		seen[match] = struct{}{}
		out = append(out, match)
	}

	return out
}

func ChangedInstructionVariables(original, rewritten string) (missing, added []string) {
	before := InstructionVariables(original)
	after := InstructionVariables(rewritten)

	kept := make(map[string]struct{}, len(after))
	for _, variable := range after {
		kept[variable] = struct{}{}
	}
	had := make(map[string]struct{}, len(before))
	for _, variable := range before {
		had[variable] = struct{}{}
		if _, ok := kept[variable]; !ok {
			missing = append(missing, variable)
		}
	}
	for _, variable := range after {
		if _, ok := had[variable]; !ok {
			added = append(added, variable)
		}
	}

	return missing, added
}

func CronExpressionValid(expression string) bool {
	trimmed := strings.TrimSpace(expression)

	return trimmed != "" && len(trimmed) <= MaxCronLength && cronutils.Validate(trimmed) == nil
}

func CronTimezoneValid(timezone string) bool {
	trimmed := strings.TrimSpace(timezone)
	if trimmed == "" {
		return false
	}
	_, err := time.LoadLocation(trimmed)

	return err == nil
}
