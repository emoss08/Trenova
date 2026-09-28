package toolcontract

import (
	"fmt"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/api/graphql/schemadiff"
	"github.com/emoss08/trenova/internal/api/writecoverage"
	"github.com/emoss08/trenova/internal/core/domain/permission"
	serviceports "github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/agenttoolpolicy/registered"
	"github.com/emoss08/trenova/pkg/toolschema"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2/ast"
)

const (
	schemaDir   = "../graphql/schema"
	mappingFile = "../writecoverage/" + writecoverage.MappingFile
)

type contract struct {
	schema  *ast.Schema
	binding *Binding
	tool    serviceports.AgentTool
	used    map[string]bool
}

func loadSchema(t *testing.T) *ast.Schema {
	t.Helper()

	schema, err := schemadiff.LoadDir(schemaDir)
	require.NoError(t, err)

	return schema
}

func loadActionTools(t *testing.T) map[string]serviceports.AgentTool {
	t.Helper()

	built, err := registered.Build()
	require.NoError(t, err)

	tools := make(map[string]serviceports.AgentTool, len(built.Actions))
	for _, tool := range built.Actions {
		tools[tool.Name()] = tool
	}

	return tools
}

func TestEveryBoundToolFitsItsGraphQLInput(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	tools := loadActionTools(t)

	for idx := range Bindings {
		binding := &Bindings[idx]
		t.Run(binding.Tool, func(t *testing.T) {
			t.Parallel()

			tool, ok := tools[binding.Tool]
			require.Truef(t, ok, "binding names %s, which is not a registered action tool",
				binding.Tool)

			c := &contract{schema: schema, binding: binding, tool: tool, used: map[string]bool{}}
			problems := c.compare(c.root(t), binding.Input, "")
			problems = append(problems, c.stale()...)

			require.Emptyf(t, problems,
				"%s no longer fits %s. Change the tool's schema, or record why in "+
					"internal/api/toolcontract/bindings.go:\n- %s",
				binding.Tool, binding.Input, strings.Join(problems, "\n- "))
		})
	}
}

func TestEveryWriteToolWithAGraphQLInputIsBound(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	tools := loadActionTools(t)
	twins := graphQLInputTwins(t, schema)

	bound := make(map[string]bool, len(Bindings))
	for idx := range Bindings {
		bound[Bindings[idx].Tool] = true
	}

	var problems []string
	for name, tool := range tools {
		operation := tool.Policy().Operation
		if operation != permission.OpCreate && operation != permission.OpUpdate {
			continue
		}
		inputs, twinned := twins[name]
		reason, listed := Unbound[name]
		switch {
		case bound[name] && listed:
			problems = append(problems, name+" is bound and also listed as unbound")
		case bound[name]:
		case listed && strings.TrimSpace(reason) == "":
			problems = append(problems, name+" is listed as unbound without a reason")
		case listed && !twinned:
			problems = append(problems, name+" is listed as unbound, but none of its "+
				"writes takes a GraphQL input any more; remove it from Unbound")
		case twinned && !listed:
			problems = append(problems, fmt.Sprintf(
				"%s writes through %s but is not bound to it: add a Binding, or an "+
					"Unbound entry saying why", name, strings.Join(inputs, ", ")))
		}
	}
	for name := range Unbound {
		if _, ok := tools[name]; !ok {
			problems = append(problems, name+" is listed as unbound but is not a registered tool")
		}
	}
	sort.Strings(problems)

	require.Emptyf(t, problems, "internal/api/toolcontract/bindings.go is out of date:\n- %s",
		strings.Join(problems, "\n- "))
}

func graphQLInputTwins(t *testing.T, schema *ast.Schema) map[string][]string {
	t.Helper()

	mapping, err := writecoverage.LoadMapping(mappingFile)
	require.NoError(t, err)

	twins := make(map[string][]string)
	for key, decision := range mapping.Writes {
		name, isMutation := strings.CutPrefix(key, writecoverage.MutationKey(""))
		if !isMutation || schema.Mutation == nil {
			continue
		}
		field := schema.Mutation.Fields.ForName(name)
		if field == nil {
			continue
		}
		for _, arg := range field.Arguments {
			def := schema.Types[arg.Type.Name()]
			if def == nil || def.Kind != ast.InputObject {
				continue
			}
			for _, tool := range decision.Tools {
				if !slices.Contains(twins[tool], def.Name) {
					twins[tool] = append(twins[tool], def.Name)
				}
			}
		}
	}

	return twins
}

func (c *contract) root(t *testing.T) map[string]any {
	t.Helper()

	schema := c.tool.ParamSchema()
	if c.binding.Param == "" {
		return schema
	}

	properties, _ := schema[toolschema.KeyProperties].(map[string]any)
	param, ok := properties[c.binding.Param].(map[string]any)
	require.Truef(t, ok, "%s has no parameter %q", c.binding.Tool, c.binding.Param)

	return objectOf(param)
}

func objectOf(property map[string]any) map[string]any {
	if property[toolschema.KeyType] == toolschema.TypeArray {
		items, _ := property[toolschema.KeyItems].(map[string]any)
		return items
	}

	return property
}

