package agentguard

import (
	"regexp"
	"strings"
)

// CodeBlockOmitted stands in a reply where the output guard took code out.
const CodeBlockOmitted = "[A code block was left out: this assistant handles " +
	"transportation work, not software.]"

const (
	ruleCodeFence    = "output_code_fence_with_language"
	ruleFunction     = "output_function_syntax"
	ruleMarkup       = "output_markup"
	fenceMarker      = "```"
	closingBraceLine = "}"
)

var (
	// fencedLanguage is a fence opened with a programming-language tag. A
	// fenced block with no tag is usually a manifest or an address rather
	// than a program, and is left alone unless what it holds is code.
	fencedLanguage = regexp.MustCompile(
		"(?i)^[ \\t]*```[ \\t]*(python|py|javascript|js|typescript|ts|go|golang|java|c\\+\\+|cpp|csharp|c#|ruby|rust|php|bash|sh|shell|sql|powershell|perl|swift|kotlin|scala|html|css)\\b",
	)
	functionSyntax = regexp.MustCompile(
		`^\s*(def\s+\w+\s*\(|function\s+\w+\s*\(|public\s+static\s+void\s+main|func\s+\w+\s*\([^)]*\)\s*\w*\s*{)`,
	)
	markup = regexp.MustCompile(`(^#!\s*/|<\?php|<!DOCTYPE\s+html|</html>|</script>)`)
)

// EvaluateOutput checks a generated reply before it reaches the person who
// asked, and returns the reply as it may be shown.
//
// Reaching this layer means something upstream let a code request through, so
// a hit is a signal that the scope guard or the system prompt needs attention,
// and the decision says the reply was altered and by which rule. Only the code
// is taken out: a fenced block with a programming-language tag, a fenced block
// holding source or markup, and a function or markup line outside a fence,
// with the body that follows it. Each stands replaced by CodeBlockOmitted. A
// reply that answered a dispatcher's question and added a snippet used to be
// refused whole, and the answer went with the snippet.
func EvaluateOutput(output string) (string, Decision) {
	lines := strings.Split(output, "\n")
	kept := make([]string, 0, len(lines))
	matched := ""
	omit := func(rule string) {
		if matched == "" {
			matched = rule
		}
		if len(kept) == 0 || kept[len(kept)-1] != CodeBlockOmitted {
			kept = append(kept, CodeBlockOmitted)
		}
	}

	for idx := 0; idx < len(lines); idx++ {
		line := lines[idx]
		if strings.HasPrefix(strings.TrimSpace(line), fenceMarker) {
			end := closingFence(lines, idx)
			if rule := fenceRule(lines, idx, end); rule != "" {
				omit(rule)
				idx = end
				continue
			}
			kept = append(kept, lines[idx:end+1]...)
			idx = end
			continue
		}
		if rule := lineRule(line); rule != "" {
			omit(rule)
			idx = bodyEnd(lines, idx)
			continue
		}
		kept = append(kept, line)
	}

	if matched == "" {
		return output, Decision{Allowed: true, Stage: StageOutput}
	}

	return strings.Join(kept, "\n"), Decision{
		Allowed:     true,
		Altered:     true,
		Stage:       StageOutput,
		Reason:      ReasonCodeGeneration,
		Category:    CategoryCodeGeneration,
		MatchedRule: matched,
	}
}

// closingFence is the line that closes the fence opened at start, or the last
// line when a reply cut off never closed it.
func closingFence(lines []string, start int) int {
	for idx := start + 1; idx < len(lines); idx++ {
		if strings.HasPrefix(strings.TrimSpace(lines[idx]), fenceMarker) {
			return idx
		}
	}

	return len(lines) - 1
}

func fenceRule(lines []string, start, end int) string {
	if fencedLanguage.MatchString(lines[start]) {
		return ruleCodeFence
	}
	for idx := start + 1; idx < end; idx++ {
		if rule := lineRule(lines[idx]); rule != "" {
			return rule
		}
	}

	return ""
}

func lineRule(line string) string {
	switch {
	case functionSyntax.MatchString(line):
		return ruleFunction
	case markup.MatchString(line):
		return ruleMarkup
	default:
		return ""
	}
}

// bodyEnd is the last line of the code a function line opens: the lines
// indented under it, and the brace that closes it at its own depth.
func bodyEnd(lines []string, start int) int {
	depth := indentOf(lines[start])
	end := start
	for idx := start + 1; idx < len(lines); idx++ {
		line := lines[idx]
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			break
		}
		if indentOf(line) > depth {
			end = idx
			continue
		}
		if trimmed == closingBraceLine && indentOf(line) == depth {
			end = idx
		}

		break
	}

	return end
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}
