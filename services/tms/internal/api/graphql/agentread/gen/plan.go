package main

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2/ast"
)

const (
	modulePath    = "github.com/emoss08/trenova"
	generatedPath = modulePath + "/internal/api/graphql/generated"
	resolverPath  = modulePath + "/internal/api/graphql/resolver"
	coreVar       = "core"
)

type executorPlan struct {
	Shards    []shardImport
	Imports   []importLine
	Params    []paramField
	Core      string
	Deps      []depsBlock
	Objects   []*objectResolver
	Queries   []*rootResolver
	Mutations []*rootResolver
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

func planExecutor(schema *ast.Schema, opts *options, bindings []*binding) (*executorPlan, error) {
	r := newReach(schema)
	for _, b := range bindings {
		r.operation(b.Operation.Query)
	}
	closeOver(schema, r.shards)

	w, err := loadWiring(opts.resolver)
	if err != nil {
		return nil, err
	}

	plan := &executorPlan{}
	shards := sortedKeys(r.shards)
	required := make(map[string]struct{})
	for _, shard := range shards {
		plan.Shards = append(plan.Shards, shardImport{
			Package: shard + "exec",
			Path:    generatedPath + "/exec/" + shard + "exec",
		})
		roots, rootsErr := requiredRoots(opts.generated, shard)
		if rootsErr != nil {
			return nil, rootsErr
		}
		for _, root := range roots {
			required[root] = struct{}{}
		}
	}

	usedDeps := make(map[string]struct{})
	var problems []string
	for _, root := range sortedKeys(required) {
		switch root {
		case rootQuery, rootMutation:
			for _, shard := range shards {
				alias := shard + root
				if resolver, ok := w.roots[alias]; ok {
					if root == rootQuery {
						plan.Queries = append(plan.Queries, resolver)
					} else {
						plan.Mutations = append(plan.Mutations, resolver)
					}
					usedDeps[resolver.Deps] = struct{}{}
				}
			}
		default:
			resolver, ok := w.objects[root]
			if !ok {
				problems = append(problems, "no resolver root builds "+root)
				continue
			}
			plan.Objects = append(plan.Objects, resolver)
			usedDeps[resolver.Deps] = struct{}{}
		}
	}

	params := make(map[string]struct{})
	packages := make(map[string]struct{})
	for _, p := range paramsUsed(w.services.core) {
		params[p] = struct{}{}
	}
	for _, alias := range packagesUsed(w.services.core) {
		packages[alias] = struct{}{}
	}
	plan.Core = w.services.core

	for _, name := range sortedKeys(usedDeps) {
		literal, ok := w.deps[name]
		if !ok {
			problems = append(problems, "no Deps literal named "+name)
			continue
		}
		block := depsBlock{Var: name, Package: literal.Package}
		packages[literal.Package] = struct{}{}
		for _, field := range literal.Fields {
			if field.Service == coreField {
				block.Fields = append(block.Fields, depsValue{Name: field.Name, Expr: coreVar})
				continue
			}
			expr, known := w.services.exprs[field.Service]
			if !known {
				problems = append(problems, "newServices does not build "+field.Service)
				continue
			}
			for _, p := range paramsUsed(expr) {
				params[p] = struct{}{}
			}
			for _, alias := range packagesUsed(expr) {
				packages[alias] = struct{}{}
			}
			block.Fields = append(block.Fields, depsValue{Name: field.Name, Expr: expr})
		}
		plan.Deps = append(plan.Deps, block)
	}

	for _, name := range sortedKeys(params) {
		field, ok := w.services.params[name]
		if !ok {
			problems = append(problems, "Params has no field "+name)
			continue
		}
		plan.Params = append(plan.Params, field)
		for _, alias := range packagesUsed(field.Type) {
			packages[alias] = struct{}{}
		}
	}
	for _, o := range plan.Objects {
		packages[o.Package] = struct{}{}
	}
	for _, q := range slices.Concat(plan.Queries, plan.Mutations) {
		packages[q.Package] = struct{}{}
	}

	for _, alias := range sortedKeys(packages) {
		path, ok := w.services.imports[alias]
		if !ok {
			path, ok = w.imports[alias]
		}
		if !ok {
			continue
		}
		plan.Imports = append(plan.Imports, importLine{Alias: alias, Path: path})
	}
	sort.Slice(plan.Imports, func(i, j int) bool { return plan.Imports[i].Path < plan.Imports[j].Path })

	if len(problems) > 0 {
		return nil, fmt.Errorf("the read executor cannot be wired:\n%s", strings.Join(problems, "\n"))
	}

	return plan, nil
}
