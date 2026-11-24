package capabilitysearch

import (
	"math"
	"slices"
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/agent/domain/agentcapability"
)

const (
	DefaultLimit = 8

	entityKeywordCap  = 10.0
	secondOpinionSlot = 3

	nameWeight           = 3.0
	phraseWeight         = 2.5
	bodyWeight           = 1.0
	paramWeight          = 0.4
	expansionWeight      = 0.6
	defaultKeywordWeight = 1.0
	defaultEntityBoost   = 2.0
	defaultPhraseTerm    = 3.0
	verbBoost            = 4.0
	readBoost            = 1.5
	writePenalty         = 0.45
	readPenalty          = 0.7
	lowConfidence        = 6.0
	semanticScale        = 8.0
)

type entry struct {
	capability agentcapability.Capability
	name       map[string]struct{}
	phrases    map[string]struct{}
	body       map[string]struct{}
	params     map[string]struct{}
	verbs      map[string]struct{}
	write      bool
	rank       int
}

type Weights struct {
	Entity  float64
	Keyword float64
	Phrase  float64
}

type Index struct {
	Weights  Weights
	byID     map[string]int
	entries  []entry
	idf      map[string]float64
	entities []entityWords
}

type entityWords struct {
	name    string
	phrases []string
}

type Text struct {
	Body   string
	Params string
}

type Query struct {
	Keyword  map[string]float64
	Text     string
	Limit    int
	Semantic map[string]float64
	Within   func(agentcapability.Capability) bool
}

type Hit struct {
	Capability agentcapability.Capability
	Score      float64
}

type Answer struct {
	Hits          []Hit
	LowConfidence bool
	Rephrasings   []string
	Entities      []string
}

