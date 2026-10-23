package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/bytedance/sonic"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/shared/stringutils"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"gopkg.in/yaml.v3"
)

const (
	verbList = "list"
	verbGet  = "get"

	deskPrefix   = "Desk"
	tableSuffix  = "Table"
	connection   = "Connection"
	edgesField   = "edges"
	nodeField    = "node"
	idArgument   = "id"
	itemsField   = "items"
	pageInfo     = "pageInfo"
	readingSplit = "_"

	listInput         = "DataTableConnectionInput"
	includeTotalCount = "includeTotalCount"
	idType            = "ID"
)

// supplied are the variables a tool of each verb sends itself, by name and
// type: a list builds the table input and turns the total count off, a get
// sends the record's id. Any other scalar, enum or id variable is a
// parameter of the tool's own (see unsupplied).
var supplied = map[string]map[string]string{
	verbList: {"": listInput, includeTotalCount: "Boolean"},
	verbGet:  {idArgument: idType},
}

// read is one entry of reads.yml: a tool that reads through GraphQL. The
// operation is worked out by the rule; naming one that differs needs a
// reason, which is the record of why the rule is wrong for it.
type read struct {
	Tool      string `yaml:"tool"`
	Operation string `yaml:"operation,omitempty"`
	Reason    string `yaml:"reason,omitempty"`
	// Entity is the Go record whose filter configuration the resolver
	// compiles filters with, named when the GraphQL type binds to a
	// generated model rather than to the entity itself.
	Entity string `yaml:"entity,omitempty"`
	// Alongside are operations a tool reads beside its own, each for records
	// another call needs (the templates a checklist can be started from, the
	// tolerance a drift list is judged by). Each is proven as a named binding
	// is.
	Alongside []string `yaml:"alongside,omitempty"`
	// Instead are operations a list reads in place of its own when a call
	// names their own variables (the sync state of documents named by id).
	Instead []string `yaml:"instead,omitempty"`
}

type readsFile struct {
	Reads []read `yaml:"reads"`
}

func loadReads(path string) ([]read, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var file readsFile
	if err = yaml.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	seen := make(map[string]struct{}, len(file.Reads))
	for _, r := range file.Reads {
		if _, dup := seen[r.Tool]; dup {
			return nil, fmt.Errorf("%s lists %s twice", path, r.Tool)
		}
		seen[r.Tool] = struct{}{}
	}
	sort.Slice(file.Reads, func(i, j int) bool { return file.Reads[i].Tool < file.Reads[j].Tool })

	return file.Reads, nil
}

func loadSchema(dir string) (*ast.Schema, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.graphqls"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	sources := make([]*ast.Source, 0, len(paths))
	for _, path := range paths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, readErr
		}
		sources = append(sources, &ast.Source{Name: filepath.Base(path), Input: string(content)})
	}
	schema, gqlErr := gqlparser.LoadSchema(sources...)
	if gqlErr != nil {
		return nil, fmt.Errorf("load schema: %w", gqlErr)
	}

	return schema, nil
}

// operation is one persisted query: what the client sends and what it
// selects at its root.
type operation struct {
	Name     string
	Hash     string
	Document string
	Query    *ast.QueryDocument
	Roots    []string
}

func loadOperations(schema *ast.Schema, path string) (map[string]*operation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	var manifest map[string]string
	if err = sonic.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}

	out := make(map[string]*operation, len(manifest))
	for hash, text := range manifest {
		doc, gqlErr := gqlparser.LoadQueryWithRules(schema, text, nil)
		if gqlErr != nil {
			return nil, fmt.Errorf("persisted operation %s does not validate: %w", hash, gqlErr)
		}
		if len(doc.Operations) != 1 || doc.Operations[0].Operation != ast.Query {
			continue
		}
		op := doc.Operations[0]
		roots := make([]string, 0, len(op.SelectionSet))
		for _, selection := range op.SelectionSet {
			if field, ok := selection.(*ast.Field); ok && !strings.HasPrefix(field.Name, "__") {
				roots = append(roots, field.Name)
			}
		}
		out[op.Name] = &operation{
			Name: op.Name, Hash: hash, Document: text, Query: doc, Roots: roots,
		}
	}

	return out, nil
}

