package pagedraft_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/workspace/domain/pagedraft"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func fieldsOf(t *testing.T, draft *pagedraft.Draft) []string {
	t.Helper()

	multiErr := errortypes.NewMultiError()
	draft.Validate("context.draft", multiErr)
	fields := make([]string, 0, len(multiErr.Errors))
	for _, fieldErr := range multiErr.Errors {
		fields = append(fields, fieldErr.Field)
	}

	return fields
}

func TestDraft_AcceptsWhatAPageShows(t *testing.T) {
	t.Parallel()

	importDraft := &pagedraft.Draft{
		Surface: pagedraft.SurfaceShipmentImport,
		ShipmentImport: &pagedraft.ShipmentImport{
			Fields: []pagedraft.ImportField{
				{Key: "rate", Label: "Rate", Value: "1200", Confidence: 0.9,
					Status: pagedraft.FieldAccepted},
			},
			Required: pagedraft.RequiredFields{CustomerID: pulid.MustNew("cus_").String()},
			Stops:    []pagedraft.ImportStop{{Role: pagedraft.StopPickup, Confidence: 1}},
		},
	}
	assert.Empty(t, fieldsOf(t, importDraft))

	formulaDraft := &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{
			SchemaID:   "shipment",
			Expression: "totalDistance * 2",
			Variables: []pagedraft.FormulaVariable{
				{Name: "perMile", Type: pagedraft.VariableNumber, DefaultValue: 2.0},
				{Name: "hazmat", Type: pagedraft.VariableBoolean, DefaultValue: true},
			},
		},
	}
	assert.Empty(t, fieldsOf(t, formulaDraft))
}

func TestDraft_RefusesAPayloadThatIsNotItsSurfaces(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{"context.draft.shipmentImport", "context.draft.formula"},
		fieldsOf(t, &pagedraft.Draft{
			Surface: pagedraft.SurfaceShipmentImport,
			Formula: &pagedraft.Formula{SchemaID: "shipment"},
		}))
	assert.Contains(t, fieldsOf(t, &pagedraft.Draft{Surface: "spreadsheet"}),
		"context.draft.surface")
}

func TestDraft_BoundsEveryText(t *testing.T) {
	t.Parallel()

	fields := make([]pagedraft.ImportField, pagedraft.MaxImportFields+1)
	for idx := range fields {
		fields[idx] = pagedraft.ImportField{Key: "k", Status: pagedraft.FieldAccepted}
	}
	assert.Contains(t, fieldsOf(t, &pagedraft.Draft{
		Surface:        pagedraft.SurfaceShipmentImport,
		ShipmentImport: &pagedraft.ShipmentImport{Fields: fields},
	}), "context.draft.shipmentImport.fields")

	got := fieldsOf(t, &pagedraft.Draft{
		Surface: pagedraft.SurfaceShipmentImport,
		ShipmentImport: &pagedraft.ShipmentImport{
			Fields: []pagedraft.ImportField{
				{Key: "rate; drop", Status: pagedraft.FieldAccepted},
				{Key: "bol", Value: strings.Repeat("x", pagedraft.MaxFieldValueLength+1),
					Status: pagedraft.FieldAccepted, Confidence: 2},
				{Key: "bol", Status: "done"},
			},
			Required: pagedraft.RequiredFields{CustomerID: "not an id"},
			Stops:    []pagedraft.ImportStop{{Role: "crossdock", LocationID: "nope"}},
		},
	})
	assert.ElementsMatch(t, []string{
		"context.draft.shipmentImport.fields[0].key",
		"context.draft.shipmentImport.fields[1].value",
		"context.draft.shipmentImport.fields[1].confidence",
		"context.draft.shipmentImport.fields[2].key",
		"context.draft.shipmentImport.fields[2].status",
		"context.draft.shipmentImport.required.customerId",
		"context.draft.shipmentImport.stops[0].role",
		"context.draft.shipmentImport.stops[0].locationId",
	}, got)

	got = fieldsOf(t, &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{
			SchemaID:   "",
			Expression: strings.Repeat("1", pagedraft.MaxExpressionLength+1),
			Variables: []pagedraft.FormulaVariable{
				{Name: "2fast", Type: pagedraft.VariableNumber},
				{Name: "rate", Type: "Money", DefaultValue: map[string]any{"a": 1}},
			},
		},
	})
	assert.ElementsMatch(t, []string{
		"context.draft.formula.schemaId",
		"context.draft.formula.expression",
		"context.draft.formula.variables[0].name",
		"context.draft.formula.variables[1].type",
		"context.draft.formula.variables[1].defaultValue",
	}, got)
}

