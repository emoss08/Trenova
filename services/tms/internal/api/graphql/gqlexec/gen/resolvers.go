package main

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/99designs/gqlgen/codegen"
	"github.com/99designs/gqlgen/codegen/config"
	"github.com/99designs/gqlgen/codegen/templates"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	resolverSuffix       = "resolver"
	baseResolverPackage  = "base"
	resolverTypesFile    = "resolvers.generated.go"
	resolverRootFilename = "resolver.generated.go"
)

type resolverLayout struct {
	Data        *codegen.Data
	BaseImport  string
	ExecImport  string
	Domains     []*resolverDomain
	Entries     []*rootEntry
	domainIndex map[string]*resolverDomain
}

type resolverDomain struct {
	Shard      string
	Package    string
	ImportPath string
	Dir        string
	StubFile   string
	BaseImport string
	DepsFields []depField
	Types      []*resolverType
	typeIndex  map[string]*resolverType
}

type resolverType struct {
	Object *codegen.Object
	Name   string
	Fields []*codegen.Field
}

type rootEntry struct {
	Accessor string
	Iface    string
	Field    string
	Type     string
	Parts    []*rootPart
}

type rootPart struct {
	Domain *resolverDomain
	Alias  string
}

func (e *rootEntry) Composite() bool {
	return len(e.Parts) > 1
}

func buildResolverLayout(data *codegen.Data) *resolverLayout {
	cfg := data.Config
	l := &resolverLayout{
		Data:        data,
		BaseImport:  path.Join(cfg.Resolver.ImportPath(), baseResolverPackage),
		ExecImport:  cfg.Exec.ImportPath(),
		domainIndex: make(map[string]*resolverDomain),
	}

	objects := make([]*codegen.Object, 0, len(data.Objects)+len(data.Inputs))
	for _, obj := range data.Objects {
		if obj.HasResolvers() {
			objects = append(objects, obj)
		}
	}
	for _, in := range data.Inputs {
		if in.HasResolvers() {
			objects = append(objects, in)
		}
	}

	for _, obj := range objects {
		entry := &rootEntry{Accessor: templates.UcFirst(obj.Name), Type: resolverTypeName(obj)}
		entry.Iface = entry.Type
		entry.Field = templates.LcFirst(entry.Accessor)

		for _, f := range obj.Fields {
			if !f.IsResolver {
				continue
			}
			pos := obj.Position
			if f.FieldDefinition != nil && f.Position != nil {
				pos = f.Position
			}
			d := l.domain(cfg, pos)
			t, ok := d.typeIndex[entry.Type]
			if !ok {
				t = &resolverType{Object: obj, Name: entry.Type}
				d.typeIndex[entry.Type] = t
				d.Types = append(d.Types, t)
				entry.Parts = append(entry.Parts, &rootPart{
					Domain: d,
					Alias:  d.Shard + templates.UcFirst(obj.Name),
				})
			}
			t.Fields = append(t.Fields, f)
		}

		sort.Slice(entry.Parts, func(i, j int) bool {
			return entry.Parts[i].Domain.Shard < entry.Parts[j].Domain.Shard
		})
		l.Entries = append(l.Entries, entry)
	}

	for _, d := range l.domainIndex {
		sort.Slice(d.Types, func(i, j int) bool { return d.Types[i].Name < d.Types[j].Name })
		l.Domains = append(l.Domains, d)
	}
	sort.Slice(l.Domains, func(i, j int) bool { return l.Domains[i].Shard < l.Domains[j].Shard })

	return l
}

func resolverTypeName(obj *codegen.Object) string {
	if obj.Kind == ast.InputObject {
		return obj.Name + "Resolver"
	}
	return templates.UcFirst(obj.Name) + "Resolver"
}

