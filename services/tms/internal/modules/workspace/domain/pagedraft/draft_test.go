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
			TemplateID: pulid.MustNew("ft_").String(),
			Name:       "Per Mile",
			Source:     perMileSource,
		},
	}
	assert.Empty(t, fieldsOf(t, formulaDraft))

	libraryDraft := &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormulaLibrary,
		FormulaLibrary: &pagedraft.FormulaLibrary{
			Name:   "accessorials",
			Source: "library \"org.accessorials\"\n",
		},
	}
	assert.Empty(t, fieldsOf(t, libraryDraft))
}

const perMileSource = `tariff "Per Mile"

input rate: Money/mi
input minimumCharge: Money = $0

charge "Per Mile" (quantity: shipment.distance, rate: rate)
minimum minimumCharge
`

func TestDraft_RefusesAPayloadThatIsNotItsSurfaces(t *testing.T) {
	t.Parallel()

	assert.ElementsMatch(t, []string{"context.draft.shipmentImport", "context.draft.formula"},
		fieldsOf(t, &pagedraft.Draft{
			Surface: pagedraft.SurfaceShipmentImport,
			Formula: &pagedraft.Formula{Source: perMileSource},
		}))
	assert.ElementsMatch(t, []string{"context.draft.formulaLibrary"},
		fieldsOf(t, &pagedraft.Draft{
			Surface: pagedraft.SurfaceFormula,
			Formula: &pagedraft.Formula{Source: perMileSource},
			FormulaLibrary: &pagedraft.FormulaLibrary{
				Source: "library \"org.accessorials\"\n",
			},
		}))
	assert.ElementsMatch(t, []string{"context.draft.formulaLibrary"},
		fieldsOf(t, &pagedraft.Draft{Surface: pagedraft.SurfaceFormulaLibrary}))
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
			TemplateID:  "not an id",
			Name:        strings.Repeat("n", pagedraft.MaxTemplateNameLength+1),
			Description: strings.Repeat("d", pagedraft.MaxDescriptionLength+1),
			Source:      strings.Repeat("x", pagedraft.MaxSourceLength+1),
		},
	})
	assert.ElementsMatch(t, []string{
		"context.draft.formula.templateId",
		"context.draft.formula.name",
		"context.draft.formula.description",
		"context.draft.formula.source",
	}, got)

	got = fieldsOf(t, &pagedraft.Draft{
		Surface: pagedraft.SurfaceFormulaLibrary,
		FormulaLibrary: &pagedraft.FormulaLibrary{
			LibraryID: "nope",
			Name:      strings.Repeat("n", pagedraft.MaxLibraryNameLength+1),
			Source:    strings.Repeat("x", pagedraft.MaxSourceLength+1),
		},
	})
	assert.ElementsMatch(t, []string{
		"context.draft.formulaLibrary.libraryId",
		"context.draft.formulaLibrary.name",
		"context.draft.formulaLibrary.source",
	}, got)
}

func TestDraft_NormalizesNamesAndKeepsSourceAsWritten(t *testing.T) {
	t.Parallel()

	source := "  " + perMileSource + "\n\n"
	normalized := (&pagedraft.Draft{
		Surface: " formula ",
		Formula: &pagedraft.Formula{Name: " Per Mile ", Source: source},
	}).Normalized()

	assert.Equal(t, pagedraft.SurfaceFormula, normalized.Surface)
	assert.Equal(t, "Per Mile", normalized.Formula.Name)
	assert.Equal(t, source, normalized.Formula.Source)

	library := (&pagedraft.Draft{
		Surface:        pagedraft.SurfaceFormulaLibrary,
		FormulaLibrary: &pagedraft.FormulaLibrary{Name: " accessorials ", Source: source},
	}).Normalized()
	assert.Equal(t, "accessorials", library.FormulaLibrary.Name)
	assert.Equal(t, source, library.FormulaLibrary.Source)
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
		"Proposed library":               {Action: pagedraft.ActionProposeLibrary},
	}
	for want, edit := range cases {
		assert.Equal(t, want, edit.Title())
	}

	require.True(t, pagedraft.IsEditTool("intake_set_stop_location"))
	require.False(t, pagedraft.IsEditTool("location_create"))
	assert.Equal(t, pagedraft.SurfaceFormula, pagedraft.ActionProposeFormula.Surface())
	assert.Equal(t, pagedraft.SurfaceFormulaLibrary, pagedraft.ActionProposeLibrary.Surface())
	assert.Equal(t, pagedraft.SurfaceShipmentImport, pagedraft.ActionAcceptField.Surface())
}
