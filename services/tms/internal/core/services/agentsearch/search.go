// Package agentsearch is how the agent-facing indexes read a person's words:
// the tool catalog that picks which tools a turn gets, and the product guide
// that says where things are in Trenova. Both have to bridge the same gap —
// an operator says driver and truck where the product says worker and tractor
// — so they share one vocabulary rather than each keeping a copy that drifts.
package agentsearch

import "strings"

// vocabulary maps what an operator says to what the schema calls it.
//
// A dispatcher says driver, the table says worker; they say truck, the table
// says tractor. Leaving that gap for the model to bridge is what failed in
// production: asked which drivers had a credential expiring, it searched for the
// literal word "driver", matched nobody, and reported the fleet as empty.
var vocabulary = map[string][]string{
	"deposit":        {"bank", "receipt"},
	"deposits":       {"bank", "receipt"},
	"lockbox":        {"bank", "receipt"},
	"remittance":     {"payment", "receipt"},
	"remit":          {"payment", "receipt"},
	"reconcile":      {"bank", "receipt"},
	"reconciliation": {"bank", "receipt", "work"},
	"cash":           {"payment", "receipt"},
	"check":          {"payment"},
	"ach":            {"payment"},
	"wire":           {"payment"},
	"driver":         {"worker"},
	"drivers":        {"worker"},
	"employee":       {"worker"},
	"roster":         {"worker"},
	"truck":          {"tractor"},
	"trucks":         {"tractor"},
	"power":          {"tractor"},
	"unit":           {"tractor"},
	"rig":            {"tractor"},
	"reefer":         {"trailer"},
	"van":            {"trailer"},
	"load":           {"shipment"},
	"loads":          {"shipment"},
	"order":          {"shipment"},
	"orders":         {"shipment"},
	"freight":        {"shipment"},
	"pro":            {"shipment"},
	"shipper":        {"customer"},
	"consignee":      {"customer"},
	"account":        {"customer"},
	"broker":         {"customer"},
	"facility":       {"location"},
	"terminal":       {"location"},
	"yard":           {"location"},
	"warehouse":      {"location"},
	"hazmat":         {"credential", "endorsement"},
	"endorsement":    {"credential"},
	"medical":        {"credential"},
	"card":           {"credential"},
	"licence":        {"credential"},
	"license":        {"credential"},
	"cdl":            {"credential"},
	"certificate":    {"credential"},
	"qualified":      {"credential"},
	"compliance":     {"credential"},
	"invoice":        {"billing", "invoice"},
	"bill":           {"billing"},
	"billed":         {"billing"},
	"receivable":     {"billing"},
	"ar":             {"billing"},
	"charge":         {"billing"},
	"rate":           {"billing"},
	"expiring":       {"expiring", "credential"},
	"expired":        {"expiring", "credential"},
	"lapsed":         {"expiring", "credential"},
	"assign":         {"assign", "move"},
	"dispatch":       {"assign", "move"},
	"pto":            {"time", "off"},
	"vacation":       {"time", "off"},
	"leave":          {"time", "off"},
	"holiday":        {"time", "off"},
	"sick":           {"time", "off"},
	"absent":         {"time", "off"},
	"away":           {"time", "off"},
	"oos":            {"status", "service"},
	"down":           {"status", "service"},
	"shop":           {"status", "maintenance"},
	"maintenance":    {"status", "maintenance"},
	"breakdown":      {"status", "service"},
	"report":         {"report", "dataset"},
	"reports":        {"report", "dataset"},
	"dataset":        {"report", "dataset"},
	"datasets":       {"report", "dataset"},
	"query":          {"report"},
	"dashboard":      {"dashboard", "tile", "home"},
	"dashboards":     {"dashboard", "tile", "home"},
	"tile":           {"dashboard"},
	"tiles":          {"dashboard"},
	"home":           {"home", "widget"},
	"homepage":       {"home", "widget"},
	"landing":        {"home", "widget"},
	"widget":         {"home", "widget"},
	"widgets":        {"home", "widget"},
	"chart":          {"chart", "report", "dashboard"},
	"graph":          {"chart", "report", "dashboard"},
	"kpi":            {"metric", "kpi"},
	"metric":         {"metric", "kpi"},
	"metrics":        {"metric", "kpi"},
	"remember":       {"memory", "remember"},
	"memory":         {"memory", "remember", "recall"},
	"note":           {"memory", "remember"},
	"forget":         {"memory", "forget"},
	"escalate":       {"exception", "review"},
	"stuck":          {"exception", "review"},
	"email":          {"email", "message"},
	"inbox":          {"inbound", "message"},
	"view":           {"view", "table"},
	"filter":         {"view", "table"},
}