func TestFormulaPolicy_IsCheckedOnTheDraftAndTheProposal(t *testing.T) {
	t.Parallel()

	minCharge, maxCharge, badAmount := "250", "1000", "two fifty"
	negative, precision, tooPrecise := "-5", 2, 5

	sound := &pagedraft.FormulaPolicy{
		MinCharge:         &minCharge,
		MaxCharge:         &maxCharge,
		RoundingMode:      "HalfEven",
		RoundingPrecision: &precision,
	}
	assert.Empty(t, fieldsOf(t, &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{
			SchemaID:    "shipment",
			Name:        "Per mile",
			Description: "Linehaul by the mile.",
			Expression:  "totalDistance * perMile",
			Breakdown: []pagedraft.BreakdownLine{
				{Name: "linehaul", Label: "Linehaul", Expression: "totalDistance * perMile"},
			},
			Policy: sound,
		},
	}))

	got := fieldsOf(t, &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{
			SchemaID:    "shipment",
			Name:        strings.Repeat("n", pagedraft.MaxTemplateNameLength+1),
			Description: strings.Repeat("d", pagedraft.MaxVariableTextLength+1),
			Breakdown: []pagedraft.BreakdownLine{
				{Name: "_hidden", Label: "", Expression: "1"},
				{Name: "line", Label: "Line", Expression: ""},
				{Name: "line", Label: "Again", Expression: "2"},
			},
			Policy: &pagedraft.FormulaPolicy{
				MinCharge:         &badAmount,
				MaxCharge:         &negative,
				RoundingMode:      "Nearest",
				RoundingPrecision: &tooPrecise,
			},
		},
	})
	assert.ElementsMatch(t, []string{
		"context.draft.formula.name",
		"context.draft.formula.description",
		"context.draft.formula.breakdown[0].name",
		"context.draft.formula.breakdown[0].label",
		"context.draft.formula.breakdown[1].expression",
		"context.draft.formula.breakdown[2].name",
		"context.draft.formula.policy.minCharge",
		"context.draft.formula.policy.maxCharge",
		"context.draft.formula.policy.roundingMode",
		"context.draft.formula.policy.roundingPrecision",
	}, got)

	multiErr := errortypes.NewMultiError()
	(&pagedraft.FormulaDetails{Policy: &pagedraft.FormulaPolicy{
		MinCharge: &maxCharge,
		MaxCharge: &minCharge,
	}}).Validate("", multiErr)
	require.Len(t, multiErr.Errors, 1)
	assert.Equal(t, "policy.minCharge", multiErr.Errors[0].Field,
		"a minimum above the maximum is refused once, on the minimum")

	multiErr = errortypes.NewMultiError()
	(&pagedraft.FormulaDetails{Policy: &pagedraft.FormulaPolicy{RoundingMode: "None"}}).
		Validate("", multiErr)
	assert.False(t, multiErr.HasErrors(), "every rounding mode the engine knows is accepted")
}

