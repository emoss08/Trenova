// Package pagemanifest builds the page access manifest (pkg/pageaccess) from
// the web app's screens, the client's GraphQL operations, the schema, the
// permission registry and the REST route table. Which kinds an operation
// reads as roots, reaches as linked records and declares comes from the same
// walk as the field guard's checks (gqlext.Operations), so the manifest says
// what the guard will do.
package pagemanifest

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/api/graphql/gqlext"
	"github.com/emoss08/trenova/internal/api/permissionlint"
	"github.com/emoss08/trenova/internal/api/routelint"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/pageaccess"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	// ScreensPath and ManifestPath are relative to the repository root.
	ScreensPath  = "services/tms/pkg/pageaccess/screens_gen.json"
	ManifestPath = "services/tms/pkg/pageaccess/manifest_gen.json"

	apiBase = "/api/v1"
)

// Screens is what the web app's generator writes: each screen's route, what
// its loaders demand, and what its code runs.
type Screens struct {
	Screens []ScreenEntry `json:"screens"`
}

type ScreenEntry struct {
	Path       string                   `json:"path"`
	Name       string                   `json:"name"`
	Requires   []pageaccess.Requirement `json:"requires"`
	Operations []string                 `json:"operations"`
	Rest       []RestCall               `json:"rest"`
}

type RestCall struct {
	Method string `json:"method"`
	Path   string `json:"path"`
}

type Inputs struct {
	Screens    Screens
	Schema     *ast.Schema
	Registry   *permission.Registry
	Operations ast.OperationList
	Fragments  ast.FragmentDefinitionList
	Routes     []routelint.Route
	Checks     []permissionlint.Pair
}

func Build(in *Inputs) (*pageaccess.Manifest, error) {
	accesses, err := gqlext.Operations(in.Schema, in.Registry, in.Operations, in.Fragments, nil)
	if err != nil {
		return nil, err
	}
	byName := make(map[string]*gqlext.OperationAccess, len(accesses))
	for i := range accesses {
		byName[accesses[i].Name] = &accesses[i]
	}
	routes := newRouteIndex(in.Routes, in.Checks)

	manifest := &pageaccess.Manifest{}
	used := make(map[string]struct{})
	var missing []string
	for _, entry := range in.Screens.Screens {
		screen := pageaccess.Screen{
			Path:       entry.Path,
			Name:       entry.Name,
			Requires:   orEmpty(entry.Requires),
			Operations: orEmpty(entry.Operations),
			Endpoints:  make([]pageaccess.Endpoint, 0, len(entry.Rest)),
		}
		var reads, linked, declared []string
		for _, name := range entry.Operations {
			access, ok := byName[name]
			if !ok {
				missing = append(missing, entry.Path+": "+name)
				continue
			}
			used[name] = struct{}{}
			if access.Operation == ast.Query {
				reads = addKinds(reads, access.Roots)
			}
			linked = addKinds(linked, access.Linked)
			declared = addKinds(declared, access.Declared)
		}
		for _, call := range entry.Rest {
			screen.Endpoints = append(screen.Endpoints, pageaccess.Endpoint{
				Method:   call.Method,
				Path:     call.Path,
				Requires: routes.requires(call.Method, call.Path),
			})
		}
		screen.Reads = sorted(reads)
		screen.Declared = sorted(declared)
		screen.References = sorted(slices.DeleteFunc(linked, func(kind string) bool {
			return slices.Contains(reads, kind) || slices.Contains(declared, kind)
		}))
		manifest.Screens = append(manifest.Screens, screen)
	}
	if len(missing) > 0 {
		return nil, fmt.Errorf(
			"screens run operations no client .graphql file defines (regenerate "+
				"screens_gen.json with `pnpm --filter @trenova/web access:generate`):\n  %s",
			strings.Join(missing, "\n  "),
		)
	}

	for i := range accesses {
		if _, ok := used[accesses[i].Name]; ok {
			manifest.Operations = append(manifest.Operations, operation(&accesses[i]))
		}
	}
	slices.SortFunc(manifest.Operations, func(a, b pageaccess.Operation) int {
		return strings.Compare(a.Name, b.Name)
	})

	body, err := sonic.ConfigStd.Marshal(manifest)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	manifest.Version = "sha256:" + hex.EncodeToString(sum[:])
	manifest.Index()

	return manifest, nil
}

// Render is the manifest as it is checked in.
func Render(manifest *pageaccess.Manifest) ([]byte, error) {
	out, err := sonic.ConfigStd.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(out, '\n'), nil
}

