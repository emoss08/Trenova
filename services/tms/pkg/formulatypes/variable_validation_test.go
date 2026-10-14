package formulatypes_test

import (
	"fmt"
	"testing"

	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/stretchr/testify/assert"
)

func variableErrors(definitions ...*formulatypes.VariableDefinition) map[string]string {
	multiErr := errortypes.NewMultiError()
	formulatypes.ValidateVariableDefinitions(definitions, multiErr)
	fields := make(map[string]string, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields[fieldErr.Field] = fmt.Sprint(fieldErr.Message, fieldErr.Args)
	}
	return fields
}

func TestValidateVariableDefinitions_AcceptsWellFormedInputs(t *testing.T) {
	t.Parallel()

	assert.Empty(t, variableErrors(
		&formulatypes.VariableDefinition{Name: "fuelPct", Type: "Number", DefaultValue: 12.5},
		&formulatypes.VariableDefinition{Name: "lane_code", Type: "String", DefaultValue: "ATL"},
		&formulatypes.VariableDefinition{Name: "isTeam", Type: "Boolean", DefaultValue: false},
		&formulatypes.VariableDefinition{Name: "zones", Type: "Array", DefaultValue: []any{"A"}},
		&formulatypes.VariableDefinition{Name: "anything", Type: "Any"},
		&formulatypes.VariableDefinition{Name: "legacy", Type: "number", DefaultValue: 1.0},
	))
}

func TestValidateVariableDefinitions_RefusesWhatTheEngineCannotUse(t *testing.T) {
	t.Parallel()

	got := variableErrors(
		&formulatypes.VariableDefinition{Name: "2fast", Type: "Number"},
		&formulatypes.VariableDefinition{Name: "rate", Type: "Number"},
		&formulatypes.VariableDefinition{Name: "rate", Type: "Number"},
		&formulatypes.VariableDefinition{Name: "unit", Type: "Decimal"},
		&formulatypes.VariableDefinition{Name: "teams", Type: "Boolean", DefaultValue: "yes"},
		nil,
	)

	assert.Contains(t, got["variableDefinitions[0].name"], "must start with a letter")
	assert.Contains(t, got["variableDefinitions[2].name"], "declared twice")
	assert.Contains(t, got["variableDefinitions[3].type"], "Type must be")
	assert.Contains(t, got["variableDefinitions[4].defaultValue"], "Boolean")
	assert.Contains(t, got["variableDefinitions[5]"], "required")
	assert.NotContains(t, got, "variableDefinitions[1].name")
}
