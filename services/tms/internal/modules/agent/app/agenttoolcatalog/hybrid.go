package agenttoolcatalog

import (
	"sort"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/agentsearch"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/shared/rankfusion"
)

const (
	DefaultSimilarityFloor = agentports.DefaultCatalogSimilarityFloor
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
	StrongOnly   bool
}

func (c *Catalog) Items() []agentports.EmbeddingCatalogItem {
	out := make([]agentports.EmbeddingCatalogItem, len(c.items))
	copy(out, c.items)

	return out
}

func DescriptorText(descriptor *agentports.AgentToolDescriptor) string {
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

func (c *Catalog) RankHybrid(q Query) []agentports.AgentToolDescriptor {
	if !q.Semantic.usable() {
		return c.Rank(q.Allowed, q.Text, q.Limit)
	}
	if q.Limit <= 0 {
		return nil
	}

	wanted := c.allowedSet(q.Allowed)
	terms := agentsearch.Terms(q.Text)
	fused, _ := c.fuse(wanted, c.keywordRanked(wanted, terms, keywordRanking{minScore: 1}), q)

	out := make([]agentports.AgentToolDescriptor, 0, q.Limit)
	taken := make(map[string]struct{}, q.Limit)
	for _, entry := range fused {
		if len(out) == q.Limit {
			return out
		}
		taken[entry.descriptor.Name] = struct{}{}
		out = append(out, entry.descriptor)
	}

	for _, candidate := range c.keywordRanked(wanted, terms, keywordRanking{}) {
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

func (c *Catalog) FindHybrid(q Query) []agentports.AgentToolDescriptor {
	return c.withFamilies(c.allowedSet(q.Allowed), c.matched(q))
}

type Match struct {
	Name  string
	Score float64
}

func (c *Catalog) Match(q Query) []Match {
	matched := c.matchedScored(q)
	out := make([]Match, 0, len(matched))
	for idx := range matched {
		out = append(out, Match{Name: matched[idx].descriptor.Name, Score: matched[idx].score})
	}

	return out
}

func (c *Catalog) matched(q Query) []agentports.AgentToolDescriptor {
	matched := c.matchedScored(q)
	found := make([]agentports.AgentToolDescriptor, 0, len(matched)+maxFamilySize)
	for idx := range matched {
		found = append(found, matched[idx].descriptor)
	}

	return found
}

type scoredDescriptor struct {
	descriptor agentports.AgentToolDescriptor
	score      float64
}

func (c *Catalog) matchedScored(q Query) []scoredDescriptor {
	if q.Limit <= 0 {
		return nil
	}

	wanted := c.allowedSet(q.Allowed)
	keyword := c.keywordRanked(wanted, agentsearch.Terms(q.Text), keywordRanking{
		minScore: 1,
		cutoff:   true,
		strong:   q.StrongOnly,
	})
	if !q.Semantic.usable() {
		if len(keyword) > q.Limit {
			keyword = keyword[:q.Limit]
		}
		out := make([]scoredDescriptor, 0, len(keyword))
		for _, candidate := range keyword {
			out = append(out, scoredDescriptor{
				descriptor: candidate.entry.descriptor,
				score:      float64(candidate.score),
			})
		}

		return out
	}

	fused, scores := c.fuse(wanted, keyword, q)
	if len(fused) > q.Limit {
		fused = fused[:q.Limit]
	}
	out := make([]scoredDescriptor, 0, len(fused))
	for _, entry := range fused {
		out = append(out, scoredDescriptor{
			descriptor: entry.descriptor,
			score:      scores[entry.descriptor.Name],
		})
	}

	return out
}

func (c *Catalog) fuse(
	wanted map[string]struct{},
	keyword []scored,
	q Query,
) ([]*indexed, map[string]float64) {
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

	return fused, scores
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
