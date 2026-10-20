package main

import (
	"path/filepath"
	"strings"
	"unicode"

	"github.com/vektah/gqlparser/v2/ast"
)

const preludeShard = "prelude"

// shardName is the executor package a schema file's types and fields live in,
// as gqlexec's generator names it.
func shardName(source string) string {
	base := strings.TrimSuffix(filepath.Base(source), filepath.Ext(source))
	var b strings.Builder
	for _, r := range strings.ToLower(base) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 || unicode.IsDigit(rune(b.String()[0])) {
		return "schema" + b.String()
	}

	return b.String()
}

func shardOf(pos *ast.Position) string {
	if pos == nil || pos.Src == nil || pos.Src.BuiltIn {
		return preludeShard
	}

	return shardName(pos.Src.Name)
}

// reach is every shard an operation needs to run: those that define a type,
// field, input or abstract type it can reach.
type reach struct {
	schema    *ast.Schema
	fragments map[string]*ast.FragmentDefinition
	shards    map[string]struct{}
	seen      map[string]struct{}
}

func newReach(schema *ast.Schema) *reach {
	r := &reach{
		schema: schema,
		shards: map[string]struct{}{preludeShard: {}},
		seen:   make(map[string]struct{}),
	}
	if schema.Query != nil {
		r.shards[shardOf(schema.Query.Position)] = struct{}{}
	}
	if schema.Mutation != nil {
		r.shards[shardOf(schema.Mutation.Position)] = struct{}{}
	}

	return r
}

func (r *reach) operation(doc *ast.QueryDocument) {
	r.fragments = make(map[string]*ast.FragmentDefinition, len(doc.Fragments))
	for _, fragment := range doc.Fragments {
		r.fragments[fragment.Name] = fragment
	}
	for _, op := range doc.Operations {
		for _, variable := range op.VariableDefinitions {
			r.input(variable.Type.Name())
		}
		r.selections(r.schema.Query.Name, op.SelectionSet)
	}
}

func (r *reach) selections(typeName string, set ast.SelectionSet) {
	for _, selection := range set {
		switch s := selection.(type) {
		case *ast.Field:
			r.field(typeName, s)
		case *ast.InlineFragment:
			condition := typeName
			if s.TypeCondition != "" {
				condition = s.TypeCondition
			}
			r.typ(condition)
			r.selections(condition, s.SelectionSet)
		case *ast.FragmentSpread:
			if fragment, ok := r.fragments[s.Name]; ok {
				r.typ(fragment.TypeCondition)
				r.selections(fragment.TypeCondition, fragment.SelectionSet)
			}
		}
	}
}

func (r *reach) field(typeName string, field *ast.Field) {
	if strings.HasPrefix(field.Name, "__") {
		return
	}
	def := r.schema.Types[typeName]
	if def == nil {
		return
	}
	definition := def.Fields.ForName(field.Name)
	if definition == nil {
		return
	}
	r.typ(typeName)
	r.shards[shardOf(definition.Position)] = struct{}{}
	for _, argument := range definition.Arguments {
		r.input(argument.Type.Name())
	}
	child := definition.Type.Name()
	r.typ(child)
	if len(field.SelectionSet) > 0 {
		r.selections(child, field.SelectionSet)
	}
}

func (r *reach) typ(name string) {
	if _, seen := r.seen["type:"+name]; seen {
		return
	}
	r.seen["type:"+name] = struct{}{}

	def := r.schema.Types[name]
	if def == nil {
		return
	}
	switch def.Kind {
	case ast.Object:
		r.shards[shardOf(def.Position)] = struct{}{}
	case ast.Interface, ast.Union:
		r.shards[shardOf(def.Position)] = struct{}{}
		for _, possible := range r.schema.PossibleTypes[name] {
			r.typ(possible.Name)
		}
	case ast.Scalar, ast.Enum, ast.InputObject:
	}
}

func (r *reach) input(name string) {
	if _, seen := r.seen["input:"+name]; seen {
		return
	}
	r.seen["input:"+name] = struct{}{}

	def := r.schema.Types[name]
	if def == nil || def.Kind != ast.InputObject {
		return
	}
	r.shards[shardOf(def.Position)] = struct{}{}
	for _, field := range def.Fields {
		r.input(field.Type.Name())
	}
}

// closeOver adds, for every shard included, the shards defining the objects
// it extends: gqlexec refuses a shard that extends an object no shard
// defines.
func closeOver(schema *ast.Schema, shards map[string]struct{}) {
	extends := make(map[string]map[string]struct{})
	for _, def := range schema.Types {
		if def.Kind != ast.Object || def.BuiltIn {
			continue
		}
		home := shardOf(def.Position)
		for _, field := range def.Fields {
			if s := shardOf(field.Position); s != home {
				if extends[s] == nil {
					extends[s] = make(map[string]struct{})
				}
				extends[s][home] = struct{}{}
			}
		}
	}
	for changed := true; changed; {
		changed = false
		for shard := range shards {
			for home := range extends[shard] {
				if _, ok := shards[home]; !ok {
					shards[home] = struct{}{}
					changed = true
				}
			}
		}
	}
}
