package templateengine

import (
	"html/template"
	"sort"
	"strings"

	"github.com/emoss08/trenova/shared/fileutils"
	"github.com/emoss08/trenova/shared/money"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/emoss08/trenova/shared/timeutils"
	"github.com/shopspring/decimal"
)

// Every function here delegates to an existing shared helper, so a date on an
// invoice PDF, in the email that delivers it, and in the notice that preceded it
// are formatted by the same code. Adding a formatter here without adding it to
// shared/ is how three renderings of the same timestamp start to disagree.
//
// The set is deliberately closed. There is no raw, safeHTML, attr, url, js, env,
// exec, or include, and no user-supplied printf format. Contextual auto-escaping
// is the property that makes an organization-authored template safe to render,
// and a single escape hatch would give it away. forbiddenFunctionNames below is
// asserted against the live map by a test.
var forbiddenFunctionNames = []string{
	"raw", "safeHTML", "safehtml", "html", "attr", "url", "urlquery",
	"js", "jsstr", "env", "exec", "include", "readFile", "template",
}

// FuncMap returns the functions available to HTML channels.
func FuncMap() template.FuncMap {
	return template.FuncMap{
		// Money. Currency leads the amount because a bare number on a
		// customer-facing document invites an argument about which currency.
		"money":       money.FormatDecimal,
		"moneyString": money.FormatDecimalString,
		"amount":      formatDecimalPlain,

		// Time. The tz-aware variants exist because a delivery at 19:00 Pacific
		// is the next day in UTC, and an invoice dated a day late is an invoice a
		// customer can refuse.
		"date":     timeutils.FormatUnixDateIn,
		"datetime": timeutils.FormatUnixDateTimeIn,
		"stamp":    timeutils.FormatStampIn,
		"minutes":  timeutils.FormatMinutes,
		"duration": timeutils.FormatLongDurationMs,

		// Text. default and truncate take their subject last so they read the way
		// an author expects in a pipeline: {{ .Memo | default "none" }} and
		// {{ .Memo | truncate 40 }}. Go passes the piped value as the final
		// argument, so the shared helpers are wrapped rather than bound directly.
		"upper":    strings.ToUpper,
		"lower":    strings.ToLower,
		"title":    stringutils.CapitalizeFirst,
		"trim":     strings.TrimSpace,
		"default":  withDefault,
		"nonEmpty": stringutils.FilterEmpty,
		"join":     joinNonEmpty,
		"truncate": truncate,

		// nl2br is the only function permitted to produce template.HTML, and it
		// escapes before it inserts any markup.
		"nl2br": nl2br,

		"bytes": fileutils.HumanizeBytes,

		// Arithmetic, for table striping and running totals in a range.
		"add": func(a, b int) int { return a + b },
		"sub": func(a, b int) int { return a - b },
		"seq": seq,
		"odd": func(i int) bool { return i%2 == 1 },
	}
}

// TextFuncMap returns the functions available to plain-text channels: subjects,
// text email bodies, notification titles and messages.
//
// nl2br is excluded rather than made a no-op. Text output has no markup, and a
// function whose name promises HTML would be a trap in a channel that cannot use
// it.
func TextFuncMap() template.FuncMap {
	out := FuncMap()
	delete(out, "nl2br")
	return out
}

// nl2br escapes text and converts newlines to line breaks.
//
// The return type is template.HTML, which means the escaper will not touch the
// result — safe only because stringutils.TextToHTMLParagraph escapes the input
// before adding any markup of its own.
func nl2br(text string) template.HTML {
	//nolint:gosec // TextToHTMLBreaks escapes before inserting any markup.
	return template.HTML(stringutils.TextToHTMLBreaks(text))
}

func formatDecimalPlain(value decimal.Decimal) string {
	return value.StringFixed(2)
}

// withDefault is stringutils.WithDefault with pipeline argument order.
func withDefault(fallback, value string) string {
	return stringutils.WithDefault(value, fallback)
}

// truncate is stringutils.TruncateRunes with pipeline argument order.
func truncate(maxLength int, value string) string {
	return stringutils.TruncateRunes(value, maxLength)
}

// joinNonEmpty joins values with a separator, dropping the empties.
//
// Address lines are the reason: joining a block that has no second street line
// must not leave a dangling comma on a customer's invoice.
func joinNonEmpty(sep string, values []string) string {
	return strings.Join(stringutils.FilterEmpty(values), sep)
}

// maxSeqLength bounds seq so a template cannot ask for a billion iterations.
const maxSeqLength = 10000

// seq produces the integers [0,n) for range loops that need an index rather than
// a collection.
func seq(n int) []int {
	if n <= 0 {
		return nil
	}
	if n > maxSeqLength {
		n = maxSeqLength
	}
	out := make([]int, n)
	for i := range out {
		out[i] = i
	}
	return out
}

// allowedFunctionSet is the lookup the analyzer uses to reject calls to anything
// outside the permitted set, including the builtins a FuncMap cannot remove.
func allowedFunctionSet() map[string]struct{} {
	fm := FuncMap()
	out := make(map[string]struct{}, len(fm)+len(builtinTemplateFunctions))
	for name := range fm {
		out[name] = struct{}{}
	}
	for _, name := range builtinTemplateFunctions {
		out[name] = struct{}{}
	}
	return out
}

// AllowedFunctionNames lists every function a template may call, sorted.
//
// It backs the "available functions" hint on an undefined-call diagnostic and the
// reference panel in the editor, so the list an author sees can never drift from
// the map the engine actually installs.
func AllowedFunctionNames() []string {
	fm := FuncMap()
	out := make([]string, 0, len(fm)+len(builtinTemplateFunctions))
	for name := range fm {
		out = append(out, name)
	}
	out = append(out, builtinTemplateFunctions...)
	sort.Strings(out)
	return out
}

// builtinTemplateFunctions are the text/template builtins we permit.
//
// Absent on purpose: "html", "js", "urlquery", and "printf". The three escapers
// would let an author choose an escaping context, which is exactly the decision
// contextual auto-escaping needs to own. printf with an author-supplied format
// string is a formatting-directive injection for no benefit the other functions
// do not already cover.
var builtinTemplateFunctions = []string{
	"and", "or", "not", "eq", "ne", "lt", "le", "gt", "ge",
	"len", "index", "slice", "print", "println", "call",
}

// ForbiddenFunctionNames exposes the deny list for the test that asserts none of
// them ever appear in the live map.
func ForbiddenFunctionNames() []string {
	out := make([]string, len(forbiddenFunctionNames))
	copy(out, forbiddenFunctionNames)
	return out
}
