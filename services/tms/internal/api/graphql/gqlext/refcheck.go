package gqlext

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/api/graphql/selfscoped"
	"github.com/emoss08/trenova/internal/modules/identity/app/refview"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	requiresReadDirective = "requiresRead"
	requiresReadArgument  = "kind"
)

// LinkedField is one field an operation selects on a record of a kind
// carried by another, outside that kind's reference view.
type LinkedField struct {
	Operation string
	Path      string
	Type      string
	Field     string
	Kind      permission.Resource
	NonNull   bool
	Declared  bool
}

func (f LinkedField) String() string {
	return fmt.Sprintf("%s selects %s.%s at %s", f.Operation, f.Type, f.Field, f.Path)
}

// LinkedFields walks operations as the field guard walks a response and
// returns every field the guard would withhold from a caller who cannot read
// its kind: a field outside the reference view of a record carried by
// another. A field inside one of those (a state under a withheld address) is
// never resolved once its parent is withheld, so only the record's own fields
// are returned. Declared says whether the operation names the kind in
// @requiresRead.
func LinkedFields(
	schema *ast.Schema,
	registry *permission.Registry,
	operations ast.OperationList,
	fragments ast.FragmentDefinitionList,
) ([]LinkedField, error) {
	kinds, err := KindTypes(schema, registry)
	if err != nil {
		return nil, err
	}
	w := &linkWalker{
		schema:    schema,
		kinds:     kinds,
		views:     refview.New(registry),
		fragments: make(map[string]*ast.FragmentDefinition, len(fragments)),
	}
	for _, fragment := range fragments {
		w.fragments[fragment.Name] = fragment
	}
	for _, op := range operations {
		var root *ast.Definition
		switch op.Operation {
		case ast.Query:
			root = schema.Query
		case ast.Mutation:
			root = schema.Mutation
		case ast.Subscription:
			root = schema.Subscription
		}
		if root == nil {
			continue
		}
		w.op = op
		w.declared = RequiredReads(op)
		w.selections(root, op.SelectionSet, linkFrame{rootType: root.Name}, op.Name, nil)
	}

	return w.found, nil
}

// RequiredReads is the kinds an operation declares with @requiresRead.
func RequiredReads(op *ast.OperationDefinition) []permission.Resource {
	var out []permission.Resource
	for _, directive := range op.Directives {
		if directive.Name != requiresReadDirective {
			continue
		}
		if argument := directive.Arguments.ForName(requiresReadArgument); argument != nil &&
			argument.Value != nil {
			out = append(out, permission.Resource(argument.Value.Raw))
		}
	}
	return out
}

type linkWalker struct {
	schema    *ast.Schema
	kinds     map[string]permission.Resource
	views     *refview.Views
	fragments map[string]*ast.FragmentDefinition
	op        *ast.OperationDefinition
	declared  []permission.Resource
	found     []LinkedField
}

// linkFrame is where a walk is: the record it is in and how that record was
// reached, mirroring recordContext at run time.
type linkFrame struct {
	rootType  string
	rootField *ast.FieldDefinition
	direct    bool
	nested    bool
	through   bool
	record    *recordContext
}

func (w *linkWalker) selections(
	parent *ast.Definition,
	set ast.SelectionSet,
	frame linkFrame,
	path string,
	seen []string,
) {
	for _, selection := range set {
		switch s := selection.(type) {
		case *ast.Field:
			w.field(parent, s, frame, path)
		case *ast.InlineFragment:
			on := parent
			if s.TypeCondition != "" {
				on = w.schema.Types[s.TypeCondition]
			}
			w.selections(on, s.SelectionSet, frame, path, seen)
		case *ast.FragmentSpread:
			fragment := w.fragments[s.Name]
			if fragment == nil || slices.Contains(seen, s.Name) {
				continue
			}
			on := w.schema.Types[fragment.TypeCondition]
			w.selections(on, fragment.SelectionSet, frame, path, append(slices.Clone(seen), s.Name))
		}
	}
}

func (w *linkWalker) field(parent *ast.Definition, f *ast.Field, frame linkFrame, path string) {
	if parent == nil || strings.HasPrefix(f.Name, "__") {
		return
	}
	def := parent.Fields.ForName(f.Name)
	if def == nil {
		return
	}
	at := path + "." + f.Name
	if frame.record != nil && f.Name != recordIDField && !frame.through {
		w.check(parent, def, frame, at)
	}
	child := w.schema.Types[def.Type.Name()]
	if child == nil || len(f.SelectionSet) == 0 {
		return
	}

	next := frame
	next.direct = false
	if isRoot(parent.Name) {
		next = linkFrame{rootType: parent.Name, rootField: def, direct: true}
	} else if !w.wrapperType(parent, frame) {
		next.nested = true
	}
	if kind, ok := w.kinds[child.Name]; ok {
		next.record = w.reach(kind, next)
		next.through = false
	} else if frame.record != nil && frame.record.reach == refview.Nested && !frame.record.part &&
		w.kindOf(parent) && w.views.InView(frame.record.kind, def.Name) {
		next.through = true
	}
	w.selections(child, f.SelectionSet, next, at, nil)
}

func (w *linkWalker) wrapperType(parent *ast.Definition, frame linkFrame) bool {
	if strings.HasSuffix(parent.Name, "Connection") || strings.HasSuffix(parent.Name, "Edge") {
		return true
	}
	if frame.rootType != rootMutation || !frame.direct {
		return false
	}
	_, record := w.kinds[parent.Name]

	return !record
}

func (w *linkWalker) kindOf(def *ast.Definition) bool {
	_, ok := w.kinds[def.Name]
	return ok
}

func (w *linkWalker) reach(kind permission.Resource, frame linkFrame) *recordContext {
	if frame.rootField != nil && selfscoped.Root(frame.rootType, frame.rootField.Name) {
		return &recordContext{kind: kind, reach: refview.SelfScoped}
	}
	if !frame.nested || (frame.record == nil && Authorizes(frame.rootField, kind)) {
		return &recordContext{kind: kind, reach: refview.Root}
	}
	if frame.record != nil && w.views.PartOf(kind, frame.record.kind) {
		carrier := *frame.record
		carrier.part = carrier.part || carrier.kind != kind
		return &carrier
	}

	return &recordContext{kind: kind, reach: refview.Nested}
}

func (w *linkWalker) check(
	parent *ast.Definition,
	def *ast.FieldDefinition,
	frame linkFrame,
	at string,
) {
	record := frame.record
	if record.reach != refview.Nested || !w.kindOf(parent) {
		return
	}
	if !record.part && w.views.InView(record.kind, def.Name) {
		return
	}
	w.found = append(w.found, LinkedField{
		Operation: w.op.Name,
		Path:      strings.TrimPrefix(at, "."+w.op.Name+"."),
		Type:      parent.Name,
		Field:     def.Name,
		Kind:      record.kind,
		NonNull:   def.Type.NonNull,
		Declared:  slices.Contains(w.declared, record.kind),
	})
}
