package mutate

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

const (
	// DefaultBaselinePath is committed, unlike the caches: it is curated policy
	// about which gaps are accepted, not derived data.
	DefaultBaselinePath = ".assay/mutation-baseline.json"

	baselineVersion = 1
)

type BaselineEntry struct {
	ID       string `json:"id"`
	Package  string `json:"package"`
	Function string `json:"function"`
	Kind     Kind   `json:"kind"`
	Note     string `json:"note,omitempty"`
}

type Baseline struct {
	Version   int             `json:"version"`
	Survivors []BaselineEntry `json:"survivors"`

	accepted map[string]struct{}
}

func LoadBaseline(path string) (*Baseline, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewBaseline(), nil
		}

		return nil, fmt.Errorf("read baseline: %w", err)
	}

	var b Baseline
	if err := json.Unmarshal(payload, &b); err != nil {
		return nil, fmt.Errorf("parse baseline %s: %w", path, err)
	}
	if b.Version != baselineVersion {
		return nil, fmt.Errorf("baseline %s has version %d, expected %d", path, b.Version, baselineVersion)
	}

	b.index()

	return &b, nil
}

func NewBaseline() *Baseline {
	return &Baseline{Version: baselineVersion, accepted: make(map[string]struct{})}
}

func (b *Baseline) index() {
	b.accepted = make(map[string]struct{}, len(b.Survivors))
	for _, entry := range b.Survivors {
		b.accepted[entry.ID] = struct{}{}
	}
}

func (b *Baseline) Accepts(id string) bool {
	if b == nil {
		return false
	}
	_, ok := b.accepted[id]

	return ok
}

func (b *Baseline) Len() int {
	if b == nil {
		return 0
	}

	return len(b.Survivors)
}

// FromResults builds a baseline that accepts exactly the survivors observed.
func FromResults(results []Result) *Baseline {
	b := NewBaseline()
	for _, r := range results {
		if r.Outcome != OutcomeSurvived {
			continue
		}
		b.Survivors = append(b.Survivors, BaselineEntry{
			ID:       r.Mutant.ID,
			Package:  r.Mutant.Package,
			Function: r.Mutant.Function,
			Kind:     r.Mutant.Kind,
			Note:     r.Mutant.Original + " -> " + r.Mutant.Replacement,
		})
	}
	sort.Slice(b.Survivors, func(i, j int) bool { return b.Survivors[i].ID < b.Survivors[j].ID })
	b.index()

	return b
}

func (b *Baseline) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create baseline directory: %w", err)
	}

	payload, err := json.MarshalIndent(b, "", "  ")
	if err != nil {
		return fmt.Errorf("encode baseline: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write baseline: %w", err)
	}

	return nil
}

type Score struct {
	Killed       int      `json:"killed"`
	Survived     int      `json:"survived"`
	Timeout      int      `json:"timeout"`
	NotBuilt     int      `json:"notBuilt"`
	NoCoverage   int      `json:"noCoverage"`
	Accepted     int      `json:"accepted"`
	NewSurvivors []Result `json:"newSurvivors,omitempty"`
}

func (s Score) Scored() int {
	return s.Killed + s.Timeout + s.Survived
}

// MSI is the mutation score: the share of mutants a test actually noticed.
// Mutants that could not be built, or that no test covers, are excluded — the
// first is an assay defect and the second is already visible as a coverage gap.
func (s Score) MSI() float64 {
	total := s.Scored()
	if total == 0 {
		return 0
	}

	return 100 * float64(s.Killed+s.Timeout) / float64(total)
}

func Evaluate(results []Result, baseline *Baseline) Score {
	var score Score

	for _, r := range results {
		switch r.Outcome {
		case OutcomeKilled:
			score.Killed++
		case OutcomeTimeout:
			score.Timeout++
		case OutcomeNotBuilt:
			score.NotBuilt++
		case OutcomeNoCoverage:
			score.NoCoverage++
		case OutcomeSurvived:
			score.Survived++
			if baseline.Accepts(r.Mutant.ID) {
				score.Accepted++

				continue
			}
			score.NewSurvivors = append(score.NewSurvivors, r)
		}
	}

	sort.Slice(score.NewSurvivors, func(i, j int) bool {
		left, right := score.NewSurvivors[i].Mutant, score.NewSurvivors[j].Mutant
		if left.File != right.File {
			return left.File < right.File
		}

		return left.Line < right.Line
	})

	return score
}