// binding is a tool and the operation it reads through.
type binding struct {
	Tool      string
	Verb      string
	Operation *operation
	RootField *ast.FieldDefinition
	// Node is the record type the tool reads: a list's connection node, or
	// a get's result.
	Node     string
	Resource permission.Resource
	Reason   string
	Entity   string
	// Alongside are the reads a tool makes beside its own, each checked by
	// its own resolver for its own resource.
	Alongside []*alongside
	// Instead are the reads a list makes in place of its own when a call
	// names their own variables.
	Instead []*alongside
}

// alongside is a read a tool makes beside its own: a connection read through
// its table input, or an unpaged list or one record keyed by the tool's own
// arguments.
type alongside struct {
	Operation *operation
	RootField *ast.FieldDefinition
	Resource  permission.Resource
	// Input is the variable a connection read takes its table input in.
	Input string
	// Keys are the variables a keyed read takes from the tool's own call
	// (a worker's recognitions by the workerId the call names).
	Keys []string
	// Object marks a keyed read whose root returns one record (an overview)
	// rather than a list.
	Object bool
	// Own are the variables a read made instead of the list's takes that the
	// list's operation does not declare: the call names them to choose it.
	Own []string
}

// unpaged reports whether a list tool's root field returns the whole list
// at once ([WorkerChecklist!]!), or its records under items with a limit and
// an offset (EdiCarrierInvoiceList), keyed by its own arguments rather than
// paged through a table input.
func (b *binding) unpaged() bool {
	return b.Verb == verbList && (b.RootField.Type.Elem != nil || b.items())
}

// items reports whether a list tool's root field returns its records under
// items rather than as a list or a connection.
func (b *binding) items() bool {
	return b.Verb == verbList && b.RootField.Type.Elem == nil &&
		itemsList(boundSchema, b.RootField)
}

// itemsList reports whether a root field returns an object that holds its
// records as a list under items.
func itemsList(schema *ast.Schema, field *ast.FieldDefinition) bool {
	if schema == nil {
		return false
	}
	def := schema.Types[field.Type.Name()]
	if def == nil || def.Kind != ast.Object {
		return false
	}
	items := def.Fields.ForName(itemsField)

	return items != nil && items.Type.Elem != nil
}

// tabled reports whether a paged list reads through a table input, whose
// record fields the server filters by its entity's configuration. A list
// paged by its root's own first and after filters by its arguments alone.
func (b *binding) tabled() bool {
	if b.Verb != verbList || b.unpaged() {
		return false
	}
	for _, variable := range b.Operation.Query.Operations[0].VariableDefinitions {
		if typ := variable.Type.Name(); typ == listInput || tableShaped(boundSchema, typ) {
			return true
		}
	}

	return false
}

// rootRead is what one Query root field reads, by the rule.
type rootRead struct {
	field    *ast.FieldDefinition
	verb     string
	node     string
	resource permission.Resource
	tool     string
}

// rootReads are the Query root fields that read one kind of record, each with
// the tool the rule names for it: a connection of X is list_<Xs>, an X by id
// is get_<x>, in reading order, then namespaced (agentports.ToolNameFor).
func rootReads(schema *ast.Schema) []rootRead {
	if schema.Query == nil {
		return nil
	}
	out := make([]rootRead, 0, len(schema.Query.Fields))
	for _, field := range schema.Query.Fields {
		verb, node, ok := readShape(schema, field)
		if !ok || (verb == verbGet && field.Arguments.ForName(idArgument) == nil) {
			continue
		}
		if verb == verbList && field.Type.Elem != nil {
			continue
		}
		resource := permission.Resource(stringutils.ConvertCamelToSnake(node))
		kind, known := resource.Kind()
		if !known {
			continue
		}
		noun := string(kind)
		if verb == verbList {
			noun = stringutils.PluralNoun(noun)
		}
		out = append(out, rootRead{
			field:    field,
			verb:     verb,
			node:     node,
			resource: resource,
			tool:     agentports.ToolNameFor(verb + readingSplit + noun),
		})
	}

	return out
}

func readShape(
	schema *ast.Schema,
	field *ast.FieldDefinition,
) (verb, node string, ok bool) {
	def := schema.Types[field.Type.Name()]
	if def == nil || def.Kind != ast.Object {
		return "", "", false
	}
	if field.Type.Elem != nil {
		return verbList, def.Name, true
	}
	if items := def.Fields.ForName(itemsField); items != nil && items.Type.Elem != nil {
		if node := schema.Types[items.Type.Name()]; node != nil && node.Kind == ast.Object {
			return verbList, node.Name, true
		}
	}
	if strings.HasSuffix(def.Name, connection) {
		edges := def.Fields.ForName(edgesField)
		if edges == nil || edges.Type.Elem == nil {
			return "", "", false
		}
		edge := schema.Types[edges.Type.Name()]
		if edge == nil {
			return "", "", false
		}
		node := edge.Fields.ForName(nodeField)
		if node == nil {
			return "", "", false
		}
		return verbList, node.Type.Name(), true
	}
	if len(field.Arguments) > 0 {
		return verbGet, def.Name, true
	}

	return "", "", false
}

