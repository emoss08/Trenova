package index

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"sort"
	"sync"
	"time"
)

const recordVersion = 3

type Range struct {
	FileID uint32
	Start  uint32
	End    uint32
}

func (r Range) Contains(line int) bool {
	return line >= int(r.Start) && line <= int(r.End)
}

type TestCoverage struct {
	Name      string
	Ranges    []Range
	AlwaysRun bool
	Note      string
	Duration  time.Duration
}

type Record struct {
	Version   int
	Package   string
	IndexedAt string
	Files     []string
	Tests     []TestCoverage
	Degraded  string

	// The lookup structures are derived lazily because records come out of gob,
	// and eagerly: a whole-package mutation run asks TestsCovering once per
	// mutation site, and linear-scanning every test's every range per site was
	// measurably worse than running the mutants.
	memo     sync.Once
	ids      map[string]uint32
	perFile  map[uint32][]fileSpan
	nameByID []string
}

type fileSpan struct {
	start uint32
	end   uint32
	test  int32
}

func (r *Record) buildLookups() {
	r.memo.Do(func() {
		r.ids = make(map[string]uint32, len(r.Files))
		for i, file := range r.Files {
			r.ids[file] = uint32(i)
		}

		r.perFile = make(map[uint32][]fileSpan)
		r.nameByID = make([]string, len(r.Tests))
		for ti, test := range r.Tests {
			r.nameByID[ti] = test.Name
			if test.AlwaysRun {
				continue
			}
			for _, rg := range test.Ranges {
				r.perFile[rg.FileID] = append(r.perFile[rg.FileID], fileSpan{
					start: rg.Start,
					end:   rg.End,
					test:  int32(ti),
				})
			}
		}
	})
}

func (r *Record) Usable() bool {
	return r != nil && r.Degraded == "" && len(r.Tests) > 0
}

func (r *Record) DurationOf(tests []string) time.Duration {
	wanted := make(map[string]struct{}, len(tests))
	for _, name := range tests {
		wanted[name] = struct{}{}
	}

	var total time.Duration
	for _, test := range r.Tests {
		if _, ok := wanted[test.Name]; ok {
			total += test.Duration
		}
	}

	return total
}

func (r *Record) Knows(absPath string) bool {
	_, ok := r.fileID(absPath)

	return ok
}

// ProfileOf reports a test's coverage breadth — how many distinct lines it
// executes across the workspace — and its indexed duration. Breadth is the
// focus signal survivor advice ranks by: among the tests that already execute
// a mutated line, the one covering the fewest lines is the one whose
// assertions sit closest to that line. Overlapping ranges are merged per file
// so a line counts once no matter how many blocks report it.
func (r *Record) ProfileOf(name string) (lines int, duration time.Duration, ok bool) {
	for _, test := range r.Tests {
		if test.Name != name {
			continue
		}

		perFile := make(map[uint32][]Range)
		for _, span := range test.Ranges {
			perFile[span.FileID] = append(perFile[span.FileID], span)
		}
		for _, spans := range perFile {
			sort.Slice(spans, func(i, j int) bool {
				if spans[i].Start != spans[j].Start {
					return spans[i].Start < spans[j].Start
				}

				return spans[i].End < spans[j].End
			})
			var coveredThrough int64 = -1
			for _, span := range spans {
				start := int64(span.Start)
				end := int64(span.End)
				if start <= coveredThrough {
					start = coveredThrough + 1
				}
				if end < start {
					continue
				}
				lines += int(end-start) + 1
				coveredThrough = end
			}
		}

		return lines, test.Duration, true
	}

	return 0, 0, false
}

// LineCoverage reports, for one file, every covered line and how many distinct
// attributable tests execute it. Always-run tests carry no ranges, so their
// unknown attribution can never dress a line as covered.
func (r *Record) LineCoverage(absPath string) map[int]int {
	id, ok := r.fileID(absPath)
	if !ok {
		return nil
	}

	testsPerLine := make(map[int]map[int32]struct{})
	for _, span := range r.perFile[id] {
		for line := int(span.start); line <= int(span.end); line++ {
			set := testsPerLine[line]
			if set == nil {
				set = make(map[int32]struct{})
				testsPerLine[line] = set
			}
			set[span.test] = struct{}{}
		}
	}

	out := make(map[int]int, len(testsPerLine))
	for line, tests := range testsPerLine {
		out[line] = len(tests)
	}

	return out
}

func (r *Record) TestNames() []string {
	out := make([]string, 0, len(r.Tests))
	for _, test := range r.Tests {
		out = append(out, test.Name)
	}
	sort.Strings(out)

	return out
}

func (r *Record) AlwaysRunTests() []string {
	var out []string
	for _, test := range r.Tests {
		if test.AlwaysRun {
			out = append(out, test.Name)
		}
	}
	sort.Strings(out)

	return out
}

func (r *Record) fileID(absPath string) (uint32, bool) {
	r.buildLookups()
	id, ok := r.ids[filepath.Clean(absPath)]

	return id, ok
}

func (r *Record) TestsCovering(absPath string, lines []int) []string {
	id, ok := r.fileID(absPath)
	if !ok {
		return nil
	}

	covered := make(map[int32]struct{})
	for _, span := range r.perFile[id] {
		if _, seen := covered[span.test]; seen {
			continue
		}
		for _, line := range lines {
			if line >= int(span.start) && line <= int(span.end) {
				covered[span.test] = struct{}{}

				break
			}
		}
	}

	if len(covered) == 0 {
		return nil
	}

	out := make([]string, 0, len(covered))
	for test := range covered {
		out = append(out, r.nameByID[test])
	}
	sort.Strings(out)

	return out
}

func (r *Record) Encode() ([]byte, error) {
	var buf bytes.Buffer
	if err := gob.NewEncoder(&buf).Encode(r); err != nil {
		return nil, fmt.Errorf("encode index record: %w", err)
	}

	return buf.Bytes(), nil
}

func Decode(payload []byte) (*Record, error) {
	var record Record
	if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(&record); err != nil {
		return nil, fmt.Errorf("decode index record: %w", err)
	}
	if record.Version != recordVersion {
		return nil, fmt.Errorf("unsupported index record version %d", record.Version)
	}

	return &record, nil
}

type builder struct {
	files map[string]uint32
	order []string
}

func newBuilder() *builder {
	return &builder{files: make(map[string]uint32)}
}

func (b *builder) intern(absPath string) uint32 {
	clean := filepath.Clean(absPath)
	if id, ok := b.files[clean]; ok {
		return id
	}

	id := uint32(len(b.order))
	b.files[clean] = id
	b.order = append(b.order, clean)

	return id
}
