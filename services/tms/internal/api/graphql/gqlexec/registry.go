package gqlexec

import (
	"context"
	"fmt"
	"sort"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

type (
	ResolveFunc   func(ctx context.Context, ec *Exec, obj any) (any, error)
	MarshalFunc   func(ctx context.Context, ec *Exec, sel ast.SelectionSet, v any) graphql.Marshaler
	ArgsFunc      func(ctx context.Context, ec *Exec, rawArgs map[string]any) (map[string]any, error)
	UnmarshalFunc func(ctx context.Context, ec *Exec, v any) (any, error)
)

type Field struct {
	Name       string
	NonNull    bool
	IsMethod   bool
	IsResolver bool
	Concurrent bool
	HasChild   bool
	ChildType  string
	ChildErr   error
	Args       ArgsFunc
	Resolve    ResolveFunc
	Marshal    MarshalFunc
	RootValue  func() any
}

type Object struct {
	Name         string
	Implementors []string
	Root         bool
	index        map[string]*Field
}

func (o *Object) field(name string) *Field {
	return o.index[name]
}

type Fields struct {
	Object string
	Fields []*Field
}

type Abstract struct {
	Name    string
	Marshal MarshalFunc
}

type Input struct {
	Name      string
	Unmarshal UnmarshalFunc
}

type ResolverRequirement struct {
	Root  string
	Check func(resolver any) bool
}

type Shard struct {
	Name      string
	Objects   []*Object
	Fields    []Fields
	Abstracts []Abstract
	Inputs    []Input
	Resolvers []ResolverRequirement
}

type Registry struct {
	objects   map[string]*Object
	abstracts map[string]MarshalFunc
	inputs    map[string]UnmarshalFunc
	resolvers []ResolverRequirement
}

func Compile(shards []*Shard) (*Registry, error) {
	reg := &Registry{
		objects:   make(map[string]*Object),
		abstracts: make(map[string]MarshalFunc),
		inputs:    make(map[string]UnmarshalFunc),
	}

	names := make(map[string]struct{}, len(shards))
	for _, s := range shards {
		if _, ok := names[s.Name]; ok {
			return nil, fmt.Errorf("gqlexec: shard %q registered twice", s.Name)
		}
		names[s.Name] = struct{}{}

		for _, obj := range s.Objects {
			if _, ok := reg.objects[obj.Name]; ok {
				return nil, fmt.Errorf("gqlexec: object %q registered twice", obj.Name)
			}
			obj.index = make(map[string]*Field)
			reg.objects[obj.Name] = obj
		}
		for _, a := range s.Abstracts {
			if _, ok := reg.abstracts[a.Name]; ok {
				return nil, fmt.Errorf("gqlexec: abstract type %q registered twice", a.Name)
			}
			reg.abstracts[a.Name] = a.Marshal
		}
		for _, in := range s.Inputs {
			if _, ok := reg.inputs[in.Name]; ok {
				return nil, fmt.Errorf("gqlexec: input %q registered twice", in.Name)
			}
			reg.inputs[in.Name] = in.Unmarshal
		}
		reg.resolvers = append(reg.resolvers, s.Resolvers...)
	}

	for _, s := range shards {
		for _, set := range s.Fields {
			obj, ok := reg.objects[set.Object]
			if !ok {
				return nil, fmt.Errorf(
					"gqlexec: shard %q extends unknown object %q",
					s.Name,
					set.Object,
				)
			}
			for _, f := range set.Fields {
				if _, ok := obj.index[f.Name]; ok {
					return nil, fmt.Errorf(
						"gqlexec: field %s.%s registered twice",
						set.Object,
						f.Name,
					)
				}
				obj.index[f.Name] = f
			}
		}
	}

	return reg, nil
}

func (r *Registry) validate(schema *ast.Schema, resolvers map[string]func() any) error {
	var problems []string

	for name, def := range schema.Types {
		if def.BuiltIn {
			continue
		}
		switch def.Kind {
		case ast.Object:
			obj, ok := r.objects[name]
			if !ok {
				problems = append(problems, "object "+name+" has no executor")
				continue
			}
			for _, f := range def.Fields {
				if obj.field(f.Name) == nil {
					problems = append(problems, "field "+name+"."+f.Name+" has no executor")
				}
			}
		case ast.Interface, ast.Union:
			if _, ok := r.abstracts[name]; !ok {
				problems = append(problems, "abstract type "+name+" has no executor")
			}
		}
	}

	if resolvers != nil {
		for _, req := range r.resolvers {
			get, ok := resolvers[req.Root]
			if !ok {
				problems = append(problems, "resolver root "+req.Root+" is not bound")
				continue
			}
			if resolver := get(); resolver != nil && !req.Check(resolver) {
				problems = append(
					problems,
					"resolver "+req.Root+" does not implement the fields its schema declares",
				)
			}
		}
	}

	if len(problems) == 0 {
		return nil
	}

	sort.Strings(problems)
	return fmt.Errorf("gqlexec: executable schema is incomplete: %v", problems)
}