// boundSchema and boundChecks are what a named binding is proven against;
// bind sets them once for the run.
var (
	boundSchema *ast.Schema
	boundChecks map[string]string
)

func bind(
	schema *ast.Schema,
	catalog map[string]*operation,
	reads []read,
	checks map[string]string,
) ([]*binding, error) {
	boundSchema, boundChecks = schema, checks
	byTool := make(map[string][]rootRead)
	for _, rr := range rootReads(schema) {
		byTool[rr.tool] = append(byTool[rr.tool], rr)
	}

	var problems []string
	bindings := make([]*binding, 0, len(reads))
	for _, r := range reads {
		b, problem := bindOne(catalog, byTool[r.Tool], r)
		if problem != "" {
			problems = append(problems, problem)
			continue
		}
		bindings = append(bindings, b)
	}
	if len(problems) > 0 {
		return nil, errors.New(strings.Join(problems, "\n"))
	}

	return bindings, nil
}

func bindOne(
	catalog map[string]*operation,
	candidates []rootRead,
	r read,
) (b *binding, problem string) {
	b, problem = chooseBinding(catalog, candidates, r)
	if problem != "" {
		return nil, problem
	}
	if problem = unsupplied(b); problem != "" {
		return nil, problem
	}
	for _, name := range r.Alongside {
		beside, besideProblem := bindAlongside(catalog, b, name)
		if besideProblem != "" {
			return nil, besideProblem
		}
		b.Alongside = append(b.Alongside, beside)
	}
	if len(r.Instead) > 0 && b.Verb != verbList {
		return nil, fmt.Sprintf("%s is a %s tool; only a list reads another operation "+
			"instead of its own", r.Tool, b.Verb)
	}
	for _, name := range r.Instead {
		instead, insteadProblem := bindInstead(catalog, b, name)
		if insteadProblem != "" {
			return nil, insteadProblem
		}
		b.Instead = append(b.Instead, instead)
	}

	return b, ""
}

// provenRead is an operation another read is made through: persisted, one
// root field, whose resolver checks a resource for read, which the tool asks
// of the reader before reading it.
func provenRead(
	catalog map[string]*operation,
	tool, name, how string,
) (beside *alongside, problem string) {
	chosen, ok := catalog[name]
	if !ok {
		return nil, fmt.Sprintf("%s reads %s %s, which is not a persisted query",
			tool, name, how)
	}
	if len(chosen.Roots) != 1 {
		return nil, fmt.Sprintf("%s reads %s %s, which reads %d root fields; one is read",
			tool, name, how, len(chosen.Roots))
	}
	field := boundSchema.Query.Fields.ForName(chosen.Roots[0])
	if field == nil {
		return nil, fmt.Sprintf("%s: %s is not a Query field", tool, chosen.Roots[0])
	}
	resource, checked := checkedResource(boundChecks, field.Name)
	if !checked {
		return nil, fmt.Sprintf("%s: the resolver of %s checks no read permission, so what "+
			"%s reads %s could not be held to it", tool, field.Name, tool, how)
	}

	return &alongside{
		Operation: chosen,
		RootField: field,
		Resource:  permission.Resource(resource),
	}, ""
}

// keyedShape reports whether a read's root field returns an unpaged list or
// one record, either of which a call's own arguments key.
func keyedShape(field *ast.FieldDefinition) (object, ok bool) {
	verb, _, shaped := readShape(boundSchema, field)
	switch {
	case shaped && verb == verbList && field.Type.Elem != nil:
		return false, true
	case shaped && verb == verbGet:
		return true, true
	}

	return false, false
}

