package main

import (
	"fmt"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

const (
	modulePath    = "github.com/emoss08/trenova"
	generatedPath = modulePath + "/internal/api/graphql/generated"
	coreVar       = "core"
)

type executorPlan struct {
	Shards     []shardImport
	Imports    []importLine
	Params     []paramField
	Core       string
	Deps       []depsBlock
	Objects    []*objectResolver
	Composites []*compositePlan
}

// compositePlan is an object the read executor builds as a composite, as
// the full root does, of the included packages' shares only.
type compositePlan struct {
	Object  string
	Var     string
	Type    string
	Members []*rootResolver
}

type shardImport struct {
	Package string
	Path    string
}

type importLine struct {
	Alias string
	Path  string
}

type depsBlock struct {
	Var     string
	Package string
	Fields  []depsValue
}

type depsValue struct {
	Name string
	Expr string
}

// planner works out what the read executor holds: the shards the bound
// operations reach, the resolvers those shards ask for, and the services
// and imports those resolvers need.
type planner struct {
	w        *wiring
	plan     *executorPlan
	deps     map[string]struct{}
	params   map[string]struct{}
	packages map[string]struct{}
	problems []string
}

func planExecutor(schema *ast.Schema, opts *options, bindings []*binding) (*executorPlan, error) {
	r := newReach(schema)
	for _, b := range bindings {
		r.operation(b.Operation.Query)
		for _, beside := range b.Alongside {
			r.operation(beside.Operation.Query)
		}
		for _, instead := range b.Instead {
			r.operation(instead.Operation.Query)
		}
	}
	closeOver(schema, r.shards)

	w, err := loadWiring(opts.resolver)
	if err != nil {
		return nil, err
	}
	p := &planner{
		w:        w,
		plan:     &executorPlan{Core: w.services.core},
		deps:     make(map[string]struct{}),
		params:   make(map[string]struct{}),
		packages: make(map[string]struct{}),
	}

	shards := sortedKeys(r.shards)
	required, err := p.addShards(opts.generated, shards)
	if err != nil {
		return nil, err
	}
	p.addResolvers(shards, required)
	p.use(w.services.core)
	p.addDeps()
	p.addParams()
	p.addImports()

	if len(p.problems) > 0 {
		return nil, fmt.Errorf("the read executor cannot be wired:\n%s",
			strings.Join(p.problems, "\n"))
	}

	return p.plan, nil
}

func (p *planner) addShards(generatedDir string, shards []string) ([]string, error) {
	required := make(map[string]struct{})
	for _, shard := range shards {
		p.plan.Shards = append(p.plan.Shards, shardImport{
			Package: shard + "exec",
			Path:    generatedPath + "/exec/" + shard + "exec",
		})
		roots, err := requiredRoots(generatedDir, shard)
		if err != nil {
			return nil, err
		}
		for _, root := range roots {
			required[root] = struct{}{}
		}
	}

	return sortedKeys(required), nil
}

// addResolvers binds every resolver root a shard asks for: a composite
// (Query, Mutation, or an object whose resolvers span schema files) of the
// included packages' shares, and any other object's resolver as the full
// root builds it.
func (p *planner) addResolvers(shards, required []string) {
	for _, root := range required {
		if p.w.composites[root] {
			p.addComposite(shards, root)
			continue
		}
		resolver, ok := p.w.objects[root]
		if !ok {
			p.problems = append(p.problems, "no resolver root builds "+root)
			continue
		}
		p.plan.Objects = append(p.plan.Objects, resolver)
		p.deps[resolver.Deps] = struct{}{}
		p.packages[resolver.Package] = struct{}{}
	}
}

func (p *planner) addComposite(shards []string, root string) {
	lower := strings.ToLower(root[:1]) + root[1:]
	composite := &compositePlan{
		Object: root,
		Var:    lower,
		Type:   lower + "Resolver",
	}
	for _, shard := range shards {
		resolver, ok := p.w.roots[shard+root]
		if !ok {
			continue
		}
		composite.Members = append(composite.Members, resolver)
		p.deps[resolver.Deps] = struct{}{}
		p.packages[resolver.Package] = struct{}{}
	}
	if len(composite.Members) == 0 {
		p.problems = append(p.problems, "no included package builds a share of "+root)
		return
	}
	p.plan.Composites = append(p.plan.Composites, composite)
}

// use records the Params fields and packages an expression copied from
// newServices reads.
func (p *planner) use(expr string) {
	for _, field := range paramsUsed(expr) {
		p.params[field] = struct{}{}
	}
	for _, alias := range packagesUsed(expr) {
		p.packages[alias] = struct{}{}
	}
}

func (p *planner) addDeps() {
	for _, name := range sortedKeys(p.deps) {
		literal, ok := p.w.deps[name]
		if !ok {
			p.problems = append(p.problems, "no Deps literal named "+name)
			continue
		}
		block := depsBlock{Var: name, Package: literal.Package}
		p.packages[literal.Package] = struct{}{}
		for _, field := range literal.Fields {
			if field.Service == coreField {
				block.Fields = append(block.Fields, depsValue{Name: field.Name, Expr: coreVar})
				continue
			}
			expr, known := p.w.services.exprs[field.Service]
			if !known {
				p.problems = append(p.problems, "newServices does not build "+field.Service)
				continue
			}
			p.use(expr)
			block.Fields = append(block.Fields, depsValue{Name: field.Name, Expr: expr})
		}
		p.plan.Deps = append(p.plan.Deps, block)
	}
}

func (p *planner) addParams() {
	for _, name := range sortedKeys(p.params) {
		field, ok := p.w.services.params[name]
		if !ok {
			p.problems = append(p.problems, "Params has no field "+name)
			continue
		}
		p.plan.Params = append(p.plan.Params, field)
		for _, alias := range packagesUsed(field.Type) {
			p.packages[alias] = struct{}{}
		}
	}
}

func (p *planner) addImports() {
	for _, alias := range sortedKeys(p.packages) {
		path, ok := p.w.services.imports[alias]
		if !ok {
			path, ok = p.w.imports[alias]
		}
		if !ok {
			continue
		}
		p.plan.Imports = append(p.plan.Imports, importLine{Alias: alias, Path: path})
	}
	sort.Slice(p.plan.Imports, func(i, j int) bool {
		return p.plan.Imports[i].Path < p.plan.Imports[j].Path
	})
}