func (c *contract) compare(object map[string]any, inputName, prefix string) []string {
	input := c.schema.Types[inputName]
	if input == nil || input.Kind != ast.InputObject {
		return []string{fmt.Sprintf("%s is not a GraphQL input type", inputName)}
	}

	properties, _ := object[toolschema.KeyProperties].(map[string]any)
	required := requiredOf(object)
	var problems []string

	for _, field := range input.Fields {
		path := prefix + field.Name
		if !field.Type.NonNull || field.DefaultValue != nil {
			continue
		}
		if _, defaulted := c.binding.Defaulted[path]; defaulted {
			c.used["Defaulted:"+path] = true
			continue
		}
		if !slices.Contains(required, field.Name) {
			problems = append(problems, fmt.Sprintf(
				"%s.%s is required by the input but not by the tool", inputName, path))
		}
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		path := prefix + name
		property, _ := properties[name].(map[string]any)
		if _, extra := c.binding.Extra[path]; extra {
			c.used["Extra:"+path] = true
			continue
		}
		field := input.Fields.ForName(name)
		if field == nil {
			problems = append(problems, fmt.Sprintf(
				"parameter %s is not a field of %s", path, inputName))
			continue
		}
		problems = append(problems, c.compareEnum(property, field, path)...)
		if nested, ok := c.binding.Nested[path]; ok {
			c.used["Nested:"+path] = true
			problems = append(problems, c.compare(objectOf(property), nested, path+".")...)
		}
	}

	return problems
}

func (c *contract) compareEnum(
	property map[string]any,
	field *ast.FieldDefinition,
	path string,
) []string {
	def := c.schema.Types[field.Type.Name()]
	if def == nil || def.Kind != ast.Enum {
		return nil
	}

	values := enumOf(objectOf(property))
	if len(values) == 0 {
		return []string{fmt.Sprintf(
			"parameter %s is free text, but %s takes the enum %s", path, c.binding.Input, def.Name)}
	}

	declared := make([]string, 0, len(def.EnumValues))
	for _, value := range def.EnumValues {
		declared = append(declared, value.Name)
	}
	sort.Strings(declared)
	sort.Strings(values)

	if _, narrowed := c.binding.Narrowed[path]; narrowed {
		c.used["Narrowed:"+path] = true
		for _, value := range values {
			if !slices.Contains(declared, value) {
				return []string{fmt.Sprintf(
					"parameter %s offers %s, which %s does not", path, value, def.Name)}
			}
		}

		return nil
	}

	if !slices.Equal(values, declared) {
		return []string{fmt.Sprintf("parameter %s offers [%s] but %s is [%s]",
			path, strings.Join(values, ", "), def.Name, strings.Join(declared, ", "))}
	}

	return nil
}

func (c *contract) stale() []string {
	var problems []string
	for kind, keys := range map[string]map[string]string{
		"Extra":     c.binding.Extra,
		"Defaulted": c.binding.Defaulted,
		"Narrowed":  c.binding.Narrowed,
		"Nested":    c.binding.Nested,
	} {
		for path, value := range keys {
			if strings.TrimSpace(value) == "" {
				problems = append(problems, fmt.Sprintf("%s %q says nothing", kind, path))
			}
			if !c.used[kind+":"+path] {
				problems = append(problems, fmt.Sprintf(
					"%s %q no longer matches anything; remove it", kind, path))
			}
		}
	}
	sort.Strings(problems)

	return problems
}

func requiredOf(object map[string]any) []string {
	switch required := object[toolschema.KeyRequired].(type) {
	case []string:
		return required
	case []any:
		names := make([]string, 0, len(required))
		for _, name := range required {
			if text, ok := name.(string); ok {
				names = append(names, text)
			}
		}
		return names
	default:
		return nil
	}
}

func enumOf(property map[string]any) []string {
	switch values := property[toolschema.KeyEnum].(type) {
	case []string:
		return slices.Clone(values)
	case []any:
		names := make([]string, 0, len(values))
		for _, value := range values {
			if text, ok := value.(string); ok {
				names = append(names, text)
			}
		}
		return names
	default:
		return nil
	}
}

func TestTheContractCatchesAParameterTheInputDoesNotTake(t *testing.T) {
	t.Parallel()

	schema := loadSchema(t)
	binding := &Binding{
		Tool:   "create_shipment",
		Input:  "ShipmentInput",
		Nested: map[string]string{"moves": "ShipmentMoveInput"},
	}
	c := &contract{schema: schema, binding: binding, used: map[string]bool{}}
	object := map[string]any{
		toolschema.KeyProperties: map[string]any{
			"customerId": map[string]any{toolschema.KeyType: toolschema.TypeString},
			"freightTerms": map[string]any{
				toolschema.KeyType: toolschema.TypeString,
				toolschema.KeyEnum: []string{"Prepaid", "Collect"},
			},
			"moves": map[string]any{
				toolschema.KeyType: toolschema.TypeArray,
				toolschema.KeyItems: map[string]any{
					toolschema.KeyProperties: map[string]any{
						"type": map[string]any{toolschema.KeyType: toolschema.TypeString},
					},
				},
			},
		},
		toolschema.KeyRequired: []string{"customerId"},
	}

	problems := c.compare(object, binding.Input, "")

	joined := strings.Join(problems, "\n")
	require.Contains(t, joined, "parameter moves.type is not a field of ShipmentMoveInput")
	require.Contains(t, joined, "ShipmentInput.formulaTemplateId is required by the input")
	require.Contains(t, joined, "parameter freightTerms offers [Collect, Prepaid] but "+
		"FreightTerms is [Collect, Prepaid, ThirdParty]")
}
