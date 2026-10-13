package formula_test

import (
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func activeTemplate(expression string) *formulatemplate.FormulaTemplate {
	return &formulatemplate.FormulaTemplate{
		ID:         pulid.MustNew("ft_"),
		Name:       "service type surcharge",
		Expression: expression,
		SchemaID:   "shipment",
		Status:     formulatemplate.StatusActive,
	}
}

func TestService_Rate_ReferencedUnresolvedFieldIsAValidationError(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	_, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(`serviceType.code == "EXP" ? 75 : 0`),
		Entity:   &shipment.Shipment{},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err), "must surface as a validation problem, not a 500")
	assert.Contains(t, err.Error(), "serviceType.code")
}

func TestService_Rate_UnresolvedFieldReferencedOnlyByABreakdownFails(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	template := activeTemplate(`100`)
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "customer", Label: "Customer", Expression: `customer.code == "ACME" ? 10 : 0`},
	}

	_, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   &shipment.Shipment{ServiceType: &servicetype.ServiceType{Code: "STD"}},
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "customer.code")
}

func TestService_Rate_UnreferencedUnresolvedFieldIsIgnored(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: activeTemplate(`serviceType.code == "EXP" ? 75 : 0`),
		Entity:   &shipment.Shipment{ServiceType: &servicetype.ServiceType{Code: "EXP"}},
	})

	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(75).Equal(resp.Amount), resp.Amount.String())
}

func TestService_Rate_OverrideSuppliesAnUnresolvedField(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template:  activeTemplate(`serviceType.code == "EXP" ? 75 : 0`),
		Entity:    &shipment.Shipment{},
		Overrides: map[string]any{"serviceType.code": "EXP"},
	})

	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(75).Equal(resp.Amount), resp.Amount.String())
}

func TestService_EvaluateExpression_ReferencedUnresolvedFieldIsAValidationError(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	_, err := svc.EvaluateExpression(t.Context(), &formula.EvaluateExpressionRequest{
		Expression: `sum(customer.code == "ACME" ? 10 : 0, 5)`,
		Entity:     &shipment.Shipment{},
		SchemaID:   "shipment",
	})

	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "customer.code")
}
