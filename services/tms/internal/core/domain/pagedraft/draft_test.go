package pagedraft_test

import (
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/pagedraft"
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

	require.True(t, pagedraft.IsEditTool("set_stop_location"))
	require.False(t, pagedraft.IsEditTool("create_location"))
	assert.Equal(t, pagedraft.SurfaceFormula, pagedraft.ActionProposeFormula.Surface())
	assert.Equal(t, pagedraft.SurfaceShipmentImport, pagedraft.ActionAcceptField.Surface())
}
