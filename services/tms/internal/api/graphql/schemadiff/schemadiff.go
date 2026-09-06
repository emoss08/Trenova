package schemadiff

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type Severity int

const (
	SeverityNonBreaking Severity = iota
	SeverityDangerous
	SeverityBreaking
)

func (s Severity) String() string {
	switch s {
	case SeverityBreaking:
		return "BREAKING"
	case SeverityDangerous:
		return "DANGEROUS"
	default:
		return "ok"
	}
}

type Change struct {
	Severity Severity
	Path     string
	Message  string
}

func (c Change) String() string {
	return fmt.Sprintf("%-9s %s: %s", c.Severity, c.Path, c.Message)
}

type Report struct {
	Changes []Change
}

func (r Report) Breaking() []Change {
	return r.filter(SeverityBreaking)
}

func (r Report) Dangerous() []Change {
	return r.filter(SeverityDangerous)
}

func (r Report) HasBreaking() bool {
	return len(r.Breaking()) > 0
}

func (r Report) filter(severity Severity) []Change {
	var out []Change
	for _, c := range r.Changes {
		if c.Severity == severity {
			out = append(out, c)
		}
	}

	return out
}

func LoadDir(dir string) (*ast.Schema, error) {
	paths, err := filepath.Glob(filepath.Join(dir, "*.graphqls"))
	if err != nil {
		return nil, fmt.Errorf("glob schema dir %s: %w", dir, err)
	}
	if len(paths) == 0 {
		return nil, fmt.Errorf("no .graphqls files in %s", dir)
	}
	sort.Strings(paths)

	sources := make([]*ast.Source, 0, len(paths))
	for _, path := range paths {
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil, fmt.Errorf("read %s: %w", path, readErr)
		}
		sources = append(sources, &ast.Source{Name: filepath.Base(path), Input: string(content)})
	}

	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return nil, fmt.Errorf("load schema from %s: %w", dir, err)
	}

	return schema, nil
}

func Compare(base, head *ast.Schema) Report {
	d := &differ{}

	for _, name := range sortedTypeNames(base) {
		if isBuiltin(name) {
			continue
		}
		baseType := base.Types[name]
		headType, ok := head.Types[name]
		if !ok {
			d.add(SeverityBreaking, name, "type removed")
			continue
		}
		if baseType.Kind != headType.Kind {
			d.add(SeverityBreaking, name,
				fmt.Sprintf("kind changed from %s to %s", baseType.Kind, headType.Kind))
			continue
		}
		d.compareType(baseType, headType)
	}

	for _, name := range sortedTypeNames(head) {
		if _, existed := base.Types[name]; !existed && !isBuiltin(name) {
			d.add(SeverityNonBreaking, name, "type added")
		}
	}

	sort.SliceStable(d.changes, func(i, j int) bool {
		if d.changes[i].Severity != d.changes[j].Severity {
			return d.changes[i].Severity > d.changes[j].Severity
		}
		return d.changes[i].Path < d.changes[j].Path
	})

	return Report{Changes: d.changes}
}

type differ struct {
	changes []Change
}

func (d *differ) add(severity Severity, path, message string) {
	d.changes = append(d.changes, Change{Severity: severity, Path: path, Message: message})
}

func (d *differ) compareType(base, head *ast.Definition) {
	switch base.Kind {
	case ast.Object, ast.Interface:
		d.compareOutputFields(base, head)
		d.compareInterfaces(base, head)
	case ast.InputObject:
		d.compareInputFields(base, head)
	case ast.Enum:
		d.compareEnumValues(base, head)
	case ast.Union:
		d.compareUnionMembers(base, head)
	case ast.Scalar:
	}
}

func (d *differ) compareOutputFields(base, head *ast.Definition) {
	for _, baseField := range base.Fields {
		path := base.Name + "." + baseField.Name
		headField := head.Fields.ForName(baseField.Name)
		if headField == nil {
			d.add(SeverityBreaking, path, "field removed")
			continue
		}
		d.compareOutputType(path, baseField.Type, headField.Type)
		d.compareArguments(path, baseField.Arguments, headField.Arguments)
		d.compareDeprecation(path, baseField.Directives, headField.Directives)
	}
	for _, headField := range head.Fields {
		if base.Fields.ForName(headField.Name) == nil {
			d.add(SeverityNonBreaking, base.Name+"."+headField.Name, "field added")
		}
	}
}

func (d *differ) compareInputFields(base, head *ast.Definition) {
	for _, baseField := range base.Fields {
		path := base.Name + "." + baseField.Name
		headField := head.Fields.ForName(baseField.Name)
		if headField == nil {
			d.add(SeverityBreaking, path, "input field removed")
			continue
		}
		d.compareInputType(path, baseField.Type, headField.Type)
	}
	for _, headField := range head.Fields {
		if base.Fields.ForName(headField.Name) != nil {
			continue
		}
		path := base.Name + "." + headField.Name
		if headField.Type.NonNull && headField.DefaultValue == nil {
			d.add(SeverityBreaking, path,
				"required input field added; existing clients do not send it")
			continue
		}
		d.add(SeverityNonBreaking, path, "optional input field added")
	}
}

