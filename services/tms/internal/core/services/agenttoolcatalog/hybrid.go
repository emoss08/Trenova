package agenttoolcatalog

import (
	"sort"
	"strings"

	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agentsearch"
	"github.com/emoss08/trenova/shared/rankfusion"
)

const (
	DefaultSimilarityFloor = serviceports.DefaultCatalogSimilarityFloor
	RRFK                   = rankfusion.DefaultK

	vectorCandidateFactor = 4
	minVectorCandidates   = 24
)

type Semantic struct {
	Similarity map[string]float64
	Floor      float64
}

func NewSemantic(similarity map[string]float64) *Semantic {
	if len(similarity) == 0 {
		return nil
	}

	return &Semantic{Similarity: similarity, Floor: DefaultSimilarityFloor}
}

func (s *Semantic) usable() bool { return s != nil && len(s.Similarity) > 0 }

type Query struct {
	Allowed      []string
	Text         string
	Limit        int
	Semantic     *Semantic
	SkipSemantic map[string]struct{}
}

func (c *Catalog) Items() []serviceports.EmbeddingCatalogItem {
	out := make([]serviceports.EmbeddingCatalogItem, len(c.items))
	copy(out, c.items)

	return out
}

func DescriptorText(descriptor serviceports.AgentToolDescriptor) string {
	var b strings.Builder
	b.Grow(len(descriptor.Name) + len(descriptor.Description) + 64)
	b.WriteString(strings.ReplaceAll(descriptor.Name, "_", " "))
	b.WriteString(": ")
	b.WriteString(strings.TrimSpace(descriptor.Description))
	if len(descriptor.SearchTerms) > 0 {
		b.WriteString("\nAlso asked for as: ")
		b.WriteString(strings.Join(descriptor.SearchTerms, ", "))
	}

	return b.String()
}

func (c *Catalog) RankHybrid(q Query) []serviceports.AgentToolDescriptor {
	if !q.Semantic.usable() {
		return c.Rank(q.Allowed, q.Text, q.Limit)
	}
	if q.Limit <= 0 {
		return nil
	}

	wanted := c.allowedSet(q.Allowed)
	terms := agentsearch.Terms(q.Text)
	fused := c.fuse(wanted, c.keywordRanked(wanted, terms, 1, false), q)

	out := make([]serviceports.AgentToolDescriptor, 0, q.Limit)
	taken := make(map[string]struct{}, q.Limit)
	for _, entry := range fused {
		if len(out) == q.Limit {
			return out
		}
		taken[entry.descriptor.Name] = struct{}{}
		out = append(out, entry.descriptor)
	}

	for _, candidate := range c.keywordRanked(wanted, terms, 0, false) {
		if len(out) == q.Limit {
			break
		}
		if _, ok := taken[candidate.entry.descriptor.Name]; ok {
			continue
		}
		out = append(out, candidate.entry.descriptor)
	}

	return out
}

func (c *Catalog) FindHybrid(q Query) []serviceports.AgentToolDescriptor {
	if !q.Semantic.usable() {
		return c.Find(q.Allowed, q.Text, q.Limit)
	}
	if q.Limit <= 0 {
		return nil
	}

	wanted := c.allowedSet(q.Allowed)
	fused := c.fuse(wanted, c.keywordRanked(wanted, agentsearch.Terms(q.Text), 1, true), q)
	if len(fused) > q.Limit {
		fused = fused[:q.Limit]
	}

	found := make([]serviceports.AgentToolDescriptor, 0, len(fused)+maxFamilySize)
	for _, entry := range fused {
		found = append(found, entry.descriptor)
	}

	return c.withFamilies(wanted, found)
}

func (c *Catalog) fuse(wanted map[string]struct{}, keyword []scored, q Query) []*indexed {
	keywordNames := make([]string, 0, len(keyword))
	matched := make(map[string]struct{}, len(keyword))
	for _, candidate := range keyword {
		keywordNames = append(keywordNames, candidate.entry.descriptor.Name)
		matched[candidate.entry.descriptor.Name] = struct{}{}
	}

	vector := c.vectorRanked(wanted, q)
	vectorNames := make([]string, 0, len(vector))
	for _, entry := range vector {
		vectorNames = append(vectorNames, entry.descriptor.Name)
	}

	scores := rankfusion.Reciprocal(RRFK, keywordNames, vectorNames)
	fused := make([]*indexed, 0, len(scores))
	for name := range scores {
		if _, ok := matched[name]; !ok && q.Semantic.Similarity[name] < q.Semantic.Floor {
			continue
		}
		fused = append(fused, &c.entries[c.byName[name]])
	}

	sort.Slice(fused, func(i, j int) bool {
		left, right := scores[fused[i].descriptor.Name], scores[fused[j].descriptor.Name]
		if left != right {
			return left > right
		}

		return fused[i].catalogRank < fused[j].catalogRank
	})

	return fused
}

func (c *Catalog) vectorRanked(wanted map[string]struct{}, q Query) []*indexed {
	ranked := make([]*indexed, 0, len(q.Semantic.Similarity))
	for i := range c.entries {
		entry := &c.entries[i]
		name := entry.descriptor.Name
		if wanted != nil {
			if _, ok := wanted[name]; !ok {
				continue
			}
		}
		if _, skip := q.SkipSemantic[name]; skip {
			continue
		}
		if _, ok := q.Semantic.Similarity[name]; !ok {
			continue
		}
		ranked = append(ranked, entry)
	}

	sort.Slice(ranked, func(i, j int) bool {
		left := q.Semantic.Similarity[ranked[i].descriptor.Name]
		right := q.Semantic.Similarity[ranked[j].descriptor.Name]
		if left != right {
			return left > right
		}

		return ranked[i].catalogRank < ranked[j].catalogRank
	})

	candidates := max(q.Limit*vectorCandidateFactor, minVectorCandidates)
	if len(ranked) > candidates {
		ranked = ranked[:candidates]
	}

	return ranked
}
