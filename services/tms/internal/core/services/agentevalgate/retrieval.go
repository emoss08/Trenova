package agentevalgate

import (
	"errors"
	"fmt"
	"math"
	"os"
	"slices"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	RetrievalRecallDepth = 5
	RetrievalNDCGDepth   = 10

	RetrievalEmbeddingFixturePath = "evals/embeddings/retrieval-nomic-embed-text.json"
)

type RetrievalSuite struct {
	Source string          `yaml:"source"`
	Corpus []RetrievalItem `yaml:"corpus"`
	Cases  []RetrievalCase `yaml:"cases"`
}

type RetrievalItem struct {
	Key        string   `yaml:"key"`
	FileName   string   `yaml:"fileName,omitempty"`
	Kind       string   `yaml:"kind,omitempty"`
	AttachedTo string   `yaml:"attachedTo,omitempty"`
	Pages      []string `yaml:"pages,omitempty"`
	Fields     []string `yaml:"fields,omitempty"`
	FromName   string   `yaml:"fromName,omitempty"`
	From       string   `yaml:"from,omitempty"`
	Subject    string   `yaml:"subject,omitempty"`
	Body       string   `yaml:"body,omitempty"`
	Tool       string   `yaml:"tool,omitempty"`
	Content    string   `yaml:"content,omitempty"`
}

type RetrievalCase struct {
	Query    string   `yaml:"query"`
	Relevant []string `yaml:"relevant"`
}

type RetrievalFloors struct {
	Cases     int     `json:"cases"`
	RecallAt5 float64 `json:"recallAt5"`
	MRR       float64 `json:"mrr"`
	NDCGAt10  float64 `json:"ndcgAt10"`
}

type RetrievalFloorSet struct {
	Keyword *RetrievalFloors `json:"keyword,omitempty"`
	Hybrid  *RetrievalFloors `json:"hybrid,omitempty"`
}

type RetrievalOutcome struct {
	Case      RetrievalCase
	Ranked    []string
	FirstHit  int
	RecallAt5 float64
	NDCGAt10  float64
}

type RetrievalReport struct {
	Outcomes  []RetrievalOutcome
	RecallAt5 float64
	MRR       float64
	NDCGAt10  float64
}

type Retriever func(query string) ([]string, error)

func LoadRetrievalSuite(path string) (*RetrievalSuite, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var suite RetrievalSuite
	if err = yaml.Unmarshal(raw, &suite); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	if err = suite.Validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}

	return &suite, nil
}

func (s *RetrievalSuite) Validate() error {
	keys := make(map[string]struct{}, len(s.Corpus))
	for _, item := range s.Corpus {
		if strings.TrimSpace(item.Key) == "" {
			return errors.New("every corpus item has a key")
		}
		if _, duplicate := keys[item.Key]; duplicate {
			return fmt.Errorf("corpus key %q is used twice", item.Key)
		}
		keys[item.Key] = struct{}{}
	}

	for _, retrieval := range s.Cases {
		if strings.TrimSpace(retrieval.Query) == "" || len(retrieval.Relevant) == 0 {
			return fmt.Errorf("case %q needs a query and at least one relevant item",
				retrieval.Query)
		}
		for _, key := range retrieval.Relevant {
			if _, ok := keys[key]; !ok {
				return fmt.Errorf("case %q names %q, which the corpus does not hold",
					retrieval.Query, key)
			}
		}
	}

	return nil
}

func (s *RetrievalSuite) Queries() []string {
	queries := make([]string, 0, len(s.Cases))
	for _, retrieval := range s.Cases {
		queries = append(queries, retrieval.Query)
	}

	return queries
}

func EvaluateRetrieval(suite *RetrievalSuite, retrieve Retriever) (RetrievalReport, error) {
	report := RetrievalReport{Outcomes: make([]RetrievalOutcome, 0, len(suite.Cases))}
	if len(suite.Cases) == 0 {
		return report, nil
	}

	var recall, reciprocal, ndcg float64
	for _, retrieval := range suite.Cases {
		ranked, err := retrieve(retrieval.Query)
		if err != nil {
			return report, fmt.Errorf("retrieve %q: %w", retrieval.Query, err)
		}

		outcome := ScoreRetrieval(retrieval, ranked)
		recall += outcome.RecallAt5
		if outcome.FirstHit > 0 {
			reciprocal += 1 / float64(outcome.FirstHit)
		}
		ndcg += outcome.NDCGAt10
		report.Outcomes = append(report.Outcomes, outcome)
	}

	total := float64(len(suite.Cases))
	report.RecallAt5 = recall / total
	report.MRR = reciprocal / total
	report.NDCGAt10 = ndcg / total

	return report, nil
}

func ScoreRetrieval(retrieval RetrievalCase, ranked []string) RetrievalOutcome {
	outcome := RetrievalOutcome{Case: retrieval, Ranked: ranked}

	found := 0
	dcg := 0.0
	for idx, key := range ranked {
		if !slices.Contains(retrieval.Relevant, key) || slices.Index(ranked, key) != idx {
			continue
		}
		position := idx + 1
		if outcome.FirstHit == 0 {
			outcome.FirstHit = position
		}
		if position <= RetrievalRecallDepth {
			found++
		}
		if position <= RetrievalNDCGDepth {
			dcg += 1 / math.Log2(float64(position)+1)
		}
	}

	relevant := len(retrieval.Relevant)
	outcome.RecallAt5 = float64(found) / float64(relevant)

	ideal := 0.0
	for position := 1; position <= min(relevant, RetrievalNDCGDepth); position++ {
		ideal += 1 / math.Log2(float64(position)+1)
	}
	if ideal > 0 {
		outcome.NDCGAt10 = dcg / ideal
	}

	return outcome
}

func (r RetrievalReport) Floors() RetrievalFloors {
	return RetrievalFloors{
		Cases:     len(r.Outcomes),
		RecallAt5: floorTo(r.RecallAt5),
		MRR:       floorTo(r.MRR),
		NDCGAt10:  floorTo(r.NDCGAt10),
	}
}

func (r RetrievalReport) Misses() string {
	var builder strings.Builder
	for _, outcome := range r.Outcomes {
		if outcome.FirstHit == 1 {
			continue
		}
		position := "not found"
		if outcome.FirstHit > 0 {
			position = fmt.Sprintf("first at %d", outcome.FirstHit)
		}
		fmt.Fprintf(&builder, "- %q wants %s, %s; ranked %s\n",
			outcome.Case.Query,
			strings.Join(outcome.Case.Relevant, " or "),
			position,
			strings.Join(outcome.Ranked, ", "),
		)
	}

	return builder.String()
}

func (r RetrievalReport) Below(floors RetrievalFloors) []string {
	below := make([]string, 0, 3)
	if r.RecallAt5 < floors.RecallAt5 {
		below = append(below, fmt.Sprintf("recall@5 %.2f < %.2f", r.RecallAt5, floors.RecallAt5))
	}
	if r.MRR < floors.MRR {
		below = append(below, fmt.Sprintf("MRR %.2f < %.2f", r.MRR, floors.MRR))
	}
	if r.NDCGAt10 < floors.NDCGAt10 {
		below = append(below, fmt.Sprintf("nDCG@10 %.2f < %.2f", r.NDCGAt10, floors.NDCGAt10))
	}

	return below
}
