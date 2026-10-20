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

// supplied are the variables a tool of each verb sends, by name and type: a
// list builds the table input and turns the total count off, a get sends the
// record's id. An operation that requires any other variable cannot be read
// by the tool's typed parameters.
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

	return b, ""
}

// unsupplied names the required variables of a binding's operation that its
// tool's parameters cannot supply.
func unsupplied(b *binding) string {
	want := supplied[b.Verb]
	var missing []string
	for _, variable := range b.Operation.Query.Operations[0].VariableDefinitions {
		if !variable.Type.NonNull || variable.DefaultValue != nil {
			continue
		}
		if typ, ok := want[variable.Variable]; ok && typ == variable.Type.Name() {
			continue
		}
		if typ, ok := want[""]; ok && typ == variable.Type.Name() {
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
	if rr.verb != verbList || rr.field.Type.Elem != nil {
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