func (d *differ) compareArguments(path string, base, head ast.ArgumentDefinitionList) {
	for _, baseArg := range base {
		argPath := path + "(" + baseArg.Name + ")"
		headArg := head.ForName(baseArg.Name)
		if headArg == nil {
			d.add(SeverityBreaking, argPath, "argument removed")
			continue
		}
		d.compareInputType(argPath, baseArg.Type, headArg.Type)
	}
	for _, headArg := range head {
		if base.ForName(headArg.Name) != nil {
			continue
		}
		argPath := path + "(" + headArg.Name + ")"
		if headArg.Type.NonNull && headArg.DefaultValue == nil {
			d.add(SeverityBreaking, argPath,
				"required argument added; existing clients do not send it")
			continue
		}
		d.add(SeverityNonBreaking, argPath, "optional argument added")
	}
}

func (d *differ) compareOutputType(path string, base, head *ast.Type) {
	if base.String() == head.String() {
		return
	}
	if sameShape(base, head) {
		if base.NonNull && !head.NonNull {
			d.add(SeverityBreaking, path,
				fmt.Sprintf("type changed from %s to %s; clients may assume non-null",
					base, head))
			return
		}
		d.add(SeverityNonBreaking, path,
			fmt.Sprintf("type tightened from %s to %s", base, head))
		return
	}

	if wireCompatible(base, head) {
		d.add(SeverityDangerous, path, wireCompatibleMessage)
		return
	}

	d.add(SeverityBreaking, path, fmt.Sprintf("type changed from %s to %s", base, head))
}

func (d *differ) compareInputType(path string, base, head *ast.Type) {
	if base.String() == head.String() {
		return
	}
	if sameShape(base, head) {
		if !base.NonNull && head.NonNull {
			d.add(SeverityBreaking, path,
				fmt.Sprintf("type changed from %s to %s; clients may omit it", base, head))
			return
		}
		d.add(SeverityNonBreaking, path,
			fmt.Sprintf("type relaxed from %s to %s", base, head))
		return
	}

	if wireCompatible(base, head) {
		d.add(SeverityDangerous, path, wireCompatibleMessage)
		return
	}

	d.add(SeverityBreaking, path, fmt.Sprintf("type changed from %s to %s", base, head))
}

func (d *differ) compareEnumValues(base, head *ast.Definition) {
	for _, baseValue := range base.EnumValues {
		path := base.Name + "." + baseValue.Name
		if head.EnumValues.ForName(baseValue.Name) == nil {
			d.add(SeverityBreaking, path, "enum value removed")
		}
	}
	for _, headValue := range head.EnumValues {
		if base.EnumValues.ForName(headValue.Name) == nil {
			d.add(SeverityDangerous, base.Name+"."+headValue.Name,
				"enum value added; clients with exhaustive switches must handle it")
		}
	}
}

func (d *differ) compareUnionMembers(base, head *ast.Definition) {
	headMembers := make(map[string]struct{}, len(head.Types))
	for _, member := range head.Types {
		headMembers[member] = struct{}{}
	}
	baseMembers := make(map[string]struct{}, len(base.Types))
	for _, member := range base.Types {
		baseMembers[member] = struct{}{}
		if _, ok := headMembers[member]; !ok {
			d.add(SeverityBreaking, base.Name+"."+member, "union member removed")
		}
	}
	for _, member := range head.Types {
		if _, ok := baseMembers[member]; !ok {
			d.add(SeverityDangerous, base.Name+"."+member,
				"union member added; clients with exhaustive fragments must handle it")
		}
	}
}

func (d *differ) compareInterfaces(base, head *ast.Definition) {
	headInterfaces := make(map[string]struct{}, len(head.Interfaces))
	for _, name := range head.Interfaces {
		headInterfaces[name] = struct{}{}
	}
	for _, name := range base.Interfaces {
		if _, ok := headInterfaces[name]; !ok {
			d.add(SeverityBreaking, base.Name,
				fmt.Sprintf("no longer implements %s", name))
		}
	}
}

func (d *differ) compareDeprecation(path string, base, head ast.DirectiveList) {
	if base.ForName("deprecated") == nil && head.ForName("deprecated") != nil {
		d.add(SeverityNonBreaking, path, "deprecated")
	}
}

const wireCompatibleMessage = "scalar changed to a wire-compatible type; regenerate clients"

var wireCompatibleScalars = map[string]string{
	"Timestamp": "Int",
	"Decimal":   "String",
}

func wireCompatibleScalar(a, b string) bool {
	return wireCompatibleScalars[a] == b || wireCompatibleScalars[b] == a
}

func wireCompatible(a, b *ast.Type) bool {
	for a != nil && b != nil {
		if a.NonNull != b.NonNull || (a.Elem == nil) != (b.Elem == nil) {
			return false
		}
		if a.Elem == nil {
			return wireCompatibleScalar(a.NamedType, b.NamedType)
		}
		a, b = a.Elem, b.Elem
	}

	return false
}

func sameShape(a, b *ast.Type) bool {
	for a != nil && b != nil {
		if (a.Elem == nil) != (b.Elem == nil) {
			return false
		}
		if a.Elem == nil {
			return a.NamedType == b.NamedType
		}
		a, b = a.Elem, b.Elem
	}

	return a == nil && b == nil
}

func sortedTypeNames(schema *ast.Schema) []string {
	names := make([]string, 0, len(schema.Types))
	for name := range schema.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	return names
}

func isBuiltin(name string) bool {
	if strings.HasPrefix(name, "__") {
		return true
	}
	switch name {
	case "Int", "Float", "String", "Boolean", "ID":
		return true
	default:
		return false
	}
}