// bindInstead proves an operation a list reads in place of its own: an
// unpaged list or one record, keyed by variables of its own that the call
// names to choose it, and by any of the list's own it shares, each of the
// same type. None of its own variables may be an input object, since a
// parameter cannot build one.
func bindInstead(
	catalog map[string]*operation,
	list *binding,
	name string,
) (instead *alongside, problem string) {
	instead, problem = provenRead(catalog, list.Tool, name, "instead")
	if problem != "" {
		return nil, problem
	}
	object, ok := keyedShape(instead.RootField)
	if !ok {
		return nil, fmt.Sprintf("%s reads %s instead, which is neither an unpaged list nor "+
			"one record", list.Tool, name)
	}
	instead.Object = object
	own := list.Operation.Query.Operations[0].VariableDefinitions
	for _, variable := range instead.Operation.Query.Operations[0].VariableDefinitions {
		if inputObject(variable.Type.Name()) {
			return nil, fmt.Sprintf("%s reads %s instead, which takes $%s: %s; a parameter "+
				"cannot build it", list.Tool, name, variable.Variable, variable.Type.String())
		}
		mine := own.ForName(variable.Variable)
		switch {
		case mine == nil:
			instead.Own = append(instead.Own, variable.Variable)
		case mine.Type.Name() != variable.Type.Name():
			return nil, fmt.Sprintf("%s reads %s instead, whose $%s is %s where the list's "+
				"is %s", list.Tool, name, variable.Variable, variable.Type.String(),
				mine.Type.String())
		default:
			instead.Keys = append(instead.Keys, variable.Variable)
		}
	}
	if len(instead.Own) == 0 {
		return nil, fmt.Sprintf("%s reads %s instead, which takes no variable of its own for "+
			"a call to choose it by", list.Tool, name)
	}

	return instead, ""
}

// bindAlongside proves an operation a tool reads beside its own: it is
// persisted, reads one root field whose resolver checks a resource for read,
// which the tool asks of the reader before reading it, and is either an
// unpaged list or one record keyed by the tool's own arguments, or, beside
// an unpaged list, a connection read that selects its pageInfo and needs no
// variable but its table input.
func bindAlongside(
	catalog map[string]*operation,
	list *binding,
	name string,
) (beside *alongside, problem string) {
	tool := list.Tool
	beside, problem = provenRead(catalog, tool, name, "alongside")
	if problem != "" {
		return nil, problem
	}
	chosen, field := beside.Operation, beside.RootField
	if object, keyed := keyedShape(field); keyed {
		beside.Object = object
		if beside.Keys, problem = alongsideKeys(list, chosen, name); problem != "" {
			return nil, problem
		}

		return beside, ""
	}
	verb, _, shaped := readShape(boundSchema, field)
	if !shaped || verb != verbList || !pages(chosen, rootRead{field: field, verb: verbList}) {
		return nil, fmt.Sprintf("%s reads %s alongside, which is neither a connection read "+
			"that selects its pageInfo nor a read keyed by the call", tool, name)
	}
	if !list.unpaged() {
		return nil, fmt.Sprintf("%s reads %s alongside, a connection; only an unpaged list "+
			"reads one, as a paged list's next call pages its own records", tool, name)
	}
	if problem = unsupplied(&binding{
		Tool: tool, Verb: verbList, Operation: chosen, RootField: field,
	}); problem != "" {
		return nil, problem
	}
	for _, variable := range chosen.Query.Operations[0].VariableDefinitions {
		if typ := variable.Type.Name(); typ == listInput || tableShaped(boundSchema, typ) {
			beside.Input = variable.Variable
		}
	}
	if beside.Input == "" {
		return nil, fmt.Sprintf("%s reads %s alongside, which takes no table input", tool, name)
	}

	return beside, ""
}

// alongsideKeys are the variables a keyed read alongside a tool takes from
// the tool's own call: each one the tool's operation declares, of the same
// type, so the read is about the record the call names. It is made only when
// the call names every one it requires.
func alongsideKeys(list *binding, chosen *operation, name string) (keys []string, problem string) {
	own := list.Operation.Query.Operations[0].VariableDefinitions
	for _, variable := range chosen.Query.Operations[0].VariableDefinitions {
		mine := own.ForName(variable.Variable)
		if mine == nil || mine.Type.Name() != variable.Type.Name() ||
			inputObject(variable.Type.Name()) {
			return nil, fmt.Sprintf("%s reads %s alongside, which takes $%s: %s; the list's "+
				"own call does not supply it", list.Tool, name, variable.Variable,
				variable.Type.String())
		}
		keys = append(keys, variable.Variable)
	}
	if len(keys) == 0 {
		return nil, fmt.Sprintf("%s reads %s alongside, a read keyed by nothing the call "+
			"names", list.Tool, name)
	}

	return keys, ""
}