func operation(access *gqlext.OperationAccess) pageaccess.Operation {
	op := pageaccess.Operation{
		Name:     access.Name,
		Type:     string(access.Operation),
		Reads:    []string{},
		Declared: sorted(addKinds(nil, access.Declared)),
	}
	if access.Operation == ast.Query {
		op.Reads = sorted(addKinds(nil, access.Roots))
	}
	fields := make(map[string][]string, len(access.Linked))
	for _, kind := range access.Linked {
		fields[kind.String()] = []string{}
	}
	for _, s := range access.Selections {
		if s.Reach != refview.Nested {
			continue
		}
		field := s.Field
		if s.Part {
			field = s.Type + "." + s.Field
		}
		if !slices.Contains(fields[s.Kind.String()], field) {
			fields[s.Kind.String()] = append(fields[s.Kind.String()], field)
		}
	}
	kinds := make([]string, 0, len(fields))
	for kind := range fields {
		kinds = append(kinds, kind)
	}
	slices.Sort(kinds)
	op.Linked = make([]pageaccess.LinkedKind, 0, len(kinds))
	for _, kind := range kinds {
		op.Linked = append(
			op.Linked,
			pageaccess.LinkedKind{Kind: kind, Fields: sorted(fields[kind])},
		)
	}
	return op
}

func addKinds(into []string, kinds []permission.Resource) []string {
	for _, kind := range kinds {
		if !slices.Contains(into, kind.String()) {
			into = append(into, kind.String())
		}
	}
	return into
}

func sorted(values []string) []string {
	out := orEmpty(slices.Clone(values))
	slices.Sort(out)
	return out
}

func orEmpty[T any](values []T) []T {
	if values == nil {
		return []T{}
	}
	return values
}

// routeIndex finds the route a client path names and the checks its
// registration runs.
type routeIndex struct {
	routes []indexedRoute
}

type indexedRoute struct {
	method   string
	segments []string
	requires []pageaccess.Requirement
}

func newRouteIndex(routes []routelint.Route, checks []permissionlint.Pair) *routeIndex {
	byFile := make(map[string][]permissionlint.Pair)
	for _, check := range checks {
		if file, _, ok := splitPosition(check.Position); ok {
			byFile[file] = append(byFile[file], check)
		}
	}

	index := &routeIndex{routes: make([]indexedRoute, 0, len(routes))}
	for _, route := range routes {
		requires := make([]pageaccess.Requirement, 0, 1)
		for _, source := range route.Sources {
			file, err := filepath.Abs(source.File)
			if err != nil {
				continue
			}
			for _, check := range byFile[file] {
				_, line, _ := splitPosition(check.Position)
				if line < source.From || line > source.To {
					continue
				}
				requirement := pageaccess.Requirement{
					Resource:  check.Resource,
					Operation: check.Operation,
				}
				if !slices.Contains(requires, requirement) {
					requires = append(requires, requirement)
				}
			}
		}
		index.routes = append(index.routes, indexedRoute{
			method:   route.Method,
			segments: segments(strings.TrimPrefix(route.Path, apiBase)),
			requires: requires,
		})
	}
	return index
}

func splitPosition(position string) (file string, line int, ok bool) {
	at := strings.LastIndex(position, ":")
	if at < 0 {
		return "", 0, false
	}
	if _, err := fmt.Sscanf(position[at+1:], "%d", &line); err != nil {
		return "", 0, false
	}
	file, err := filepath.Abs(filepath.FromSlash(position[:at]))
	if err != nil {
		return "", 0, false
	}
	return file, line, true
}

func segments(path string) []string {
	return strings.Split(strings.Trim(path, "/"), "/")
}

// requires is what the route a client call reaches checks: the route of the
// same method whose segments the call's fill with the most literal matches.
func (r *routeIndex) requires(method, path string) []pageaccess.Requirement {
	call := segments(path)
	best, bestScore := -1, -1
	for i, route := range r.routes {
		if route.method != method {
			continue
		}
		score, ok := fill(route.segments, call)
		if ok && score > bestScore {
			best, bestScore = i, score
		}
	}
	if best < 0 {
		return []pageaccess.Requirement{}
	}
	out := slices.Clone(r.routes[best].requires)
	slices.SortFunc(out, func(a, b pageaccess.Requirement) int {
		return strings.Compare(a.Resource+":"+a.Operation, b.Resource+":"+b.Operation)
	})
	return out
}

func fill(route, call []string) (int, bool) {
	literals := 0
	for i, segment := range route {
		if strings.HasPrefix(segment, "*") {
			return literals, i < len(call)
		}
		if i >= len(call) {
			return 0, false
		}
		switch {
		case strings.HasPrefix(segment, ":"):
		case segment == call[i]:
			literals++
		default:
			return 0, false
		}
	}
	return literals, len(route) == len(call)
}