func TestFormulaPolicy_ResolvesToWhatTheEngineApplies(t *testing.T) {
	t.Parallel()

	var none *pagedraft.FormulaPolicy
	two := 2
	assert.Equal(t, &pagedraft.FormulaPolicy{RoundingMode: "HalfUp", RoundingPrecision: &two},
		none.Resolved(), "no policy is the default rounding and no bounds")

	minCharge, zero := " 250.00 ", 0
	resolved := (&pagedraft.FormulaPolicy{
		MinCharge:         &minCharge,
		RoundingMode:      "Up",
		RoundingPrecision: &zero,
	}).Resolved()
	require.NotNil(t, resolved.MinCharge)
	assert.Equal(t, "250", *resolved.MinCharge)
	assert.Nil(t, resolved.MaxCharge)
	assert.Equal(t, "Up", resolved.RoundingMode)
	require.NotNil(t, resolved.RoundingPrecision)
	assert.Equal(t, 0, *resolved.RoundingPrecision, "zero places next to a chosen mode is kept")

	charge := (&pagedraft.FormulaPolicy{MaxCharge: &minCharge}).ChargePolicy()
	assert.False(t, charge.MinCharge.Valid)
	assert.True(t, charge.MaxCharge.Valid)
	assert.Equal(t, "250", charge.MaxCharge.Decimal.String())
	assert.Equal(t, "HalfUp", charge.RoundingMode.String())
	assert.Equal(t, int32(2), charge.RoundingPrecision)

	normalized := (&pagedraft.Draft{
		Surface: pagedraft.SurfaceFormula,
		Formula: &pagedraft.Formula{
			SchemaID: "shipment",
			Name:     " Per mile ",
			Breakdown: []pagedraft.BreakdownLine{
				{Name: " linehaul ", Label: " Linehaul ", Expression: " totalDistance "},
			},
			Policy: &pagedraft.FormulaPolicy{MinCharge: &minCharge, RoundingMode: " Down "},
		},
	}).Normalized()
	assert.Equal(t, "Per mile", normalized.Formula.Name)
	assert.Equal(t, "linehaul", normalized.Formula.Breakdown[0].Name)
	assert.Equal(t, "250.00", *normalized.Formula.Policy.MinCharge)
	assert.Equal(t, "Down", normalized.Formula.Policy.RoundingMode)
	assert.Nil(t, normalized.Formula.Policy.RoundingPrecision)
}

func TestImportStop_HasSchedule(t *testing.T) {
	t.Parallel()

	for date, want := range map[string]bool{
		"":                     false,
		"06:00-22:00":          false,
		"2026-10-01 08:00":     true,
		"2026-10-01":           true,
		"1790000000":           true,
		"0":                    false,
		"Tuesday":              false,
		"2026-10-01T08:00:00Z": true,
	} {
		stop := pagedraft.ImportStop{Date: date}
		assert.Equal(t, want, stop.HasSchedule(), date)
	}
}

func TestEditTitles(t *testing.T) {
	t.Parallel()

	one := 1
	cases := map[string]pagedraft.Edit{
		"Accepted Rate": {Action: pagedraft.ActionAcceptField, FieldKey: "rate", Label: "Rate"},
		"Set bol":       {Action: pagedraft.ActionSetFieldValue, FieldKey: "bol"},
		"Set the customer to Acme Foods": {
			Action: pagedraft.ActionSetRequiredField, FieldKey: "customerId", Label: "Acme Foods",
		},
		"Matched stop 2 to Reno DC": {
			Action: pagedraft.ActionSetStopLocation, StopIndex: &one, Label: "Reno DC",
		},
		"Scheduled stop 2": {
			Action:    pagedraft.ActionSetStopSchedule,
			StopIndex: &one,
		},
		"Accepted every confident field": {Action: pagedraft.ActionAcceptAllConfident},
		"Proposed formula":               {Action: pagedraft.ActionProposeFormula},
	}
	for want, edit := range cases {
		assert.Equal(t, want, edit.Title())
	}

	require.True(t, pagedraft.IsEditTool("intake_set_stop_location"))
	require.False(t, pagedraft.IsEditTool("location_create"))
	assert.Equal(t, pagedraft.SurfaceFormula, pagedraft.ActionProposeFormula.Surface())
	assert.Equal(t, pagedraft.SurfaceShipmentImport, pagedraft.ActionAcceptField.Surface())
}
