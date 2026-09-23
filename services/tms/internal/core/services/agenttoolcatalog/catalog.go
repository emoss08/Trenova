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
	"github.com/emoss08/trenova/internal/core/services/agentsearch"
)

const (
	nameMatchWeight        = 4
	descriptionMatchWeight = 2
	parameterMatchWeight   = 1

	// findCutoffDivisor keeps a find_tools answer to the tools that matched
	// about as well as the best one. Without it "list trailers" loaded the
	// trailer tools and then every other list tool on the word "list".
	findCutoffDivisor = 2
)

type indexed struct {
	descriptor  serviceports.AgentToolDescriptor
	nameTokens  map[string]struct{}
	bodyTokens  map[string]struct{}
	paramTokens map[string]struct{}
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
			descriptor: descriptor,
			nameTokens: agentsearch.TokenSet(
				descriptor.Name + " " + strings.Join(descriptor.SearchTerms, " "),
			),
			bodyTokens:  agentsearch.TokenSet(descriptor.Description),
			paramTokens: agentsearch.TokenSet(parameterText(descriptor.Parameters)),
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
	terms := agentsearch.Terms(query)

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

	if minScore > 0 && len(candidates) > 0 {
		floor := candidates[0].score / findCutoffDivisor
		kept := candidates[:0]
		for _, candidate := range candidates {
			if candidate.score >= floor {
				kept = append(kept, candidate)
			}
		}
		candidates = kept
	}

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

// Prerequisites names the tools a tool's arguments come from.
func (c *Catalog) Prerequisites(name string) []string {
	descriptor, ok := c.Descriptor(name)
	if !ok {
		return nil
	}

	return descriptor.Prerequisites
}

// IsQuery reports whether a tool only reads.
func (c *Catalog) IsQuery(name string) bool {
	descriptor, ok := c.Descriptor(name)

	return ok && descriptor.Query
}

// Names lists every tool in the catalog, in a stable order.
func (c *Catalog) Names() []string {
	out := make([]string, 0, len(c.entries))
	for i := range c.entries {
		out = append(out, c.entries[i].descriptor.Name)
	}

	return out
}

// allowedSet reads nil as the whole catalog and an empty list as nothing.
// The two used to read the same, so a person permitted no tools at all was
// offered every tool in the catalog.
func (c *Catalog) allowedSet(allowed []string) map[string]struct{} {
	if allowed == nil {
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
			continue
		}
		if _, ok := entry.paramTokens[term]; ok {
			total += parameterMatchWeight
		}
	}

	return total
}

// parameterText is the searchable text of a tool's parameters: their names
// and descriptions, one level of properties deep. A tool whose purpose is
// only in what it takes — a customerId filter on list_shipments — is found
// by it, below anything that names the thing outright.
func parameterText(schema map[string]any) string {
	properties, _ := schema["properties"].(map[string]any)
	if len(properties) == 0 {
		return ""
	}

	var b strings.Builder
	for name, raw := range properties {
		b.WriteString(agentsearch.SplitCamel(name))
		b.WriteByte(' ')
		if property, ok := raw.(map[string]any); ok {
			if description, ok := property["description"].(string); ok {
				b.WriteString(description)
				b.WriteByte(' ')
			}
		}
	}

	return b.String()
}
