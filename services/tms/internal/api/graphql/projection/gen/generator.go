package main

import (
	"bytes"
	"fmt"
	"go/format"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"text/template"

	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
	"gopkg.in/yaml.v3"
)

func run(opts generatorOptions) error {
	manifestData, err := loadManifest(opts.ManifestPath)
	if err != nil {
		return err
	}

	schema, err := loadSchema(opts.SchemaDir)
	if err != nil {
		return err
	}

	output, err := generate(manifestData, schema, opts.FieldMaps)
	if err != nil {
		return err
	}

	return os.WriteFile(opts.OutputPath, output, 0o644)
}

func loadManifest(path string) (manifest, error) {
	f, err := os.Open(path)
	if err != nil {
		return manifest{}, fmt.Errorf("opening manifest %q: %w", path, err)
	}
	defer f.Close()

	var data manifest
	decoder := yaml.NewDecoder(f)
	decoder.KnownFields(true)
	if err = decoder.Decode(&data); err != nil {
		return manifest{}, fmt.Errorf("decoding manifest %q: %w", path, err)
	}

	return data, nil
}

func loadSchema(dir string) (*ast.Schema, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.graphqls"))
	if err != nil {
		return nil, fmt.Errorf("globbing schema files: %w", err)
	}
	if len(matches) == 0 {
		return nil, fmt.Errorf("no GraphQL schema files found in %q", dir)
	}

	sort.Strings(matches)
	sources := make([]*ast.Source, 0, len(matches))
	for _, match := range matches {
		input, readErr := os.ReadFile(match)
		if readErr != nil {
			return nil, fmt.Errorf("reading schema file %q: %w", match, readErr)
		}
		sources = append(sources, &ast.Source{
			Name:  match,
			Input: string(input),
		})
	}

	schema, err := gqlparser.LoadSchema(sources...)
	if err != nil {
		return nil, fmt.Errorf("loading GraphQL schema: %w", err)
	}

	return schema, nil
}

func generate(data manifest, schema *ast.Schema, fieldMaps map[string]map[string]string) ([]byte, error) {
	if len(data.Types) == 0 {
		return nil, fmt.Errorf("manifest must declare at least one type")
	}

	if err := validateObjectCoverage(data, schema); err != nil {
		return nil, err
	}

	names := make([]string, 0, len(data.Types))
	for name := range data.Types {
		names = append(names, name)
	}
	sort.Strings(names)

	specs := make([]generatedSpec, 0, len(names))
	for _, name := range names {
		spec, err := buildSpec(name, data, schema, fieldMaps)
		if err != nil {
			return nil, err
		}
		specs = append(specs, spec)
	}

	var buf bytes.Buffer
	tmpl := template.Must(template.New("specs").Funcs(template.FuncMap{
		"quote": quote,
	}).Parse(specsTemplate))
	if err := tmpl.Execute(&buf, specs); err != nil {
		return nil, fmt.Errorf("executing template: %w", err)
	}

	formatted, err := format.Source(buf.Bytes())
	if err != nil {
		return nil, fmt.Errorf("formatting generated specs: %w\n%s", err, buf.String())
	}

	return formatted, nil
}

func validateObjectCoverage(data manifest, schema *ast.Schema) error {
	seen := make(map[string]struct{}, len(data.Types)+len(data.Skip))
	for name := range data.Types {
		if err := validateManifestType(schema, name); err != nil {
			return err
		}
		seen[name] = struct{}{}
	}
	for _, name := range data.Skip {
		if err := validateManifestType(schema, name); err != nil {
			return fmt.Errorf("skip %q: %w", name, err)
		}
		seen[name] = struct{}{}
	}

	objectNames := make([]string, 0, len(schema.Types))
	for name, def := range schema.Types {
		if def.Kind != ast.Object || strings.HasPrefix(name, "__") {
			continue
		}
		objectNames = append(objectNames, name)
	}
	sort.Strings(objectNames)

	for _, name := range objectNames {
		if _, ok := seen[name]; !ok {
			return fmt.Errorf("schema object %q must be declared under types or skip", name)
		}
	}

	return nil
}

func validateManifestType(schema *ast.Schema, name string) error {
	def := schema.Types[name]
	if def == nil {
		return fmt.Errorf("unknown schema type %q", name)
	}
	if def.Kind != ast.Object {
		return fmt.Errorf("schema type %q is %s, not OBJECT", name, def.Kind)
	}

	return nil
}