// tableFields are what makes an input a table's: a list tool builds any input
// that holds them, and sends the input's other fields as typed parameters.
var tableFields = []string{"first", "after", "query", "fieldFilters", "sort"}

// inputObject reports whether a type is an input object, which a tool's
// parameter cannot be.
func inputObject(name string) bool {
	if boundSchema == nil {
		return false
	}
	def := boundSchema.Types[name]

	return def != nil && def.Kind == ast.InputObject
}

// tableShaped reports whether an input is a table's input under another name
// (WorkerPTOEntriesInput: the table's fields and the screen's own filters).
func tableShaped(schema *ast.Schema, name string) bool {
	if schema == nil {
		return false
	}
	def := schema.Types[name]
	if def == nil || def.Kind != ast.InputObject {
		return false
	}
	for _, field := range tableFields {
		if def.Fields.ForName(field) == nil {
			return false
		}
	}

	return true
}

// unsupplied names the required variables of a binding's operation that its
// tool's parameters cannot supply. Every tool takes its operation's scalar,
// enum and id variables as parameters of its own (workerId, a required
// integrationType), and a list builds any input object: its table input
// from the table's fields, and a filter input from the filter's own. A get
// cannot build an input object.
func unsupplied(b *binding) string {
	want := supplied[b.Verb]
	var missing []string
	for _, variable := range b.Operation.Query.Operations[0].VariableDefinitions {
		if !variable.Type.NonNull || variable.DefaultValue != nil {
			continue
		}
		if b.RootField != nil && !inputObject(variable.Type.Name()) {
			continue
		}
		if typ, ok := want[variable.Variable]; ok && typ == variable.Type.Name() {
			continue
		}
		if typ, ok := want[""]; ok && (typ == variable.Type.Name() ||
			(b.Verb == verbList && tableShaped(boundSchema, variable.Type.Name()))) {
			continue
		}
		if b.RootField != nil && b.Verb == verbList && !b.unpaged() &&
			buildable(variable.Type.Name()) {
			continue
		}
		missing = append(missing, "$"+variable.Variable+": "+variable.Type.String())
	}
	if len(missing) == 0 {
		return ""
	}

	return fmt.Sprintf("%s reads %s, which requires %s; a %s tool's parameters cannot supply it",
		b.Tool, b.Operation.Name, strings.Join(missing, ", "), b.Verb)
}

// buildable reports whether a paged list can build a filter input from its
// parameters: every field it requires is a scalar, an enum or an id.
func buildable(name string) bool {
	def := boundSchema.Types[name]
	if def == nil || def.Kind != ast.InputObject {
		return false
	}
	for _, field := range def.Fields {
		if field.Type.NonNull && field.DefaultValue == nil && inputObject(field.Type.Name()) {
			return false
		}
	}

	return true
}

func chooseBinding(
	catalog map[string]*operation,
	candidates []rootRead,
	r read,
) (b *binding, problem string) {
	ruled, ruledRoot, problem := ruleOperation(catalog, candidates, r.Tool)
	switch {
	case r.Operation == "" && problem != "":
		return nil, problem
	case r.Operation == "":
		return newBinding(r, ruled, ruledRoot, ""), ""
	case ruled != nil && ruled.Name == r.Operation:
		return nil, fmt.Sprintf("%s names %s, which the rule picks anyway; remove the override",
			r.Tool, r.Operation)
	case r.Reason == "":
		return nil, fmt.Sprintf("%s names %s rather than the rule's choice (%s); say why in reason",
			r.Tool, r.Operation, describe(ruled, problem))
	}

	chosen, ok := catalog[r.Operation]
	if !ok {
		return nil, fmt.Sprintf("%s names %s, which is not a persisted query", r.Tool, r.Operation)
	}
	if len(chosen.Roots) != 1 {
		return nil, fmt.Sprintf("%s names %s, which reads %d root fields; a tool reads one",
			r.Tool, r.Operation, len(chosen.Roots))
	}

	return namedBinding(r, chosen)
}

