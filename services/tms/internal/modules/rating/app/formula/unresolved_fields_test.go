package formula_test

import (
	"context"
	"testing"

	"github.com/emoss08/trenova/internal/modules/operations/domain/shipment"
	"github.com/emoss08/trenova/internal/modules/rating/app/formula"
	"github.com/emoss08/trenova/internal/modules/rating/domain/formulatemplate"
	"github.com/emoss08/trenova/internal/modules/rating/ports/formulatemplateports"
	"github.com/emoss08/trenova/internal/modules/reference/domain/servicetype"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/pkg/formulatemplatetypes"
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

func TestService_Rate_AnUnresolvedFieldReadOnlyByABreakdownFailsThatLine(t *testing.T) {
	t.Parallel()

	svc := setupService(t)
	template := activeTemplate(`100`)
	template.BreakdownDefinitions = []*formulatypes.BreakdownDefinition{
		{Name: "customer", Label: "Customer", Expression: `customer.code == "ACME" ? 10 : 0`},
	}

	resp, err := svc.Rate(t.Context(), &formula.RateRequest{
		Template: template,
		Entity:   &shipment.Shipment{ServiceType: &servicetype.ServiceType{Code: "STD"}},
	})

	require.NoError(t, err, "the bill still prices; only the line that cannot is marked")
	assert.Equal(t, "100", resp.Amount.String())
	require.Len(t, resp.Breakdown, 1)
	assert.Contains(t, resp.Breakdown[0].Error, "customer.code")
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

func TestService_Preview_AFieldTheRecordCouldNotSupplyIsNamedNotPricedAsEmpty(t *testing.T) {
	t.Parallel()

	svc := setupService(t)

	preview, err := svc.Preview(t.Context(), &formula.PreviewRequest{
		Expression: `sum(customer.code == "ACME" ? 10 : 0, 5)`,
		Entity:     &shipment.Shipment{},
		SchemaID:   "shipment",
	})

	require.NoError(t, err)
	require.NotNil(t, preview.Error)
	assert.Equal(t, formula.MainScope, preview.Error.Scope)
	assert.Equal(t, formula.CodeUnresolvedField, preview.Error.Err.Code)
	assert.Equal(t, []string{"customer.code"}, preview.Error.Err.Fields)
	assert.Contains(t, preview.Error.Err.Message.String(), "could not be read from this record")
}

func TestService_Calculate_RefusesATemplateReadingAnUnavailableInput(t *testing.T) {
	t.Parallel()

	template := activeTemplate(`round(sellTotal * (baseRate / 100), 2)`)
	template.VariableDefinitions = []*formulatypes.VariableDefinition{
		{Name: "sellTotal", Type: "number", DefaultValue: 0.0},
	}
	svc := setupServiceWithRepo(t, &singleTemplateRepo{template: template})

	unavailable := map[string]string{
		"sellTotal": "sellTotal is only known when a carrier is priced against the customer's charge",
	}

	_, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID:        template.ID,
		Entity:            &shipment.Shipment{ServiceType: &servicetype.ServiceType{Code: "STD"}},
		Overrides:         map[string]any{"baseRate": 85.0},
		UnavailableInputs: unavailable,
	})
	require.Error(t, err)
	assert.True(t, errortypes.IsError(err))
	assert.Contains(t, err.Error(), "only known when a carrier is priced")

	resp, err := svc.Calculate(t.Context(), &formulatemplatetypes.CalculateRequest{
		TemplateID: template.ID,
		Entity:     &shipment.Shipment{ServiceType: &servicetype.ServiceType{Code: "STD"}},
		Overrides:  map[string]any{"baseRate": 85.0, "sellTotal": 2_000.0},
	})
	require.NoError(t, err)
	assert.True(t, decimal.NewFromInt(1_700).Equal(resp.Amount), resp.Amount.String())
}

type singleTemplateRepo struct {
	formulatemplateports.FormulaTemplateRepository
	template *formulatemplate.FormulaTemplate
}

func (r *singleTemplateRepo) GetByID(
	_ context.Context,
	_ formulatemplateports.GetFormulaTemplateByIDRequest,
) (*formulatemplate.FormulaTemplate, error) {
	return r.template, nil
}
