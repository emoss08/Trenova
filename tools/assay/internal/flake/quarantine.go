package flake

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
	// DefaultQuarantinePath is committed, unlike the journal: it is curated
	// policy about which tests may not gate anything, not derived data.
	DefaultQuarantinePath = ".assay/flaky-tests.json"

	quarantineVersion = 1
)

type QuarantineEntry struct {
	Package   string `json:"package"`
	Test      string `json:"test"`
	Note      string `json:"note,omitempty"`
	AddedUnix int64  `json:"addedUnix,omitempty"`
}

// Quarantine is the committed list of tests whose verdicts may not be trusted
// to judge anything — most importantly, mutants: a flaky covering test marks
// mutants killed for reasons that have nothing to do with the mutation, and
// every such false kill inflates the score.
type Quarantine struct {
	Version int               `json:"version"`
	Tests   []QuarantineEntry `json:"tests"`

	known map[[2]string]struct{}
}

func NewQuarantine() *Quarantine {
	return &Quarantine{Version: quarantineVersion, known: make(map[[2]string]struct{})}
}

func LoadQuarantine(path string) (*Quarantine, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return NewQuarantine(), nil
		}

		return nil, fmt.Errorf("read quarantine: %w", err)
	}

	var q Quarantine
	if err := json.Unmarshal(payload, &q); err != nil {
		return nil, fmt.Errorf("parse quarantine %s: %w", path, err)
	}
	if q.Version != quarantineVersion {
		return nil, fmt.Errorf("quarantine %s has version %d, expected %d", path, q.Version, quarantineVersion)
	}
	q.index()

	return &q, nil
}

func (q *Quarantine) index() {
	q.known = make(map[[2]string]struct{}, len(q.Tests))
	for _, entry := range q.Tests {
		q.known[[2]string{entry.Package, entry.Test}] = struct{}{}
	}
}

func (q *Quarantine) Contains(pkg, test string) bool {
	if q == nil {
		return false
	}
	_, ok := q.known[[2]string{pkg, test}]

	return ok
}

func (q *Quarantine) Len() int {
	if q == nil {
		return 0
	}

	return len(q.Tests)
}

// Add records entries that are not already present and reports how many were
// new.
func (q *Quarantine) Add(entries ...QuarantineEntry) int {
	added := 0
	for _, entry := range entries {
		key := [2]string{entry.Package, entry.Test}
		if _, dup := q.known[key]; dup {
			continue
		}
		q.known[key] = struct{}{}
		q.Tests = append(q.Tests, entry)
		added++
	}
	sort.Slice(q.Tests, func(i, j int) bool {
		if q.Tests[i].Package != q.Tests[j].Package {
			return q.Tests[i].Package < q.Tests[j].Package
		}

		return q.Tests[i].Test < q.Tests[j].Test
	})

	return added
}

// Blocked renders the quarantine in the shape mutation exclusion consumes.
func (q *Quarantine) Blocked() map[string][]string {
	if q == nil || len(q.Tests) == 0 {
		return nil
	}

	out := make(map[string][]string)
	for _, entry := range q.Tests {
		out[entry.Package] = append(out[entry.Package], entry.Test)
	}

	return out
}

func (q *Quarantine) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create quarantine directory: %w", err)
	}

	payload, err := json.MarshalIndent(q, "", "  ")
	if err != nil {
		return fmt.Errorf("encode quarantine: %w", err)
	}
	payload = append(payload, '\n')

	if err := os.WriteFile(path, payload, 0o644); err != nil {
		return fmt.Errorf("write quarantine: %w", err)
	}

	return nil
}
