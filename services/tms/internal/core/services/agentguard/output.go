package agentguard

import "regexp"

// outputRules catch source code in a reply. They are narrower than the input
// rules: the assistant legitimately quotes identifiers, reference numbers, and
// tabular data, and a fenced block with no language tag is usually a manifest or
// an address rather than a program. Only a programming-language tag or
// unmistakable source syntax counts.
var outputRules = []rule{
	{
		name:     "output_code_fence_with_language",
		pattern:  regexp.MustCompile("(?i)```[ \\t]*(python|py|javascript|js|typescript|ts|go|golang|java|c\\+\\+|cpp|csharp|c#|ruby|rust|php|bash|sh|shell|sql|powershell|perl|swift|kotlin|scala|html|css)\\b"),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name:     "output_function_syntax",
		pattern:  regexp.MustCompile(`(?m)^\s*(def\s+\w+\s*\(|function\s+\w+\s*\(|public\s+static\s+void\s+main|func\s+\w+\s*\([^)]*\)\s*\w*\s*{)`),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name:     "output_markup",
		pattern:  regexp.MustCompile(`(?m)(^#!\s*/|<\?php|<!DOCTYPE\s+html|</html>|</script>)`),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
}

// EvaluateOutput checks a generated reply before it reaches the person who
// asked.
//
// Reaching this layer means something upstream let a code request through, so a
// hit is worth logging as a signal that the scope guard or the system prompt
// needs attention. The whole turn is refused rather than the offending block
// stripped, because redacting would hide exactly that signal while still leaving
// a half-answer that looks like the assistant tried.
func EvaluateOutput(output string) Decision {
	for _, r := range outputRules {
		if r.pattern.MatchString(output) {
			return refused(StageOutput, r.reason, r.category, r.name)
		}
	}

	return Decision{Allowed: true, Stage: StageOutput}
}
