package agentquerytoolservice

import (
	"reflect"
	"slices"
	"testing"

	"github.com/emoss08/trenova/internal/modules/agent/ports/agentports"
	"github.com/emoss08/trenova/internal/modules/identity/domain/permission"
	"github.com/emoss08/trenova/pkg/filtercatalog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGraphQLReadsAreBoundToRegisteredResources(t *testing.T) {
	t.Parallel()

	registry := permission.NewRegistry()
	for tool, read := range graphQLReads {
		assert.Truef(t, registry.HasResource(read.Resource),
			"%s reads %s, which the permission registry does not know", tool, read.Resource)
		assert.NotEmptyf(t, agentports.NamespaceOf(tool), "%s is outside a tool namespace", tool)
	}
}

func TestGraphQLReadsDeriveTheirTypedParameters(t *testing.T) {
	t.Parallel()

	for tool, read := range graphQLReads {
		t.Run(tool, func(t *testing.T) {
			t.Parallel()

			params := read.typedParams()
			require.NotEmpty(t, params, "a list tool reads by at least one typed parameter")

			names := make(map[string]struct{}, len(params))
			for _, param := range params {
				_, dup := names[param.Name]
				assert.Falsef(t, dup, "parameter %s is derived twice", param.Name)
				names[param.Name] = struct{}{}
				assert.NotEmpty(t, param.Schema["description"], param.Name)
			}
		})
	}
}

// Every field a converted tool's hand-kept catalog entry lets a model filter
// by must still be a typed parameter, or the conversion loses a capability.
func TestGraphQLReadsKeepEveryFilterTheCatalogOffered(t *testing.T) {
	t.Parallel()

	catalog := make(map[string][]filtercatalog.Field)
	for _, spec := range listCatalogSpecs() {
		catalog[spec.name] = spec.fields
	}

	for tool, read := range graphQLReads {
		fields, listed := catalog[tool]
		if !listed {
			continue
		}
		t.Run(tool, func(t *testing.T) {
			t.Parallel()

			params := read.typedParams()
			sources := make([]string, 0, len(params))
			for _, param := range params {
				sources = append(sources, param.Field)
			}
			for _, field := range fields {
				assert.Truef(t, slices.Contains(sources, field.Name),
					"%s filtered by %s; no typed parameter reads it", tool, field.Name)
			}
		})
	}
}

func TestTypedParamsCompileIntoTheOperationsFilters(t *testing.T) {
	t.Parallel()

	read := graphQLReads["masterdata_list_hazardous_materials"]
	params := read.typedParams()
	clk := filtercatalog.NewClock("America/Chicago")

	filters, err := compileFilters(params, map[string]any{
		"status":          []any{"Active"},
		"placardRequired": true,
		"createdAtFrom":   "2026-10-01",
		"createdAtTo":     "2026-10-20",
		"name":            "  acid ",
	}, clk)
	require.NoError(t, err)

	from, _ := clk.Day("2026-10-01")
	to, _ := clk.Day("2026-10-20")
	assert.ElementsMatch(t, []map[string]any{
		{"field": "status", "operator": "in", "value": []string{"Active"}},
		{"field": "placardRequired", "operator": "eq", "value": true},
		{"field": "createdAt", "operator": "gte", "value": from},
		{"field": "createdAt", "operator": "lte", "value": clk.DayEnd(to)},
		{"field": "name", "operator": "contains", "value": "acid"},
	}, filters)
}

func TestTypedParamsRefuseADayThatIsNotADay(t *testing.T) {
	t.Parallel()

	read := graphQLReads["masterdata_list_hazardous_materials"]
	params := read.typedParams()

	_, err := compileFilters(params, map[string]any{"createdAtFrom": "last week"},
		filtercatalog.NewClock("UTC"))

	assert.ErrorContains(t, err, "createdAtFrom")
}

// A tool reads only through a root field whose resolver checks the tool's
// own resource for read, so the check its read meets is the one dispatch's
// authorize already made.
func TestGraphQLReadsCheckTheToolsOwnResource(t *testing.T) {
	t.Parallel()

	for _, provider := range ToolProviders() {
		fn := reflect.ValueOf(provider)
		args := make([]reflect.Value, fn.Type().NumIn())
		for idx := range args {
			args[idx] = reflect.Zero(fn.Type().In(idx))
		}
		tool, ok := fn.Call(args)[0].Interface().(agentports.AgentQueryTool)
		if !ok || tool == nil {
			continue
		}
		read, bound := graphQLReads[tool.Name()]
		if !bound {
			continue
		}
		assert.Equalf(t, read.Resource, tool.Policy().Resource.String(),
			"%s reads %s, whose resolver checks %s", tool.Name(), read.Operation, read.Resource)
	}
}
