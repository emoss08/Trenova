package formulatemplateservice

import (
	"strconv"
	"testing"

	"github.com/emoss08/trenova/internal/modules/rating/domain/ratematrix"
	"github.com/emoss08/trenova/internal/modules/rating/ports/ratematrixports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/i18n"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func refusals(t *testing.T, err error) map[string]*errortypes.Error {
	t.Helper()

	var multiErr *errortypes.MultiError
	require.ErrorAs(t, err, &multiErr)
	byField := make(map[string]*errortypes.Error, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		byField[fieldErr.Field] = fieldErr
	}
	return byField
}

func TestValidateTemplate_RefusesEachExpressionOnItsFieldWithItsFirstError(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()
	entity.Expression = `totalDistance * 2 + unknownThing`
	entity.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "linehaul", Label: "Linehaul", Expression: `totalDistance * 2`},
		{Name: "flag", Label: "Flag", Expression: `origin.zip == 75201`},
	}

	err := deps.svc.validateTemplate(t.Context(), entity)
	require.Error(t, err)

	byField := refusals(t, err)
	require.Len(t, byField, 2, "one refusal per expression with an error")

	main := byField["expression"]
	require.NotNil(t, main)
	assert.Equal(t, errortypes.ErrInvalid, main.Code)
	assert.Contains(t, errortypes.LocalizedText(t.Context(), main), "unknownThing")

	line := byField["breakdownDefinitions[1].expression"]
	require.NotNil(t, line, "a breakdown is refused on its own field path")
	assert.NotContains(t, main.Error(), "{0}", "the message is filled in")
	assert.NotEmpty(t, i18n.Translate(i18n.ES, line.Message, line.Args...))
}

func TestValidateTemplate_AWarningDoesNotRefuseTheSave(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()
	entity.Expression = `weight * 0.5`

	require.NoError(t, deps.svc.validateTemplate(t.Context(), entity))
}

func TestValidateTemplate_RefusesARateTableTheTenantDoesNotHave(t *testing.T) {
	t.Parallel()

	matrixID := pulid.MustNew("rmx_")
	deps := setupTestWithMatrices(t, []*ratematrixports.RateMatrixLookupData{{
		Matrix: &ratematrix.RateMatrix{
			ID:   matrixID,
			Code: "fsc",
			Dimensions: []*ratematrix.RateMatrixDimension{{
				MatchMode: ratematrix.MatchModeExact,
				Kind:      ratematrix.DimensionKindCustom,
			}},
		},
		Cells: []*ratematrix.RateMatrixCell{{
			RateMatrixID: matrixID,
			D0Key:        "DIESEL",
			Value:        decimal.RequireFromString("0.35"),
		}},
	}})

	entity := newTestTemplate()
	entity.OrganizationID = newTenantInfo().OrgID
	entity.BusinessUnitID = newTenantInfo().BuID

	entity.Expression = `lookup("fsc", "DIESEL") * totalDistance`
	require.NoError(t, deps.svc.validateTemplate(t.Context(), entity))

	entity.Expression = `lookup("nope", "DIESEL") * totalDistance`
	err := deps.svc.validateTemplate(t.Context(), entity)
	require.Error(t, err)
	assert.Contains(t, errortypes.LocalizedText(t.Context(), refusals(t, err)["expression"]), "nope")
}

func TestValidateTemplate_RefusesAnInputNamedAfterAFunctionOrAField(t *testing.T) {
	t.Parallel()
	deps := setupTest(t)

	entity := newTestTemplate()
	entity.VariableDefinitions = append(entity.VariableDefinitions,
		&formulatypes.VariableDefinition{Name: "round", Type: formulatypes.VariableValueTypeNumber},
		&formulatypes.VariableDefinition{Name: "weight", Type: formulatypes.VariableValueTypeNumber},
	)

	err := deps.svc.validateTemplate(t.Context(), entity)
	require.Error(t, err)
	byField := refusals(t, err)
	count := len(entity.VariableDefinitions)
	assert.Contains(t, byField[variableNameField(count-2)].Error(), "formula function")
	assert.Contains(t, byField[variableNameField(count-1)].Error(), "already a shipment value")
}

func variableNameField(index int) string {
	return "variableDefinitions[" + strconv.Itoa(index) + "].name"
}