func New(capabilities []agentcapability.Capability, texts map[string]Text) *Index {
	sorted := slices.Clone(capabilities)
	sort.SliceStable(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	index := &Index{
		entries: make([]entry, 0, len(sorted)),
		idf:     make(map[string]float64, 2048),
		byID:    make(map[string]int, len(sorted)),
		Weights: Weights{Entity: defaultEntityBoost, Keyword: defaultKeywordWeight, Phrase: defaultPhraseTerm},
	}
	df := make(map[string]int, 2048)
	for idx := range sorted {
		e := newEntry(&sorted[idx], idx, texts[sorted[idx].ID])
		seen := make(map[string]struct{}, 32)
		for _, set := range []map[string]struct{}{e.name, e.body, e.params} {
			for term := range set {
				seen[term] = struct{}{}
			}
		}
		for p := range e.phrases {
			for _, term := range strings.Fields(p) {
				seen[term] = struct{}{}
			}
		}
		for term := range seen {
			df[term]++
		}
		index.byID[e.capability.ID] = len(index.entries)
		index.entries = append(index.entries, e)
	}
	total := float64(len(index.entries))
	for term, count := range df {
		index.idf[term] = math.Log(1 + total/float64(count))
	}
	for _, entity := range agentcapability.Entities() {
		words := entityWords{name: entity.Name}
		words.phrases = append(words.phrases, phrase(strings.ReplaceAll(entity.Name, "_", " ")), phrase(entity.Title))
		for _, synonym := range entity.Synonyms {
			words.phrases = append(words.phrases, phrase(synonym))
		}
		index.entities = append(index.entities, words)
	}

	return index
}

func newEntry(capability *agentcapability.Capability, rank int, text Text) entry {
	e := entry{
		capability: *capability,
		name:       make(map[string]struct{}, 8),
		phrases:    make(map[string]struct{}, 8),
		body:       make(map[string]struct{}, 32),
		params:     make(map[string]struct{}, 32),
		verbs:      make(map[string]struct{}, 4),
		write:      capability.Access == agentcapability.AccessWrite,
		rank:       rank,
	}
	nameText := strings.ReplaceAll(capability.Name, "_", " ")
	if capability.IsEntityRead() {
		nameText = string(capability.Verb) + " " + strings.ReplaceAll(capability.Entity, "_", " ")
	}
	for _, term := range Terms(nameText) {
		e.name[term] = struct{}{}
	}
	for _, term := range Terms(strings.ReplaceAll(capability.Entity, "_", " ")) {
		e.name[term] = struct{}{}
	}
	_, rest, _ := strings.Cut(capability.Name, "_")
	for _, word := range strings.Split(rest, "_") {
		e.verbs[Stem(word)] = struct{}{}
	}
	for _, text := range append(slices.Clone(capability.Synonyms), capability.Examples...) {
		if p := phrase(text); p != "" {
			e.phrases[p] = struct{}{}
		}
	}
	for _, term := range Terms(capability.Summary + " " + capability.UseWhen + " " + text.Body) {
		e.body[term] = struct{}{}
	}
	for _, term := range Terms(text.Params) {
		e.params[term] = struct{}{}
	}

	return e
}

func (i *Index) Search(q Query) Answer {
	limit := q.Limit
	if limit <= 0 {
		limit = DefaultLimit
	}
	terms := Terms(q.Text)
	expanded := expandTerms(terms)
	joined := " " + strings.Join(terms, " ") + " "
	entities := i.mentioned(joined)
	intent := intentOf(q.Text, i.entityStart)

	hits := make([]Hit, 0, len(i.entries))
	for idx := range i.entries {
		e := &i.entries[idx]
		if q.Within != nil && !q.Within(e.capability) {
			continue
		}
		score := i.lexical(e, terms, joined) + expansionWeight*i.termScore(e, expanded)
		if weight, ok := entities[e.capability.Entity]; ok {
			if e.write {
				weight *= 0.5
			}
			score += i.Weights.Entity * weight
		}
		score += intent.adjust(e)
		score += i.Weights.Keyword * i.keyword(e, q.Keyword)
		if similarity, ok := q.Semantic[e.capability.ID]; ok {
			score += similarity * semanticScale
		}
		if score <= 0 {
			continue
		}
		if e.write && !intent.writes {
			score *= writePenalty
		}
		if !e.write && intent.writes && !intent.reads {
			score *= readPenalty
		}
		hits = append(hits, Hit{Capability: e.capability, Score: score})
	}
	sortHits(hits)
	hits = i.withSecondOpinion(hits, q.Keyword, intent)
	hits = i.withChain(hits)
	if len(hits) > limit {
		hits = hits[:limit]
	}

	answer := Answer{Hits: hits, Entities: sortedEntities(entities)}
	if len(hits) == 0 || hits[0].Score < lowConfidence {
		answer.LowConfidence = true
		answer.Rephrasings = rephrasings(hits, answer.Entities, intent)
	}

	return answer
}

func (i *Index) lexical(e *entry, terms []string, joined string) float64 {
	score := i.termScore(e, terms)
	for p := range e.phrases {
		if strings.Contains(p, " ") && strings.Contains(joined, " "+p+" ") {
			score += phraseWeight * float64(strings.Count(p, " ")+1)
		} else if !strings.Contains(p, " ") && p != "" && strings.Contains(joined, " "+p+" ") && !has(e.name, p) {
			score += phraseWeight * 0.5
		}
	}

	return score
}

func (i *Index) termScore(e *entry, terms []string) float64 {
	score := 0.0
	for _, term := range terms {
		idf := i.idf[term]
		if idf == 0 {
			continue
		}
		switch {
		case has(e.name, term):
			score += nameWeight * idf
		case inPhrases(e.phrases, term):
			score += i.Weights.Phrase * idf
		case has(e.body, term):
			score += bodyWeight * idf
		case has(e.params, term):
			score += paramWeight * idf
		}
	}

	return score
}

func expandTerms(terms []string) []string {
	out := make([]string, 0, len(terms))
	for _, term := range terms {
		for _, synonym := range agentsearch.Synonyms(term) {
			synonym = Stem(synonym)
			if !slices.Contains(terms, synonym) && !slices.Contains(out, synonym) {
				out = append(out, synonym)
			}
		}
	}

	return out
}

func (i *Index) mentioned(joined string) map[string]float64 {
	type mention struct {
		length int
		start  int
	}
	found := make(map[string]mention, 4)
	for _, entity := range i.entities {
		best := mention{start: math.MaxInt}
		for _, p := range entity.phrases {
			if p == "" {
				continue
			}
			at := strings.Index(joined, " "+p+" ")
			if at < 0 {
				continue
			}
			length := strings.Count(p, " ") + 1
			start := strings.Count(joined[:at+1], " ")
			if start < best.start || (start == best.start && length > best.length) {
				best = mention{length: length, start: start}
			}
		}
		if best.length > 0 {
			found[entity.name] = best
		}
	}
	out := make(map[string]float64, len(found))
	head := ""
	for name, m := range found {
		if head == "" || m.start < found[head].start ||
			(m.start == found[head].start && m.length > found[head].length) {
			head = name
		}
	}
	for name, m := range found {
		weight := float64(m.length)
		if name == head {
			weight++
		} else {
			weight *= 0.5
		}
		out[name] = weight
	}

	return out
}

type intent struct {
	entityStart func(first, second string) bool
	reads       bool
	writes      bool
	get         bool
	actions     map[string]struct{}
}

func intentOf(text string, entityStart func(first, second string) bool) intent {
	out := intent{actions: make(map[string]struct{}, 4), entityStart: entityStart}
	for _, clause := range strings.FieldsFunc(text, func(r rune) bool {
		return r == ',' || r == ';' || r == '.' || r == '!' || r == '?' || r == ':'
	}) {
		out.read(Tokens(clause))
	}

	return out
}

func (i *Index) entityStart(first, second string) bool {
	pair := Stem(first) + " " + Stem(second)
	for _, entity := range i.entities {
		for _, p := range entity.phrases {
			if strings.HasPrefix(p+" ", pair+" ") {
				return true
			}
		}
	}
	for idx := range i.entries {
		if i.entries[idx].write {
			continue
		}
		for p := range i.entries[idx].phrases {
			if strings.HasPrefix(p+" ", pair+" ") {
				return true
			}
		}
	}

	return false
}

func (out *intent) read(raw []string) {
	for idx, word := range raw {
		if _, ok := readWords[word]; ok {
			out.reads = true
		}
		if _, ok := getWords[word]; ok {
			out.get = true
		}
		if isNumber(word) {
			out.get = true
		}
		if idx > 0 && !leadsVerb(raw[idx-1]) {
			continue
		}
		if idx+1 < len(raw) && out.entityStart != nil && out.entityStart(word, raw[idx+1]) {
			continue
		}
		mapped, known := writeWords[word]
		_, imperative := imperatives[word]
		if !known && !imperative {
			continue
		}
		out.writes = true
		out.actions[Stem(word)] = struct{}{}
		for _, verb := range mapped {
			out.actions[verb] = struct{}{}
		}
	}
}

func leadsVerb(previous string) bool {
	switch previous {
	case "and", "then", "please", "to", "can", "could", "should", "lets", "i", "we", "you", "also", "now",
		"just", "so", "it", "them":
		return true
	}

	return false
}

func (in intent) adjust(e *entry) float64 {
	score := 0.0
	if in.writes && e.write {
		for action := range in.actions {
			if _, ok := e.verbs[action]; ok {
				score += verbBoost
			}
		}
	}
	if e.capability.IsEntityRead() {
		switch {
		case in.get && e.capability.Verb == agentcapability.VerbGet:
			score += readBoost * 2
		case !in.get && e.capability.Verb == agentcapability.VerbList:
			score += readBoost * 2
		}
		if in.reads {
			score += readBoost
		}
	}

	return score
}

func rephrasings(hits []Hit, entities []string, in intent) []string {
	out := make([]string, 0, 3)
	seen := make(map[string]struct{}, 3)
	add := func(text string) {
		if _, ok := seen[text]; ok || len(out) == 3 {
			return
		}
		seen[text] = struct{}{}
		out = append(out, text)
	}
	verb := "list"
	if in.writes {
		verb = "change"
	}
	for _, entity := range entities {
		add(verb + " " + strings.ReplaceAll(entity, "_", " ") + " by status, date or customer")
	}
	for idx := range hits {
		if hits[idx].Capability.Entity != "" {
			add(verb + " " + strings.ReplaceAll(hits[idx].Capability.Entity, "_", " "))
		}
	}
	if len(out) == 0 {
		add("name the record (shipment, invoice, driver, carrier...) and what to do with it")
	}

	return out
}

func sortedEntities(found map[string]float64) []string {
	out := make([]string, 0, len(found))
	for name := range found {
		out = append(out, name)
	}
	sort.Slice(out, func(a, b int) bool {
		if found[out[a]] != found[out[b]] {
			return found[out[a]] > found[out[b]]
		}

		return out[a] < out[b]
	})

	return out
}

func has(set map[string]struct{}, term string) bool {
	_, ok := set[term]
	return ok
}

func inPhrases(phrases map[string]struct{}, term string) bool {
	for p := range phrases {
		if p == term || strings.HasPrefix(p, term+" ") || strings.HasSuffix(p, " "+term) ||
			strings.Contains(p, " "+term+" ") {
			return true
		}
	}

	return false
}

func (i *Index) withChain(hits []Hit) []Hit {
	if len(hits) == 0 {
		return hits
	}
	top := hits[0].Capability
	position := slices.Index(top.Chain, top.ID)
	if position <= 0 {
		return hits
	}
	insert := make([]Hit, 0, position)
	for _, step := range top.Chain[:position] {
		id := agentcapability.CapabilityIDForTool(step)
		at := slices.IndexFunc(hits, func(hit Hit) bool { return hit.Capability.ID == id })
		if at >= 0 && at < 3 {
			continue
		}
		idx, ok := i.byID[id]
		if !ok {
			continue
		}
		if at >= 0 {
			hits = slices.Delete(hits, at, at+1)
		}
		insert = append(insert, Hit{Capability: i.entries[idx].capability, Score: hits[0].Score})
	}
	if len(insert) == 0 {
		return hits
	}

	return slices.Concat(hits[:1], insert, hits[1:])
}

func (i *Index) keyword(e *entry, scores map[string]float64) float64 {
	if len(scores) == 0 {
		return 0
	}
	if !e.capability.IsEntityRead() {
		return scores[e.capability.Name]
	}
	entity, ok := agentcapability.EntityNamed(e.capability.Entity)
	if !ok {
		return 0
	}
	backing := entity.List
	if e.capability.Verb == agentcapability.VerbGet {
		backing = entity.Get
	}

	return min(scores[backing], entityKeywordCap)
}

func sortHits(hits []Hit) {
	sort.SliceStable(hits, func(a, b int) bool {
		if hits[a].Score != hits[b].Score {
			return hits[a].Score > hits[b].Score
		}

		return hits[a].Capability.ID < hits[b].Capability.ID
	})
}

func (i *Index) withSecondOpinion(hits []Hit, scores map[string]float64, in intent) []Hit {
	if len(scores) == 0 || len(hits) < secondOpinionSlot {
		return hits
	}
	best, bestScore := -1, 0.0
	for idx := range i.entries {
		e := &i.entries[idx]
		value := i.keyword(e, scores)
		if e.write && !in.writes {
			value *= writePenalty
		}
		if value > bestScore {
			best, bestScore = idx, value
		}
	}
	if best < 0 {
		return hits
	}
	id := i.entries[best].capability.ID
	at := slices.IndexFunc(hits, func(hit Hit) bool { return hit.Capability.ID == id })
	if at >= 0 && at < secondOpinionSlot {
		return hits
	}
	if at >= 0 {
		hits = slices.Delete(hits, at, at+1)
	}
	opinion := Hit{Capability: i.entries[best].capability, Score: hits[secondOpinionSlot-1].Score}

	return slices.Insert(hits, secondOpinionSlot-1, opinion)
}
