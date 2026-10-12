package agentguard

import (
	"regexp"
	"strings"
)

// maxRequestChars bounds what is accepted before classification. A request far
// beyond this is either a paste of something that belongs in a document or an
// attempt to bury an instruction deep in filler.
const maxRequestChars = 8000

// rule is one deterministic pattern.
type rule struct {
	name     string
	pattern  *regexp.Regexp
	reason   Reason
	category Category
}

// The deterministic layer is deliberately narrow.
//
// Freight vocabulary and programming vocabulary overlap badly: a dispatcher
// legitimately asks about a route, a load, a container, a terminal, a freight
// class, a package, a driver, a broker, a hub, and a schedule. Every one of those
// words also means something in software. Matching on them would break the
// product for the people it is built for, and a refusal that fires on "show me
// the route for load 12345" is far more damaging than letting one code request
// through to the classifier behind it.
//
// So these patterns match only constructs that carry no freight reading at all —
// actual source syntax, or an explicit request for code naming a programming
// language. Everything ambiguous is left to the classifier.
var rules = []rule{
	{
		name: "code_fence_with_language",
		pattern: regexp.MustCompile(
			"(?i)```[ \\t]*(python|py|javascript|js|typescript|ts|go|golang|java|c\\+\\+|cpp|csharp|c#|ruby|rust|php|bash|sh|shell|sql|powershell|perl|swift|kotlin|scala|r|matlab|html|css)\\b",
		),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name: "function_definition_syntax",
		pattern: regexp.MustCompile(
			`(?m)^\s*(def\s+\w+\s*\(|function\s+\w+\s*\(|public\s+static\s+void\s+main|class\s+\w+\s*[:({]|func\s+\w+\s*\()`,
		),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name:     "shebang_or_markup",
		pattern:  regexp.MustCompile(`(?m)(^#!\s*/|<\?php|<!DOCTYPE\s+html|</html>|</script>)`),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name: "explicit_code_request",
		// Requires both an authoring verb and a named language, so "write up the
		// route plan" and "explain freight class 70" both pass.
		pattern: regexp.MustCompile(
			`(?i)\b(write|generate|create|produce|give me|show me|build|code|implement|refactor|debug|fix)\b[^.?!]{0,60}\b(python|javascript|typescript|golang|java|c\+\+|c#|csharp|ruby|rust|php|bash|shell script|powershell|sql query|regex|regular expression)\b`,
		),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name: "code_artifact_request",
		// "program" is deliberately absent: a safety program, a driver training
		// program, and a compliance program are all ordinary freight requests, and
		// the classifier is a better judge of "a program that calls your API" than
		// a pattern that would also refuse "build a safety program".
		pattern: regexp.MustCompile(
			`(?i)\b(write|generate|create|produce|give me|show me|build|make)\b[^.?!]{0,40}\b(a |an |some )?(script|source code|code snippet|unit test|shell command|terminal command|cli command)\b`,
		),
		reason:   ReasonCodeGeneration,
		category: CategoryCodeGeneration,
	},
	{
		name: "instruction_override",
		pattern: regexp.MustCompile(
			`(?i)\b(ignore|disregard|forget|override|bypass)\b[^.?!]{0,40}\b(previous|prior|above|earlier|initial|original|all)\b[^.?!]{0,20}\b(instruction|instructions|prompt|prompts|rule|rules|direction|directions)\b`,
		),
		reason:   ReasonPromptManipulation,
		category: CategoryPromptManipulation,
	},
	{
		name: "prompt_extraction",
		pattern: regexp.MustCompile(
			`(?i)\b(what|show|reveal|repeat|print|output|tell me)\b[^.?!]{0,40}\byour\b[^.?!]{0,30}\b(system prompt|initial prompt|instructions|system message|configuration prompt)\b`,
		),
		reason:   ReasonPromptManipulation,
		category: CategoryPromptManipulation,
	},
	{
		name: "persona_override",
		pattern: regexp.MustCompile(
			`(?i)\b(you are now|from now on you are|act as (a |an )?(?:general|unrestricted|uncensored)|pretend (that )?you are (a |an )?(?:general|unrestricted|uncensored)|developer mode|jailbreak|DAN mode)\b`,
		),
		reason:   ReasonPromptManipulation,
		category: CategoryPromptManipulation,
	},
}

// EvaluateDeterministic runs the pattern layer. An empty Decision with Allowed
// true means nothing fired and the request should continue to the classifier; it
// is not by itself a judgement that the request is in scope.
func EvaluateDeterministic(input string) Decision {
	trimmed := strings.TrimSpace(input)

	if len(trimmed) > maxRequestChars {
		return refused(StageDeterministic, ReasonOversized, CategoryOther, "max_length")
	}

	for _, r := range rules {
		if r.pattern.MatchString(trimmed) {
			return refused(StageDeterministic, r.reason, r.category, r.name)
		}
	}

	return Decision{Allowed: true, Stage: StageDeterministic}
}