// namedBinding binds a tool to the operation reads.yml names. The root field
// it reads must have the shape the tool's verb says, and the resource its
// resolver checks becomes the binding's, which a test holds to the tool's
// own policy.
func namedBinding(r read, chosen *operation) (b *binding, problem string) {
	field := boundSchema.Query.Fields.ForName(chosen.Roots[0])
	if field == nil {
		return nil, fmt.Sprintf("%s: %s is not a Query field", r.Tool, chosen.Roots[0])
	}
	verb, node, ok := readShape(boundSchema, field)
	want, _, _ := strings.Cut(agentports.ReadingOrder(r.Tool), readingSplit)
	if !ok || verb != want {
		return nil, fmt.Sprintf("%s is a %s tool, but %s reads %s, which is not one",
			r.Tool, want, r.Operation, chosen.Roots[0])
	}
	resource, checked := checkedResource(boundChecks, field.Name)
	if !checked {
		return nil, fmt.Sprintf("%s: the resolver of %s checks no read permission, so the "+
			"tool's own could not be held to it", r.Tool, field.Name)
	}

	return &binding{
		Tool:      r.Tool,
		Verb:      verb,
		Operation: chosen,
		RootField: field,
		Node:      node,
		Resource:  permission.Resource(resource),
		Reason:    r.Reason,
		Entity:    r.Entity,
	}, ""
}

func describe(op *operation, problem string) string {
	if op != nil {
		return op.Name
	}

	return problem
}

func newBinding(r read, op *operation, rr rootRead, reason string) *binding {
	resource := rr.resource
	if checked, ok := checkedResource(boundChecks, rr.field.Name); ok {
		resource = permission.Resource(checked)
	}

	return &binding{
		Tool:      r.Tool,
		Verb:      rr.verb,
		Operation: op,
		RootField: rr.field,
		Node:      rr.node,
		Resource:  resource,
		Reason:    reason,
		Entity:    r.Entity,
	}
}

// ruleOperation picks a tool's operation: among the persisted queries that
// read only the tool's root field, the Desk's own (written for an agent to
// read), then the table's, then the only one.
func ruleOperation(
	catalog map[string]*operation,
	candidates []rootRead,
	tool string,
) (op *operation, rr rootRead, problem string) {
	if len(candidates) == 0 {
		return nil, rootRead{}, tool + ": no Query root field reads the record it is named for"
	}
	if len(candidates) > 1 {
		names := make([]string, 0, len(candidates))
		for _, c := range candidates {
			names = append(names, c.field.Name)
		}
		return nil, rootRead{}, fmt.Sprintf("%s: several root fields read its record (%s); name "+
			"the operation", tool, strings.Join(names, ", "))
	}

	rr = candidates[0]
	var ops []*operation
	for _, op := range catalog {
		if len(op.Roots) == 1 && op.Roots[0] == rr.field.Name && pages(op, rr) {
			ops = append(ops, op)
		}
	}
	sort.Slice(ops, func(i, j int) bool { return ops[i].Name < ops[j].Name })

	pick := func(match func(*operation) bool) []*operation {
		var out []*operation
		for _, op := range ops {
			if match(op) {
				out = append(out, op)
			}
		}
		return out
	}
	for _, tier := range [][]*operation{
		pick(func(op *operation) bool { return strings.HasPrefix(op.Name, deskPrefix) }),
		pick(func(op *operation) bool { return op.Name == rr.node+tableSuffix }),
		ops,
	} {
		switch len(tier) {
		case 0:
			continue
		case 1:
			return tier[0], rr, ""
		default:
			names := make([]string, 0, len(tier))
			for _, op := range tier {
				names = append(names, op.Name)
			}
			return nil, rr, fmt.Sprintf("%s: %s is read by several operations (%s); name one",
				tool, rr.field.Name, strings.Join(names, ", "))
		}
	}

	return nil, rr, fmt.Sprintf("%s: no persisted query reads %s alone; add one", tool,
		rr.field.Name)
}

// pages reports whether an operation can serve a list tool's paging: a
// connection read must select its pageInfo, or a tool reading through it
// could never offer the next page.
func pages(op *operation, rr rootRead) bool {
	if rr.verb != verbList || rr.field.Type.Elem != nil || itemsList(boundSchema, rr.field) {
		return true
	}
	for _, selection := range op.Query.Operations[0].SelectionSet {
		root, ok := selection.(*ast.Field)
		if !ok || root.Name != rr.field.Name {
			continue
		}
		fragments := make(map[string]*ast.FragmentDefinition, len(op.Query.Fragments))
		for _, fragment := range op.Query.Fragments {
			fragments[fragment.Name] = fragment
		}
		for _, f := range collect(root.SelectionSet, fragments) {
			if f.Name == pageInfo {
				return true
			}
		}
	}

	return false
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}
