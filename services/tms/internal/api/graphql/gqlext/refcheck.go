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

// RootChecks reports the kinds a root field's resolver checks read on (or an
// operation implying it) before it answers.
type RootChecks func(root, field string) []permission.Resource

// Selection is one field an operation selects on a record of a kind, and how
// the field guard will have reached that record. A field of a record that is
// part of another (a shipment's stop under a linked shipment) is the
// carrier's: Kind is the carrier's kind and Part is set.
type Selection struct {
	Path     string
	Type     string
	Field    string
	Kind     permission.Resource
	Reach    refview.Reach
	Part     bool
	NonNull  bool
	Declared bool
}

// OperationAccess is what one operation reads, as the field guard sees it at
// run time: the kinds its root resolvers return (Roots), the kinds it reaches
// under a self-scoped root (Own), the kinds it reaches as records another
// record links to (Linked), the kinds it declares with @requiresRead, and
// every field it selects on a record of a kind.
type OperationAccess struct {
	Name       string
	Operation  ast.Operation
	Roots      []permission.Resource
	Own        []permission.Resource
	Linked     []permission.Resource
	Declared   []permission.Resource
	Selections []Selection
}

// Operations walks operations as the field guard walks a response. A
// shipment, order, invoice or worker a mutation's payload carries is the
// operation's root record only when the mutation's resolver checks that kind
// (checks says which it does; nil means none), as the guard sees it for a
// caller who cannot read the kind. A field
// inside a reference view field (a state under a customer's address) is never
// decided on its own once the field holding it is shown, so only a record's
// own fields are selections.
func Operations(
	schema *ast.Schema,
	registry *permission.Registry,
	operations ast.OperationList,
	fragments ast.FragmentDefinitionList,
	checks RootChecks,
) ([]OperationAccess, error) {
	kinds, err := KindTypes(schema, registry)
	if err != nil {
		return nil, err
	}
	w := &linkWalker{
		schema:      schema,
		kinds:       kinds,
		views:       refview.New(registry),
		checks:      checks,
		scopedKinds: registry.DatabaseScopedResources(),
		fragments:   make(map[string]*ast.FragmentDefinition, len(fragments)),
	}
	for _, fragment := range fragments {
		w.fragments[fragment.Name] = fragment
	}
	out := make([]OperationAccess, 0, len(operations))
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
		w.access = &OperationAccess{
			Name:      op.Name,
			Operation: op.Operation,
			Declared:  RequiredReads(op),
		}
		w.selections(root, op.SelectionSet, linkFrame{rootType: root.Name}, op.Name, nil)
		out = append(out, *w.access)
	}

	return out, nil
}

// LinkedFields is every field the guard would withhold from a caller who
// cannot read its kind: a field outside the reference view of a record
// carried by another, any field of a part of one, or a shipment, order,
// invoice or worker carried by a mutation's payload when the mutation's
// resolver does not check that kind. Declared says whether the operation
// names the kind in @requiresRead.
func LinkedFields(
	schema *ast.Schema,
	registry *permission.Registry,
	operations ast.OperationList,
	fragments ast.FragmentDefinitionList,
	checks RootChecks,
) ([]LinkedField, error) {
	accesses, err := Operations(schema, registry, operations, fragments, checks)
	if err != nil {
		return nil, err
	}
	views := refview.New(registry)
	var found []LinkedField
	for i := range accesses {
		for _, s := range accesses[i].Selections {
			if !BeyondView(views, s) {
				continue
			}
			found = append(found, LinkedField{
				Operation: accesses[i].Name,
				Path:      s.Path,
				Type:      s.Type,
				Field:     s.Field,
				Kind:      s.Kind,
				NonNull:   s.NonNull,
				Declared:  s.Declared,
			})
		}
	}

	return found, nil
}

// BeyondView reports whether a selection is withheld from a caller who
// cannot read its kind.
func BeyondView(views *refview.Views, s Selection) bool {
	return s.Reach == refview.Nested && (s.Part || !views.InView(s.Kind, s.Field))
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
	schema      *ast.Schema
	kinds       map[string]permission.Resource
	views       *refview.Views
	checks      RootChecks
	scopedKinds []string
	fragments   map[string]*ast.FragmentDefinition
	op          *ast.OperationDefinition
	access      *OperationAccess
}

// linkFrame is where a walk is: the record it is in and how that record was
// reached, mirroring recordContext at run time.
type linkFrame struct {
	rootType  string
	rootField *ast.FieldDefinition
	direct    bool
	nested    bool
	carried   bool
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
	} else if w.payloadType(parent, frame) {
		next.carried = true
	} else if !wrapperType(parent) {
		next.nested = true
	}
	if kind, ok := w.kinds[child.Name]; ok {
		next.record = w.reach(kind, next)
		next.through = false
		w.reached(next.record)
	} else if frame.record != nil && frame.record.reach == refview.Nested && !frame.record.part &&
		w.kindOf(parent) && w.views.InView(frame.record.kind, def.Name) {
		next.through = true
	}
	w.selections(child, f.SelectionSet, next, at, nil)
}

func wrapperType(parent *ast.Definition) bool {
	return strings.HasSuffix(parent.Name, "Connection") || strings.HasSuffix(parent.Name, "Edge")
}

func (w *linkWalker) payloadType(parent *ast.Definition, frame linkFrame) bool {
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
	if frame.record == nil && Authorizes(frame.rootField, kind) {
		return &recordContext{kind: kind, reach: refview.Root}
	}
	if !frame.nested && (!frame.carried || !w.scoped(kind) || w.checked(frame, kind)) {
		return &recordContext{kind: kind, reach: refview.Root}
	}
	if frame.record != nil && w.views.PartOf(kind, frame.record.kind) {
		carrier := *frame.record
		carrier.part = carrier.part || carrier.kind != kind
		return &carrier
	}

	return &recordContext{kind: kind, reach: refview.Nested}
}

func (w *linkWalker) reached(record *recordContext) {
	switch {
	case record.reach == refview.Root:
		w.access.Roots = appendKind(w.access.Roots, record.kind)
	case record.reach == refview.SelfScoped:
		w.access.Own = appendKind(w.access.Own, record.kind)
	case !record.part:
		w.access.Linked = appendKind(w.access.Linked, record.kind)
	}
}

func appendKind(kinds []permission.Resource, kind permission.Resource) []permission.Resource {
	if slices.Contains(kinds, kind) {
		return kinds
	}
	return append(kinds, kind)
}

func (w *linkWalker) scoped(kind permission.Resource) bool {
	return slices.Contains(w.scopedKinds, kind.String())
}

func (w *linkWalker) checked(frame linkFrame, kind permission.Resource) bool {
	return w.checks != nil && frame.rootField != nil &&
		slices.Contains(w.checks(frame.rootType, frame.rootField.Name), kind)
}

func (w *linkWalker) check(
	parent *ast.Definition,
	def *ast.FieldDefinition,
	frame linkFrame,
	at string,
) {
	if !w.kindOf(parent) {
		return
	}
	record := frame.record
	w.access.Selections = append(w.access.Selections, Selection{
		Path:     strings.TrimPrefix(at, "."+w.op.Name+"."),
		Type:     parent.Name,
		Field:    def.Name,
		Kind:     record.kind,
		Reach:    record.reach,
		Part:     record.part,
		NonNull:  def.Type.NonNull,
		Declared: slices.Contains(w.access.Declared, record.kind),
	})
}
