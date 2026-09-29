package agentruntime

import (
	"fmt"
	"slices"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/shared/jsonutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxFilingEchoBytes    = 2048
	maxFilingEchoValue    = 240
	filedArgumentsLead    = "Filed arguments: "
	filingEchoCutNote     = "(cut short; the card holds the whole call)"
	filingUnreadableValue = "a value that could not be repeated"
	filingPreviewLead     = "The card shows the person what this proposal would do. Say " +
		"only what it and this result hold; anything else is not part of the proposal."
	filingArgumentsLead = "The proposal holds exactly these arguments. Say only what " +
		"they hold; anything else is not part of the proposal."
)

func filingEcho(baseline *serviceports.ProposalBaselineResult, args map[string]any) string {
	if baseline != nil && baseline.Preview != nil &&
		strings.TrimSpace(baseline.Preview.Summary) != "" {
		var b strings.Builder
		b.WriteString("\n")
		b.WriteString(filingPreviewLead)
		b.WriteString("\nSummary: ")
		b.WriteString(strings.TrimSpace(baseline.Preview.Summary))
		for idx := range baseline.Preview.Warnings {
			for _, line := range baseline.Preview.Warnings[idx].ReasonLines() {
				b.WriteString("\nWarning: ")
				b.WriteString(line)
			}
		}

		return b.String()
	}

	return "\n" + filingArgumentsLead + "\n" + filedArgumentsLead + boundedArguments(args)
}

func boundedArguments(args map[string]any) string {
	encoded, err := jsonutils.CanonicalMarshal(args)
	if err == nil && len(encoded) <= maxFilingEchoBytes {
		return string(encoded)
	}

	keys := make([]string, 0, len(args))
	for key := range args {
		keys = append(keys, key)
	}
	slices.Sort(keys)

	var b strings.Builder
	for _, key := range keys {
		line := "\n- " + key + ": " + describeFiledValue(args[key])
		if b.Len()+len(line) > maxFilingEchoBytes {
			break
		}
		b.WriteString(line)
	}
	b.WriteString("\n")
	b.WriteString(filingEchoCutNote)

	return b.String()
}

func describeFiledValue(value any) string {
	switch typed := value.(type) {
	case []any:
		return fmt.Sprintf("%d %s", len(typed), stringutils.Pluralize("item", "items", len(typed)))
	case map[string]any:
		encoded, err := jsonutils.CanonicalMarshal(typed)
		if err == nil && len(encoded) <= maxFilingEchoValue {
			return string(encoded)
		}

		return fmt.Sprintf("an object of %d %s", len(typed),
			stringutils.Pluralize("field", "fields", len(typed)))
	case string:
		if len(typed) > maxFilingEchoValue {
			return fmt.Sprintf("%d characters of text", len([]rune(typed)))
		}
	}

	encoded, err := jsonutils.CanonicalMarshal(value)
	if err != nil {
		return filingUnreadableValue
	}

	return string(encoded)
}
