package index

import (
	"bytes"
	"encoding/gob"
	"fmt"
	"path/filepath"
	"sort"
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
	clean := filepath.Clean(absPath)
	for i, candidate := range r.Files {
		if candidate == clean {
			return uint32(i), true
		}
	}

	return 0, false
}

func (r *Record) TestsCovering(absPath string, lines []int) []string {
	id, ok := r.fileID(absPath)
	if !ok {
		return nil
	}

	var out []string
	for _, test := range r.Tests {
		if test.AlwaysRun {
			continue
		}
		if coversAny(test.Ranges, id, lines) {
			out = append(out, test.Name)
		}
	}
	sort.Strings(out)

	return out
}

func coversAny(ranges []Range, fileID uint32, lines []int) bool {
	for _, r := range ranges {
		if r.FileID != fileID {
			continue
		}
		for _, line := range lines {
			if r.Contains(line) {
				return true
			}
		}
	}

	return false
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