func (l *resolverLayout) domain(cfg *config.Config, pos *ast.Position) *resolverDomain {
	source := "prelude.graphql"
	if pos != nil && pos.Src != nil {
		source = pos.Src.Name
	}
	shard := shardName(source)
	if d, ok := l.domainIndex[shard]; ok {
		return d
	}

	pkg := shard + resolverSuffix
	schemaBase := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	stub := strings.ReplaceAll(cfg.Resolver.FilenameTemplate, "{name}", schemaBase)
	if stub == "" {
		stub = schemaBase + ".resolvers.go"
	}

	d := &resolverDomain{
		Shard:      shard,
		Package:    pkg,
		ImportPath: path.Join(cfg.Resolver.ImportPath(), pkg),
		Dir:        filepath.Join(cfg.Resolver.Dir(), pkg),
		StubFile:   stub,
		BaseImport: l.BaseImport,
		typeIndex:  make(map[string]*resolverType),
	}
	l.domainIndex[shard] = d
	return d
}

func writeResolvers(cfg *config.Config, l *resolverLayout) error {
	if err := removeStaleResolverTypes(cfg, l); err != nil {
		return err
	}

	svc, err := loadServices(cfg.Resolver.Dir())
	if err != nil {
		return err
	}

	for _, d := range l.Domains {
		if err = os.MkdirAll(d.Dir, 0o755); err != nil {
			return fmt.Errorf("create %s: %w", d.Dir, err)
		}
		if err = syncStubs(cfg, d); err != nil {
			return fmt.Errorf("sync resolvers for %s: %w", d.Shard, err)
		}
		if d.DepsFields, err = svc.usedBy(d.Dir); err != nil {
			return err
		}
		if err = checkShadowedDeps(d); err != nil {
			return err
		}
		if err = writeDomainTypes(d); err != nil {
			return fmt.Errorf("write resolver types for %s: %w", d.Shard, err)
		}
	}

	if err = templates.Render(templates.Options{
		PackageName:     cfg.Resolver.Package,
		Filename:        filepath.Join(cfg.Resolver.Dir(), resolverRootFilename),
		Template:        resolverRootTemplate,
		Data:            l,
		GeneratedHeader: true,
		Packages:        cfg.Packages,
		PruneOptions:    cfg.GetPruneOptions(),
	}); err != nil {
		return fmt.Errorf("render resolver root: %w", err)
	}

	return nil
}

func removeStaleResolverTypes(cfg *config.Config, l *resolverLayout) error {
	entries, err := os.ReadDir(cfg.Resolver.Dir())
	if err != nil {
		return fmt.Errorf("read %s: %w", cfg.Resolver.Dir(), err)
	}
	live := make(map[string]struct{}, len(l.Domains))
	for _, d := range l.Domains {
		live[d.Package] = struct{}{}
	}
	for _, e := range entries {
		if !e.IsDir() || !strings.HasSuffix(e.Name(), resolverSuffix) {
			continue
		}
		if _, ok := live[e.Name()]; ok {
			continue
		}
		stale := filepath.Join(cfg.Resolver.Dir(), e.Name(), resolverTypesFile)
		if err = os.Remove(stale); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove %s: %w", stale, err)
		}
	}
	return nil
}

var (
	//go:embed resolver_root.gotpl
	resolverRootTemplate string
	//go:embed resolver_stubs.gotpl
	resolverStubsTemplate string
)

// checkShadowedDeps refuses a resolver method that has the same name as a
// dependency its package uses. The method would hide the promoted field, so
// r.X in that package means the method and the dependency is unreachable.
func checkShadowedDeps(d *resolverDomain) error {
	deps := make(map[string]struct{}, len(d.DepsFields))
	for _, f := range d.DepsFields {
		deps[f.Name] = struct{}{}
	}
	var errs []error
	for _, t := range d.Types {
		for _, f := range t.Fields {
			if _, ok := deps[f.GoFieldName]; ok {
				errs = append(errs, fmt.Errorf(
					"%s.%s hides the %s.%s dependency; rename the field in %s",
					t.Name, f.GoFieldName, servicesType, f.GoFieldName, servicesFile,
				))
			}
		}
	}
	return errors.Join(errs...)
}
