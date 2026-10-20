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
	readingSplit = "_"
)

// read is one entry of reads.yml: a tool that reads through GraphQL. The
// operation is worked out by the rule; naming one that differs needs a
// reason, which is the record of why the rule is wrong for it.
type read struct {
	Tool      string `yaml:"tool"`
	Operation string `yaml:"operation,omitempty"`
	Reason    string `yaml:"reason,omitempty"`
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
		if !ok {
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
	if field.Type.Elem != nil {
		return "", "", false
	}
	def := schema.Types[field.Type.Name()]
	if def == nil || def.Kind != ast.Object {
		return "", "", false
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
	if field.Arguments.ForName(idArgument) != nil {
		return verbGet, def.Name, true
	}

	return "", "", false
}

func bind(
	schema *ast.Schema,
	catalog map[string]*operation,
	reads []read,
) ([]*binding, error) {
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
	for _, candidate := range candidates {
		if candidate.field.Name == chosen.Roots[0] {
			return newBinding(r, chosen, candidate, r.Reason), ""
		}
	}

	return nil, fmt.Sprintf(
		"%s names %s, whose root field %s reads no record the tool is named for",
		r.Tool,
		r.Operation,
		chosen.Roots[0],
	)
}

func describe(op *operation, problem string) string {
	if op != nil {
		return op.Name
	}

	return problem
}

func newBinding(r read, op *operation, rr rootRead, reason string) *binding {
	return &binding{
		Tool:      r.Tool,
		Verb:      rr.verb,
		Operation: op,
		RootField: rr.field,
		Node:      rr.node,
		Resource:  rr.resource,
		Reason:    reason,
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
		if len(op.Roots) == 1 && op.Roots[0] == rr.field.Name {
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

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	slices.Sort(keys)

	return keys
}