// stopWords are the words a question is built from rather than about. They are
// dropped so "which drivers" scores on "driver" alone.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "and": {}, "any": {}, "are": {}, "be": {}, "by": {},
	"can": {}, "could": {}, "did": {}, "do": {}, "does": {}, "for": {},
	"from": {}, "get": {}, "give": {}, "has": {}, "have": {}, "how": {},
	"i": {}, "in": {}, "is": {}, "it": {}, "its": {}, "me": {}, "my": {},
	"of": {}, "on": {}, "or": {}, "our": {}, "please": {}, "show": {},
	"tell": {}, "that": {}, "the": {}, "their": {}, "them": {}, "there": {},
	"these": {}, "this": {}, "those": {}, "to": {}, "us": {}, "was": {},
	"were": {}, "what": {}, "when": {}, "where": {}, "which": {}, "who": {},
	"whom": {}, "whose": {}, "why": {}, "with": {}, "would": {},
	"list": {}, "find": {}, "look": {}, "see": {}, "need": {}, "want": {},
	"tool": {}, "tools": {}, "data": {}, "record": {}, "records": {},
	"all": {}, "some": {}, "one": {}, "up": {}, "about": {}, "currently": {},
}

// Terms are the terms a query is searched on: its meaningful words, their
// singulars, and the product's words for them.
func Terms(query string) map[string]struct{} {
	return expand(Tokens(query))
}

// SplitCamel breaks an identifier at its capitals, so "customerId" is
// searchable as "customer Id".
func SplitCamel(name string) string {
	var b strings.Builder
	for i, r := range name {
		if i > 0 && 'A' <= r && r <= 'Z' {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}

	return b.String()
}

// expand adds the schema's word for each of the operator's words, keeping the
// original so a question phrased in schema terms still matches.
func expand(raw []string) map[string]struct{} {
	terms := make(map[string]struct{}, len(raw)*2)
	for _, token := range raw {
		terms[token] = struct{}{}
		for _, synonym := range vocabulary[token] {
			terms[synonym] = struct{}{}
		}
		if singular := Singularize(token); singular != token {
			terms[singular] = struct{}{}
			for _, synonym := range vocabulary[singular] {
				terms[synonym] = struct{}{}
			}
		}
	}

	return terms
}

// Tokens are the words of a text that carry meaning, lowercased, with the
// words a question is built from dropped.
func Tokens(text string) []string {
	fields := strings.FieldsFunc(strings.ToLower(text), func(r rune) bool {
		return !('a' <= r && r <= 'z') && !('0' <= r && r <= '9')
	})

	out := make([]string, 0, len(fields))
	for _, field := range fields {
		if _, stop := stopWords[field]; stop {
			continue
		}
		out = append(out, field)
	}

	return out
}

// TokenSet is the searchable terms of a text, plurals folded to their
// singular: what an index stores.
func TokenSet(text string) map[string]struct{} {
	raw := Tokens(text)
	set := make(map[string]struct{}, len(raw)*2)
	for _, token := range raw {
		set[token] = struct{}{}
		if singular := Singularize(token); singular != token {
			set[singular] = struct{}{}
		}
	}

	return set
}

// Singularize trims a trailing plural so "drivers" and "driver" are one term.
// It is deliberately crude: the cost of a wrong stem is a slightly worse
// ranking, not a wrong answer.
func Singularize(token string) string {
	if len(token) > 3 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") {
		return strings.TrimSuffix(token, "s")
	}

	return token
}
