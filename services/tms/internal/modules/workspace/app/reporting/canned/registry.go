package canned

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/keel/analytics/ir"
	"github.com/emoss08/trenova/internal/modules/workspace/domain/report"
)

//go:embed reports/*.json
var reports embed.FS

type Entry struct {
	Key           string
	Version       string
	Name          string
	Description   string
	Category      string
	Tags          []string
	DefaultFormat report.Format
	Definition    *ir.Definition
}

type file struct {
	Key           string                 `json:"key"`
	Version       string                 `json:"version"`
	Name          string                 `json:"name"`
	Description   string                 `json:"description"`
	Category      string                 `json:"category"`
	Tags          []string               `json:"tags"`
	DefaultFormat report.Format          `json:"defaultFormat"`
	Definition    sonic.NoCopyRawMessage `json:"definition"`
}

type Registry struct {
	entries map[string]*Entry
	ordered []*Entry
}

func NewRegistry(entries []*Entry) *Registry {
	r := &Registry{
		entries: make(map[string]*Entry, len(entries)),
		ordered: make([]*Entry, 0, len(entries)),
	}
	for _, entry := range entries {
		r.entries[entry.Key] = entry
		r.ordered = append(r.ordered, entry)
	}
	sort.Slice(r.ordered, func(i, j int) bool { return r.ordered[i].Key < r.ordered[j].Key })
	return r
}

func (r *Registry) Get(key string) (*Entry, bool) {
	entry, ok := r.entries[key]
	return entry, ok
}

func (r *Registry) All() []*Entry {
	return r.ordered
}

func Default() (*Registry, error) {
	names, err := fs.Glob(reports, "reports/*.json")
	if err != nil {
		return nil, fmt.Errorf("list canned reports: %w", err)
	}
	entries := make([]*Entry, 0, len(names))
	for _, name := range names {
		entry, loadErr := load(name)
		if loadErr != nil {
			return nil, loadErr
		}
		entries = append(entries, entry)
	}
	return NewRegistry(entries), nil
}

func load(name string) (*Entry, error) {
	raw, err := reports.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("read canned report %s: %w", name, err)
	}
	var f file
	if err = sonic.Unmarshal(raw, &f); err != nil {
		return nil, fmt.Errorf("decode canned report %s: %w", name, err)
	}
	if want := strings.TrimSuffix(path.Base(name), ".json"); f.Key != want {
		return nil, fmt.Errorf("canned report %s declares key %q", name, f.Key)
	}
	if !f.DefaultFormat.IsValid() {
		return nil, fmt.Errorf("canned report %s has format %q", name, f.DefaultFormat)
	}
	definition, err := ir.Decode([]byte(f.Definition))
	if err != nil {
		return nil, fmt.Errorf("decode canned report %s definition: %w", name, err)
	}
	return &Entry{
		Key:           f.Key,
		Version:       f.Version,
		Name:          f.Name,
		Description:   f.Description,
		Category:      f.Category,
		Tags:          f.Tags,
		DefaultFormat: f.DefaultFormat,
		Definition:    definition,
	}, nil
}