func buildSpec(
	name string,
	data manifest,
	schema *ast.Schema,
	fieldMaps map[string]map[string]string,
) (generatedSpec, error) {
	typeData := data.Types[name]
	def := schema.Types[name]
	fieldMap, ok := fieldMaps[name]
	if !ok {
		return generatedSpec{}, fmt.Errorf("type %q has no buncolgen field map registered", name)
	}

	knownFields := make(map[string]*ast.FieldDefinition, len(def.Fields))
	for _, field := range def.Fields {
		knownFields[field.Name] = field
	}
	if err := validateManifestFields(name, typeData, knownFields); err != nil {
		return generatedSpec{}, err
	}

	always, err := buildAlwaysColumns(name, typeData, fieldMap)
	if err != nil {
		return generatedSpec{}, err
	}

	fields := make([]generatedField, 0, len(def.Fields))
	for _, field := range def.Fields {
		generated, err := buildField(fieldBuildParams{
			TypeName: name,
			Field:    field,
			TypeData: typeData,
			Manifest: data,
			FieldMap: fieldMap,
		})
		if err != nil {
			return generatedSpec{}, err
		}
		fields = append(fields, generated)
	}

	return generatedSpec{
		Name:          name,
		FieldMap:      "buncolgen." + name + "FieldMap",
		AlwaysColumns: always,
		Fields:        fields,
	}, nil
}

func validateManifestFields(
	typeName string,
	typeData typeManifest,
	knownFields map[string]*ast.FieldDefinition,
) error {
	for _, field := range typeData.Always {
		if _, ok := knownFields[field]; !ok {
			return fmt.Errorf("type %q always field %q does not exist in schema", typeName, field)
		}
	}
	for field := range typeData.Aliases {
		if _, ok := knownFields[field]; !ok {
			return fmt.Errorf("type %q alias field %q does not exist in schema", typeName, field)
		}
	}
	for field := range typeData.Virtuals {
		if _, ok := knownFields[field]; !ok {
			return fmt.Errorf("type %q virtual field %q does not exist in schema", typeName, field)
		}
	}
	for field := range typeData.Relations {
		if _, ok := knownFields[field]; !ok {
			return fmt.Errorf("type %q relation field %q does not exist in schema", typeName, field)
		}
	}

	return nil
}

func buildAlwaysColumns(
	typeName string,
	typeData typeManifest,
	fieldMap map[string]string,
) ([]string, error) {
	columns := make([]string, 0, len(typeData.Always))
	for _, field := range typeData.Always {
		key := fieldMapKey(field, typeData)
		column, ok := fieldMap[key]
		if !ok {
			return nil, fmt.Errorf(
				"type %q always field %q uses missing field-map key %q",
				typeName,
				field,
				key,
			)
		}
		columns = append(columns, column)
	}

	return columns, nil
}

type fieldBuildParams struct {
	TypeName string
	Field    *ast.FieldDefinition
	TypeData typeManifest
	Manifest manifest
	FieldMap map[string]string
}

func buildField(params fieldBuildParams) (generatedField, error) {
	field := params.Field
	generated := generatedField{
		Name: field.Name,
	}

	if special, ok := params.TypeData.Virtuals[field.Name]; ok {
		generated.Special = special
		return generated, nil
	}

	if relation, ok := params.TypeData.Relations[field.Name]; ok {
		if _, targetOK := params.Manifest.Types[relation.Target]; !targetOK {
			return generatedField{}, fmt.Errorf(
				"type %q relation %q references unknown target %q",
				params.TypeName,
				field.Name,
				relation.Target,
			)
		}

		if relation.Target != namedType(field.Type) {
			return generatedField{}, fmt.Errorf(
				"type %q relation %q target %q does not match schema field type %q",
				params.TypeName,
				field.Name,
				relation.Target,
				namedType(field.Type),
			)
		}
		generated.Relation = &generatedRelation{
			Target: relation.Target,
			Gate:   relation.Gate,
		}
		if relation.ColumnKey != "" {
			if _, ok := params.FieldMap[relation.ColumnKey]; !ok {
				return generatedField{}, fmt.Errorf(
					"type %q relation %q uses missing field-map key %q",
					params.TypeName,
					field.Name,
					relation.ColumnKey,
				)
			}
			generated.FieldMapKey = relation.ColumnKey
		}
		return generated, nil
	}

	if _, ok := params.Manifest.Types[namedType(field.Type)]; ok {
		return generatedField{}, fmt.Errorf(
			"type %q field %q is object type %q but has no relation metadata",
			params.TypeName,
			field.Name,
			namedType(field.Type),
		)
	}

	key := fieldMapKey(field.Name, params.TypeData)
	if _, ok := params.FieldMap[key]; !ok {
		return generatedField{}, fmt.Errorf(
			"type %q field %q uses missing field-map key %q",
			params.TypeName,
			field.Name,
			key,
		)
	}
	generated.FieldMapKey = key

	return generated, nil
}

func fieldMapKey(field string, typeData typeManifest) string {
	if key, ok := typeData.Aliases[field]; ok {
		return key
	}

	return field
}

func namedType(t *ast.Type) string {
	if t == nil {
		return ""
	}

	return t.Name()
}

func quote(value string) string {
	return fmt.Sprintf("%q", value)
}
