// Package pageaccess is the page access manifest: for every screen of the
// web app, what its route demands, the GraphQL operations and REST endpoints
// its code runs, the kinds it reads as root records, the kinds it shows only
// through their reference views, and the kinds whose further fields it
// declares with @requiresRead.
//
// manifest_gen.json is generated in two steps: the web app's route tree and
// module graph give screens_gen.json (`pnpm --filter @trenova/web
// access:generate`), and `go run ./internal/api/graphql/gqlext/refgen` reads
// it with the operations, the schema and the permission registry. See
// docs/engineering/reference-views.md. It is embedded, so a server always
// describes the app it was built with.
package pageaccess

import (
	_ "embed"
	"fmt"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
)

//go:embed manifest_gen.json
var raw []byte

// Requirement is a permission a route's loader or a REST route checks.
type Requirement struct {
	Resource  string `json:"resource"`
	Operation string `json:"operation"`
}

// Endpoint is a REST endpoint a screen calls, with the permissions its route
// checks before its handler. Requires is empty for a route that checks
// inside its handler or service.
type Endpoint struct {
	Method   string        `json:"method"`
	Path     string        `json:"path"`
	Requires []Requirement `json:"requires"`
}

// LinkedKind is one kind an operation reaches as a record another record
// links to, and every field it selects on one. A field written Type.field
// belongs to a record that is part of one of the kind (a move of a linked
// shipment), which has no view of its own.
type LinkedKind struct {
	Kind   string   `json:"kind"`
	Fields []string `json:"fields"`
}

type Operation struct {
	Name string `json:"name"`
	// Type is query, mutation or subscription.
	Type     string       `json:"type"`
	Reads    []string     `json:"reads"`
	Linked   []LinkedKind `json:"linked"`
	Declared []string     `json:"declared"`
}

type Screen struct {
	Path       string        `json:"path"`
	Name       string        `json:"name"`
	Requires   []Requirement `json:"requires"`
	Operations []string      `json:"operations"`
	Endpoints  []Endpoint    `json:"endpoints"`
	// Reads is the kinds the screen's queries return from their root
	// resolvers, which need the kind's read.
	Reads []string `json:"reads"`
	// References is the kinds the screen shows only through their reference
	// views.
	References []string `json:"references"`
	// Declared is the kinds whose fields beyond the view the screen asks for
	// with @requiresRead.
	Declared []string `json:"declared"`
}

type Manifest struct {
	Version    string      `json:"version"`
	Screens    []Screen    `json:"screens"`
	Operations []Operation `json:"operations"`

	operations map[string]int
	exact      map[string]int
}

// Default is the manifest this server was built with.
var Default = mustLoad(raw)

func mustLoad(data []byte) *Manifest {
	manifest, err := Load(data)
	if err != nil {
		panic(fmt.Sprintf("pageaccess: %v", err))
	}
	return manifest
}

// Load reads a manifest and indexes it.
func Load(data []byte) (*Manifest, error) {
	manifest := new(Manifest)
	if err := sonic.Unmarshal(data, manifest); err != nil {
		return nil, fmt.Errorf("read the page access manifest: %w", err)
	}
	manifest.Index()
	return manifest, nil
}

// Index builds the lookups Operation and Screen read; Load calls it.
func (m *Manifest) Index() {
	m.operations = make(map[string]int, len(m.Operations))
	for i := range m.Operations {
		m.operations[m.Operations[i].Name] = i
	}
	m.exact = make(map[string]int, len(m.Screens))
	for i := range m.Screens {
		m.exact[m.Screens[i].Path] = i
	}
}

// Operation is one operation by name.
func (m *Manifest) Operation(name string) (*Operation, bool) {
	i, ok := m.operations[name]
	if !ok {
		return nil, false
	}
	return &m.Operations[i], true
}

// Screen is the screen a location shows: the route with that path, or the
// route pattern it fills in with the most literal segments.
func (m *Manifest) Screen(location string) (*Screen, bool) {
	path := normalize(location)
	if i, ok := m.exact[path]; ok {
		return &m.Screens[i], true
	}
	segments := strings.Split(path, "/")
	best, bestLiterals := -1, -1
	for i := range m.Screens {
		literals, ok := matches(m.Screens[i].Path, segments)
		if ok && literals > bestLiterals {
			best, bestLiterals = i, literals
		}
	}
	if best < 0 {
		return nil, false
	}
	return &m.Screens[best], true
}

func normalize(location string) string {
	path, _, _ := strings.Cut(location, "?")
	path, _, _ = strings.Cut(path, "#")
	if path != "/" {
		path = strings.TrimRight(path, "/")
	}
	if path == "" {
		return "/"
	}
	return path
}

func matches(pattern string, segments []string) (int, bool) {
	parts := strings.Split(pattern, "/")
	if len(parts) != len(segments) {
		return 0, false
	}
	literals := 0
	for i, part := range parts {
		switch {
		case strings.HasPrefix(part, ":"):
			if segments[i] == "" {
				return 0, false
			}
		case part == segments[i]:
			literals++
		default:
			return 0, false
		}
	}
	return literals, true
}

// Linked is every kind a screen's operations reach as linked records, with
// every field they select on one, sorted.
func (m *Manifest) Linked(screen *Screen) []LinkedKind {
	fields := make(map[string][]string)
	for _, name := range screen.Operations {
		op, ok := m.Operation(name)
		if !ok {
			continue
		}
		for _, linked := range op.Linked {
			for _, field := range linked.Fields {
				if !slices.Contains(fields[linked.Kind], field) {
					fields[linked.Kind] = append(fields[linked.Kind], field)
				}
			}
		}
	}
	out := make([]LinkedKind, 0, len(fields))
	for kind, list := range fields {
		slices.Sort(list)
		out = append(out, LinkedKind{Kind: kind, Fields: list})
	}
	slices.SortFunc(out, func(a, b LinkedKind) int { return strings.Compare(a.Kind, b.Kind) })
	return out
}

// Use is how a screen depends on one kind.
type Use string

const (
	// UseRequired is a kind the screen's route, or a REST endpoint it calls,
	// checks.
	UseRequired Use = "required"
	// UseRead is a kind the screen reads as root records.
	UseRead Use = "read"
	// UseDeclared is a kind whose fields beyond its view the screen asks for.
	UseDeclared Use = "declared"
	// UseReference is a kind the screen shows only through its view.
	UseReference Use = "reference"
)

// ScreenUse is one screen that depends on a kind, and the strongest way it
// does.
type ScreenUse struct {
	Screen *Screen
	Use    Use
}

// ScreensUsing is every screen that depends on a kind, by path.
func (m *Manifest) ScreensUsing(kind string) []ScreenUse {
	out := make([]ScreenUse, 0)
	for i := range m.Screens {
		if use, ok := m.Screens[i].UseOf(kind); ok {
			out = append(out, ScreenUse{Screen: &m.Screens[i], Use: use})
		}
	}
	return out
}

// UseOf is the strongest way a screen depends on a kind.
func (s *Screen) UseOf(kind string) (Use, bool) {
	required := func(requires []Requirement) bool {
		return slices.ContainsFunc(requires, func(r Requirement) bool { return r.Resource == kind })
	}
	switch {
	case required(s.Requires) ||
		slices.ContainsFunc(s.Endpoints, func(e Endpoint) bool { return required(e.Requires) }):
		return UseRequired, true
	case slices.Contains(s.Reads, kind):
		return UseRead, true
	case slices.Contains(s.Declared, kind):
		return UseDeclared, true
	case slices.Contains(s.References, kind):
		return UseReference, true
	default:
		return "", false
	}
}
