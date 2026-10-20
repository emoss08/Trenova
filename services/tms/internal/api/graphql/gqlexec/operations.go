package gqlexec

import (
	"fmt"
	"sort"

	"github.com/vektah/gqlparser/v2/ast"
)

type operationWalk struct {
	reg       *Registry
	schema    *ast.Schema
	fragments map[string]*ast.FragmentDefinition
	problems  map[string]struct{}
	seenTypes map[string]struct{}
}

// validateOperations is validate for an executor built from some of the
// schema's shards: it checks that every type and field the given operations
// can reach has an executor, and that every input their arguments take can be
// decoded, rather than that the whole schema can be executed. The schema
// itself stays whole, so an operation is validated exactly as on HTTP.
func (r *Registry) validateOperations(
	schema *ast.Schema,
	docs []*ast.QueryDocument,
	resolvers map[string]func() any,
) error {
	walk := &operationWalk{
		reg:       r,
		schema:    schema,
		problems:  make(map[string]struct{}),
		seenTypes: make(map[string]struct{}),
	}
	for _, doc := range docs {
		walk.document(doc)
	}

	walk.resolvers(resolvers)

	if len(walk.problems) == 0 {
		return nil
	}

	problems := make([]string, 0, len(walk.problems))
	for problem := range walk.problems {
		problems = append(problems, problem)
	}
	sort.Strings(problems)

	return fmt.Errorf("gqlexec: executor cannot run its operations: %v", problems)
}

func (w *operationWalk) document(doc *ast.QueryDocument) {
	w.fragments = make(map[string]*ast.FragmentDefinition, len(doc.Fragments))
	for _, fragment := range doc.Fragments {
		w.fragments[fragment.Name] = fragment
	}

	for _, op := range doc.Operations {
		root := w.schema.Query
		if op.Operation == ast.Mutation {
			root = w.schema.Mutation
		}
		if root == nil {
			w.problem("operation " + op.Name + " has no root type")
			continue
		}
		for _, variable := range op.VariableDefinitions {
			w.input(variable.Type.Name())
		}
		w.selections(root.Name, op.SelectionSet)
	}
}

func (w *operationWalk) problem(text string) {
	w.problems[text] = struct{}{}
}

func (w *operationWalk) selections(typeName string, set ast.SelectionSet) {
	for _, selection := range set {
		switch s := selection.(type) {
		case *ast.Field:
			w.field(typeName, s)
		case *ast.InlineFragment:
			condition := typeName
			if s.TypeCondition != "" {
				condition = s.TypeCondition
			}
			w.typeExecutes(condition)
			w.selections(condition, s.SelectionSet)
		case *ast.FragmentSpread:
			fragment, ok := w.fragments[s.Name]
			if !ok {
				w.problem("fragment " + s.Name + " is not defined")
				continue
			}
			w.typeExecutes(fragment.TypeCondition)
			w.selections(fragment.TypeCondition, fragment.SelectionSet)
		}
	}
}

func (w *operationWalk) field(typeName string, field *ast.Field) {
	if field.Name == "__typename" {
		return
	}
	def := w.schema.Types[typeName]
	if def == nil {
		w.problem("type " + typeName + " is not in the schema")
		return
	}
	definition := def.Fields.ForName(field.Name)
	if definition == nil {
		w.problem("field " + typeName + "." + field.Name + " is not in the schema")
		return
	}
	if def.Kind == ast.Object {
		obj, ok := w.reg.objects[typeName]
		if !ok {
			w.problem("object " + typeName + " has no executor")
		} else if obj.field(field.Name) == nil {
			w.problem("field " + typeName + "." + field.Name + " has no executor")
		}
	}
	for _, argument := range definition.Arguments {
		w.input(argument.Type.Name())
	}

	child := definition.Type.Name()
	w.typeExecutes(child)
	if len(field.SelectionSet) > 0 {
		w.selections(child, field.SelectionSet)
	}
}

// typeExecutes checks that a value of the named type can be marshalled: an
// object needs its executor, an interface or union its abstract marshaller and
// every possible type's executor.
func (w *operationWalk) typeExecutes(name string) {
	if _, seen := w.seenTypes[name]; seen {
		return
	}
	w.seenTypes[name] = struct{}{}

	def := w.schema.Types[name]
	if def == nil {
		return
	}
	switch def.Kind {
	case ast.Object:
		if _, ok := w.reg.objects[name]; !ok {
			w.problem("object " + name + " has no executor")
		}
	case ast.Interface, ast.Union:
		if _, ok := w.reg.abstracts[name]; !ok {
			w.problem("abstract type " + name + " has no executor")
		}
		for _, possible := range w.schema.PossibleTypes[name] {
			w.typeExecutes(possible.Name)
		}
	case ast.Scalar, ast.Enum, ast.InputObject:
	}
}

// input checks that an argument's input type, and every input it holds, can
// be decoded. Scalars and enums are decoded by the shard that uses them.
func (w *operationWalk) input(name string) {
	if _, seen := w.seenTypes["input:"+name]; seen {
		return
	}
	w.seenTypes["input:"+name] = struct{}{}

	def := w.schema.Types[name]
	if def == nil || def.Kind != ast.InputObject {
		return
	}
	if _, ok := w.reg.inputs[name]; !ok {
		w.problem("input " + name + " has no decoder")
	}
	for _, field := range def.Fields {
		w.input(field.Type.Name())
	}
}

func (w *operationWalk) resolvers(resolvers map[string]func() any) {
	if resolvers == nil {
		return
	}
	for _, req := range w.reg.resolvers {
		get, ok := resolvers[req.Root]
		if !ok {
			w.problem("resolver root " + req.Root + " is not bound")
			continue
		}
		if resolver := get(); resolver != nil && !req.Check(resolver) {
			w.problem("resolver " + req.Root + " does not implement the fields its schema declares")
		}
	}
}
