// Package agenttoolcatalog indexes the agent tool registries so a turn can be
// given the few tools it needs instead of all of them.
//
// A flat catalog does not scale. Thirteen tools already cost roughly 3,900
// tokens of schema on every turn, so covering Trenova's actions — 171 permission
// resources across twenty operations — would cost more context than most models
// have, and small models pick badly long before they run out of room. Ranking
// here costs nothing at inference time and works the same on a free model as on
// a frontier one.
package agenttoolcatalog

import (
	"sort"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
)

const (
	nameMatchWeight        = 3
	descriptionMatchWeight = 1
)

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
}

type indexed struct {
	descriptor  serviceports.AgentToolDescriptor
	nameTokens  map[string]struct{}
	bodyTokens  map[string]struct{}
	catalogRank int
}

type Catalog struct {
	entries []indexed
	byName  map[string]int
}

func New(descriptors []serviceports.AgentToolDescriptor) *Catalog {
	sorted := make([]serviceports.AgentToolDescriptor, len(descriptors))
	copy(sorted, descriptors)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Name < sorted[j].Name })

	catalog := &Catalog{
		entries: make([]indexed, 0, len(sorted)),
		byName:  make(map[string]int, len(sorted)),
	}

	for i, descriptor := range sorted {
		catalog.byName[descriptor.Name] = i
		catalog.entries = append(catalog.entries, indexed{
			descriptor:  descriptor,
			nameTokens:  tokenSet(descriptor.Name),
			bodyTokens:  tokenSet(descriptor.Description),
			catalogRank: i,
		})
	}

	return catalog
}

// Descriptor returns one tool by its exact name.
func (c *Catalog) Descriptor(name string) (serviceports.AgentToolDescriptor, bool) {
	index, ok := c.byName[name]
	if !ok {
		return serviceports.AgentToolDescriptor{}, false
	}

	return c.entries[index].descriptor, true
}

// Rank orders the tools an agent holds by how well they fit a request.
//
// allowed bounds the result to the names the agent is configured for; nil means
// the whole catalog. Narrowing must never widen: a tool the agent does not hold
// is not offered however well it scores, because pre-selection is a convenience
// and the configured list is the grant.
func (c *Catalog) Rank(
	allowed []string,
	query string,
	limit int,
) []serviceports.AgentToolDescriptor {
	return c.rank(allowed, query, limit, 0)
}

func (c *Catalog) rank(
	allowed []string,
	query string,
	limit int,
	minScore int,
) []serviceports.AgentToolDescriptor {
	if limit <= 0 {
		return nil
	}

	wanted := c.allowedSet(allowed)
	terms := expand(tokens(query))

	type scored struct {
		entry *indexed
		score int
	}

	candidates := make([]scored, 0, len(c.entries))
	for i := range c.entries {
		entry := &c.entries[i]
		if wanted != nil {
			if _, ok := wanted[entry.descriptor.Name]; !ok {
				continue
			}
		}
		value := score(entry, terms)
		if value < minScore {
			continue
		}
		candidates = append(candidates, scored{entry: entry, score: value})
	}

	// Ties break on catalog position so the same question always produces the
	// same toolbox; a model that sees a different set each turn cannot be
	// debugged from a transcript.
	sort.SliceStable(candidates, func(i, j int) bool {
		if candidates[i].score != candidates[j].score {
			return candidates[i].score > candidates[j].score
		}

		return candidates[i].entry.catalogRank < candidates[j].entry.catalogRank
	})

	if len(candidates) > limit {
		candidates = candidates[:limit]
	}

	out := make([]serviceports.AgentToolDescriptor, 0, len(candidates))
	for _, candidate := range candidates {
		out = append(out, candidate.entry.descriptor)
	}

	return out
}

// Find is the model's recovery path when pre-selection guessed wrong. It returns
// whole descriptors, schema included, because a name alone is not callable.
//
// Unlike Rank it returns only tools that actually matched. Rank is pre-selection
// and must always fill its slots — a turn needs a toolbox whatever the wording —
// but a search that answers a nonsense query with six arbitrary tools and the
// words "these tools are now callable" is worse than an empty answer: it tells
// the model it found what it was looking for.
func (c *Catalog) Find(
	allowed []string,
	query string,
	limit int,
) []serviceports.AgentToolDescriptor {
	return c.rank(allowed, query, limit, 1)
}

// Names lists every tool in the catalog, in a stable order.
func (c *Catalog) Names() []string {
	out := make([]string, 0, len(c.entries))
	for i := range c.entries {
		out = append(out, c.entries[i].descriptor.Name)
	}

	return out
}

func (c *Catalog) allowedSet(allowed []string) map[string]struct{} {
	if len(allowed) == 0 {
		return nil
	}

	wanted := make(map[string]struct{}, len(allowed))
	for _, name := range allowed {
		wanted[name] = struct{}{}
	}

	return wanted
}

func score(entry *indexed, terms map[string]struct{}) int {
	total := 0
	for term := range terms {
		if _, ok := entry.nameTokens[term]; ok {
			total += nameMatchWeight
			continue
		}
		if _, ok := entry.bodyTokens[term]; ok {
			total += descriptionMatchWeight
		}
	}

	return total
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
		if singular := singularize(token); singular != token {
			terms[singular] = struct{}{}
			for _, synonym := range vocabulary[singular] {
				terms[synonym] = struct{}{}
			}
		}
	}

	return terms
}

func tokens(text string) []string {
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

func tokenSet(text string) map[string]struct{} {
	raw := tokens(text)
	set := make(map[string]struct{}, len(raw)*2)
	for _, token := range raw {
		set[token] = struct{}{}
		if singular := singularize(token); singular != token {
			set[singular] = struct{}{}
		}
	}

	return set
}

// singularize trims a trailing plural so "drivers" and "driver" are one term.
// It is deliberately crude: the cost of a wrong stem is a slightly worse
// ranking, not a wrong answer.
func singularize(token string) string {
	if len(token) > 3 && strings.HasSuffix(token, "s") && !strings.HasSuffix(token, "ss") {
		return strings.TrimSuffix(token, "s")
	}

	return token
}
