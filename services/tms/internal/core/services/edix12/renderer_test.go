package edix12

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender204_ResolvesRepeatPartnerAndDateTimeFields(t *testing.T) {
	t.Parallel()

	start := time.Date(2026, 5, 16, 14, 30, 0, 0, time.UTC).Unix()
	input := renderInput(edi.ValidationModeStrict)
	input.Profile.PartnerSettings = map[string]any{
		"carrier": map[string]any{"scac": "ABCD"},
		"contact": map[string]any{
			"name":  "Jane Dispatcher",
			"phone": "5551212",
		},
	}
	input.Payload.ShipmentID = pulid.MustNew("shp_")
	input.Payload.BOL = "BOL-1"
	input.Payload.Moves = []edi.LoadTenderMove{
		{
			Sequence: 1,
			Stops: []edi.LoadTenderStop{
				{
					Type:                 "LD",
					Sequence:             1,
					LocationName:         "Chicago Dock",
					LocationAddressLine1: "100 Main",
					LocationCity:         "Chicago",
					LocationStateCode:    "IL",
					LocationPostalCode:   "60601",
					ScheduledWindowStart: start,
				},
			},
		},
	}
	input.Payload.Commodities = []edi.LoadTenderCommodity{
		{CommodityDescription: "Palletized freight"},
	}

	result, err := Render204(input)

	require.NoError(t, err)
	assert.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "B2*ABCD*"+input.Payload.ShipmentID.String())
	assert.Contains(t, result.RawX12, "G62*37*20260516*I*1430")
	assert.Contains(t, result.RawX12, "N1*LD*Chicago Dock")
	assert.Contains(t, result.RawX12, "G61*IC*Jane Dispatcher*TE*5551212")
	assert.Contains(t, result.RawX12, "L5**Palletized freight")
}

func TestRender204_AppliesCustomSeparatorsAndTrailerCounts(t *testing.T) {
	t.Parallel()

	input := renderInput(edi.ValidationModeStrict)
	input.Profile.Envelope.ElementSeparator = "|"
	input.Profile.Envelope.SegmentTerminator = "!"
	input.Payload.ShipmentID = pulid.MustNew("shp_")
	input.Payload.Moves = []edi.LoadTenderMove{
		{
			Sequence: 1,
			Stops: []edi.LoadTenderStop{
				{Type: "LD", Sequence: 1},
			},
		},
	}

	result, err := Render204(input)

	require.NoError(t, err)
	assert.Contains(t, result.RawX12, "ST|204|0000!")
	assert.Contains(t, result.RawX12, "SE|12|0000!")
	assert.True(t, strings.HasSuffix(result.RawX12, "!"))
	assert.Equal(t, int64(16), result.SegmentCount)
}

func TestRender204_PreservesElementPositionsAndISAFixedWidths(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Profile.Envelope.InterchangeSenderID = "trenova"
	input.Profile.Envelope.InterchangeReceiverID = "partner"
	input.Runtime = RuntimeValues(input.Profile, edi.DefaultX12204Version)
	SetProvisionalControlNumbers(input.Runtime)

	result, err := Render204(input)

	require.NoError(t, err)
	segments := strings.Split(result.RawX12, input.Profile.Envelope.SegmentTerminator)
	require.NotEmpty(t, segments)
	isa := strings.Split(segments[0], input.Profile.Envelope.ElementSeparator)
	require.Len(t, isa, 17)
	assert.Len(t, isa[2], 10)
	assert.Len(t, isa[4], 10)
	assert.Len(t, isa[6], 15)
	assert.Equal(t, "TRENOVA        ", isa[6])
	assert.Len(t, isa[8], 15)
	assert.Equal(t, "PARTNER        ", isa[8])
	assert.Contains(t, result.RawX12, "B2**"+input.Payload.ShipmentID.String()+"**PP")
}

func TestRender204_TrailerCountsAndControlNumbersUseRenderedEnvelope(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Runtime["isaControlNumber"] = "000000321"
	input.Runtime["groupControlNumber"] = "77"
	input.Runtime["transactionControlNumber"] = "0077"

	result, err := Render204(input)

	require.NoError(t, err)
	assert.Contains(t, result.RawX12, "ST*204*0077~")
	assert.Contains(t, result.RawX12, "SE*12*0077~")
	assert.Contains(t, result.RawX12, "GE*1*77~")
	assert.Contains(t, result.RawX12, "IEA*1*000000321~")
}

func TestRender204_SanitizesAllConfiguredSeparators(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.StarlarkScript = `def value(ctx):
    return "A*B~C>D^E"`

	result, err := Render204(input)

	require.NoError(t, err)
	assert.Contains(t, result.RawX12, "L11*A B C D E*BM~")
}

func TestRender204_FiltersDiagnosticsByValidationMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mode         edi.ValidationMode
		wantCount    int
		wantSeverity edi.ValidationSeverity
	}{
		{
			name:         "strict keeps errors",
			mode:         edi.ValidationModeStrict,
			wantCount:    2,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name:         "warn only downgrades errors",
			mode:         edi.ValidationModeWarnOnly,
			wantCount:    2,
			wantSeverity: edi.ValidationSeverityWarning,
		},
		{
			name:      "disabled suppresses field diagnostics",
			mode:      edi.ValidationModeDisabled,
			wantCount: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := renderInput(tt.mode)
			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, tt.wantCount)
			for _, diagnostic := range result.Diagnostics {
				assert.Equal(t, tt.wantSeverity, diagnostic.Severity)
			}
		})
	}
}

func TestRender204_SegmentPathConditionRendersWhenTruthy(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = "BOL-1"
	findSegment(t, input, "L11").Condition = "shipment.bol"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*BOL-1*BM")
}

func TestRender204_SegmentPathConditionSkipsWhenFalsey(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	findSegment(t, input, "L11").Condition = "shipment.bol"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.NotContains(t, result.RawX12, "L11")
}

func TestRender204_NegatedPathConditionRendersWhenFalsey(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	findSegment(t, input, "L11").Condition = "!shipment.bol"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11**BM")
}

func TestRender204_StringComparisonConditionsSupportQuotedEmptyValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		bol       string
		condition string
		wantRaw   string
	}{
		{
			name:      "single quoted inequality",
			bol:       "BOL-1",
			condition: `shipment.bol != ''`,
			wantRaw:   "L11*BOL-1*BM",
		},
		{
			name:      "double quoted empty equality",
			condition: `shipment.bol == ""`,
			wantRaw:   "L11**BM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			input.Payload.BOL = tt.bol
			findSegment(t, input, "L11").Condition = tt.condition

			result, err := Render204(input)

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			assert.Contains(t, result.RawX12, tt.wantRaw)
		})
	}
}

func TestRender204_RepeatedSegmentConditionFiltersRepeatItems(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.Moves[0].Stops = []edi.LoadTenderStop{
		{Type: "LD", Sequence: 1, LocationName: "Load Dock"},
		{Type: "UL", Sequence: 2, LocationName: "Unload Dock"},
	}
	findSegment(t, input, "N1").Condition = `repeat.type == "LD"`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "N1*LD*Load Dock")
	assert.NotContains(t, result.RawX12, "N1*UL*Unload Dock")
}

func TestRender204_RepeatedSegmentConditionSkipsNonMatchingComparison(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	findSegment(t, input, "N1").Condition = `repeat.type == "ZZ"`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.NotContains(t, result.RawX12, "N1*")
}

func TestRender204_ElementConditionRendersAndBlanks(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		bol        string
		wantRaw    string
		notWantRaw string
		condition  string
	}{
		{
			name:       "renders when true",
			bol:        "BOL-1",
			wantRaw:    "L11*BOL-1*BM",
			condition:  "shipment.bol",
			notWantRaw: "L11**BM",
		},
		{
			name:       "blanks when false",
			wantRaw:    "L11**BM",
			condition:  "shipment.bol",
			notWantRaw: "L11*BOL-1*BM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			input.Payload.BOL = tt.bol
			findElement(t, input, "L11", 0).Condition = tt.condition

			result, err := Render204(input)

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			assert.Contains(t, result.RawX12, tt.wantRaw)
			assert.NotContains(t, result.RawX12, tt.notWantRaw)
		})
	}
}

func TestRender204_RequiredElementFalseConditionDoesNotValidateRequired(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "B2", 1)
	element.Condition = "shipment.bol"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "B2****PP")
}

func TestRender204_InvalidConditionEmitsConditionError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		condition string
	}{
		{
			name:      "invalid root",
			condition: "unknown.bol",
		},
		{
			name:      "unsupported boolean operator",
			condition: `shipment.bol && partner.carrier.scac`,
		},
		{
			name:      "unquoted comparison value",
			condition: `shipment.bol == BOL-1`,
		},
		{
			name:      "incomplete comparison",
			condition: `shipment.bol ==`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			findSegment(t, input, "L11").Condition = tt.condition

			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, 1)
			diagnostic := result.Diagnostics[0]
			assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
			assert.Equal(t, "condition_error", diagnostic.Code)
			assert.Equal(t, "L11", diagnostic.SegmentID)
			assert.Equal(t, 0, diagnostic.ElementPosition)
			assert.Equal(t, tt.condition, diagnostic.Path)
			assert.Equal(t, conditionSuggestedFix, diagnostic.SuggestedFix)
		})
	}
}

func TestRender204_ElementConditionErrorIncludesElementPosition(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	findElement(t, input, "L11", 0).Condition = "unknown.bol"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, "condition_error", diagnostic.Code)
	assert.Equal(t, "L11", diagnostic.SegmentID)
	assert.Equal(t, 1, diagnostic.ElementPosition)
	assert.Equal(t, "unknown.bol", diagnostic.Path)
	assert.Contains(t, result.RawX12, "L11**BM")
}

func TestRender204_ConditionDiagnosticsRespectValidationMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		mode         edi.ValidationMode
		wantSeverity edi.ValidationSeverity
	}{
		{
			name:         "disabled preserves condition error",
			mode:         edi.ValidationModeDisabled,
			wantSeverity: edi.ValidationSeverityError,
		},
		{
			name:         "warn only downgrades condition error",
			mode:         edi.ValidationModeWarnOnly,
			wantSeverity: edi.ValidationSeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(tt.mode)
			findSegment(t, input, "L11").Condition = "unknown.bol"

			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, 1)
			assert.Equal(t, "condition_error", result.Diagnostics[0].Code)
			assert.Equal(t, tt.wantSeverity, result.Diagnostics[0].Severity)
		})
	}
}

func TestHasBlockingDiagnostics_BlocksStrictConditionDiagnostics(t *testing.T) {
	t.Parallel()

	diagnostics := []Diagnostic{
		{
			Severity: edi.ValidationSeverityError,
			Code:     "condition_error",
		},
	}

	assert.True(t, HasBlockingDiagnostics(diagnostics, edi.ValidationModeStrict))
	assert.False(t, HasBlockingDiagnostics(diagnostics, edi.ValidationModeWarnOnly))
}

func TestRender204_StarlarkSegmentCondition(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		condition   string
		wantSegment bool
	}{
		{
			name: "true includes segment",
			condition: `starlark:def include(ctx):
    return True`,
			wantSegment: true,
		},
		{
			name: "false skips segment",
			condition: `starlark:def include(ctx):
    return False`,
			wantSegment: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			input.Payload.BOL = "BOL-1"
			findSegment(t, input, "L11").Condition = tt.condition

			result, err := Render204(input)

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			if tt.wantSegment {
				assert.Contains(t, result.RawX12, "L11*BOL-1*BM")
				return
			}
			assert.NotContains(t, result.RawX12, "L11")
		})
	}
}

func TestRender204_StarlarkRepeatConditionFiltersItems(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.Moves[0].Stops = []edi.LoadTenderStop{
		{Type: "LD", Sequence: 1, LocationName: "Load Dock"},
		{Type: "UL", Sequence: 2, LocationName: "Unload Dock"},
	}
	findSegment(t, input, "N1").Condition = `starlark:def include(ctx, item):
    return item["type"] == "LD"`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "N1*LD*Load Dock")
	assert.NotContains(t, result.RawX12, "N1*UL*Unload Dock")
}

func TestRender204_StarlarkConditionRuntimeErrorEmitsConditionError(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	findSegment(t, input, "L11").Condition = `starlark:def include(ctx):
    return 1 / 0`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
	assert.Equal(t, "condition_error", diagnostic.Code)
	assert.Equal(t, "L11", diagnostic.SegmentID)
	assert.Equal(t, starlarkConditionPath(), diagnostic.Path)
	assert.Contains(t, diagnostic.Message, "starlark_runtime_error")
	assert.Contains(t, diagnostic.Message, "floating-point division by zero")
	assert.Equal(t, conditionSuggestedFix, diagnostic.SuggestedFix)
}

func TestRender204_StarlarkConditionStepLimitBlocksInStrictMode(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	findSegment(t, input, "L11").Condition = `starlark:def include(ctx):
    while True:
        pass`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, "condition_error", result.Diagnostics[0].Code)
	assert.Contains(t, result.Diagnostics[0].Message, "starlark_step_limit")
	assert.Contains(t, result.Diagnostics[0].Message, "execution step limit exceeded")
	assert.True(t, HasBlockingDiagnostics(result.Diagnostics, edi.ValidationModeStrict))
}

func TestRender204_StarlarkConstantScalarRenders(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.StarlarkScript = `def value(ctx):
    return "STAR-REF"`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*STAR-REF*BM")
}

func TestRender204_StarlarkReadsShipmentContext(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = "BOL-STARK"
	element := findElement(t, input, "B2", 2)
	element.Source = edi.TemplateElementSourceStarlark
	element.StarlarkScript = `def value(ctx):
    return ctx["shipment"]["bol"]`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "B2**"+input.Payload.ShipmentID.String()+"**BOL-STARK")
}

func TestRender204_StarlarkRepeatValueRenders(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		script string
	}{
		{
			name: "second function argument",
			script: `def value(ctx, item):
    return item["locationName"]`,
		},
		{
			name: "ctx item alias",
			script: `def value(ctx):
    return ctx["item"]["locationName"]`,
		},
		{
			name: "ctx repeat alias",
			script: `def value(ctx):
    return ctx["repeat"]["locationName"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			input.Payload.Moves[0].Stops[0].LocationName = "Starlark Dock"
			element := findElement(t, input, "N1", 1)
			element.Source = edi.TemplateElementSourceStarlark
			element.StarlarkScript = tt.script

			result, err := Render204(input)

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			assert.Contains(t, result.RawX12, "N1*LD*Starlark Dock")
		})
	}
}

func TestRender204_StarlarkNoneFallsThroughToRequiredValidation(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "B2", 1)
	element.Source = edi.TemplateElementSourceStarlark
	element.StarlarkScript = `def value(ctx):
    return None`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
	assert.Equal(t, "required", diagnostic.Code)
	assert.Equal(t, "B2", diagnostic.SegmentID)
	assert.Equal(t, 2, diagnostic.ElementPosition)
	assert.Equal(t, "starlark:value", diagnostic.Path)
	assert.Contains(t, diagnostic.Message, "Shipment Identification Number is required")
}

func TestRender204_StarlarkMaxLengthWarnsAndTruncates(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.StarlarkScript = `def value(ctx):
    return "REFERENCE-TOO-LONG"`
	element.Validation.MaxLength = 9

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, edi.ValidationSeverityWarning, diagnostic.Severity)
	assert.Equal(t, "max_length", diagnostic.Code)
	assert.Equal(t, "L11", diagnostic.SegmentID)
	assert.Equal(t, 1, diagnostic.ElementPosition)
	assert.Contains(t, diagnostic.Message, "Reference Identification exceeds max length 9")
	assert.Contains(t, result.RawX12, "L11*REFERENCE*BM")
}

func TestRender204_StarlarkValueSanitizesSeparators(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.StarlarkScript = `def value(ctx):
    return "A*B~C>D"`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*A B C D*BM~")
}

func TestRenderX12_StarterTemplatesRenderTransactionSets(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		transactionSet edi.TransactionSet
		payload        edi.DocumentPayload
	}{
		{
			name:           "204 load tender",
			transactionSet: edi.TransactionSet204,
			payload: edi.NewLoadTenderDocumentPayload(edi.LoadTenderPayload{
				ShipmentID: pulid.MustNew("shp_"),
				BOL:        "BOL-204",
			}),
		},
		{
			name:           "210 invoice",
			transactionSet: edi.TransactionSet210,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet210,
				FreightInvoice: &edi.FreightInvoicePayload{
					InvoiceID:     pulid.MustNew("inv_"),
					InvoiceNumber: "INV-210",
					BOL:           "BOL-210",
					CurrencyCode:  "USD",
					TotalAmount: decimal.NullDecimal{
						Decimal: decimal.NewFromInt(100),
						Valid:   true,
					},
					LineCharges: []edi.FreightInvoiceCharge{
						{Sequence: 1, Description: "Linehaul", Amount: decimal.NewFromInt(100)},
					},
				},
			},
		},
		{
			name:           "214 shipment status",
			transactionSet: edi.TransactionSet214,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet214,
				ShipmentStatus: &edi.ShipmentStatusPayload{
					ShipmentID: pulid.MustNew("shp_"),
					BOL:        "BOL-214",
					StatusCode: "X3",
				},
			},
		},
		{
			name:           "990 tender response",
			transactionSet: edi.TransactionSet990,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet990,
				TenderResponse: &edi.TenderResponsePayload{
					ShipmentID:   pulid.MustNew("shp_"),
					BOL:          "BOL-990",
					ResponseCode: "A",
				},
			},
		},
		{
			name:           "997 functional acknowledgment",
			transactionSet: edi.TransactionSet997,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet997,
				FunctionalAcknowledgment: &edi.FunctionalAcknowledgmentPayload{
					OriginalFunctionalGroupID:        "SM",
					OriginalGroupControlNumber:       "7",
					OriginalTransactionSet:           edi.TransactionSet204,
					OriginalTransactionControlNumber: "0007",
					GroupAcknowledgmentCode:          "A",
					TransactionAcknowledgmentCode:    "A",
					AcceptedTransactionSetCount:      1,
					ReceivedTransactionSetCount:      1,
					IncludedTransactionSetCount:      1,
				},
			},
		},
		{
			name:           "999 implementation acknowledgment",
			transactionSet: edi.TransactionSet999,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet999,
				ImplementationAcknowledgment: &edi.ImplementationAckPayload{
					OriginalFunctionalGroupID:        "SM",
					OriginalGroupControlNumber:       "9",
					OriginalTransactionSet:           edi.TransactionSet204,
					OriginalTransactionControlNumber: "0009",
					GroupAcknowledgmentCode:          "A",
					TransactionAcknowledgmentCode:    "A",
					AcceptedTransactionSetCount:      1,
					ReceivedTransactionSetCount:      1,
					IncludedTransactionSetCount:      1,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			versionID := pulid.MustNew("editv_")
			segments, err := editemplates.StarterSegments(
				pagination.TenantInfo{},
				versionID,
				tt.transactionSet,
			)
			require.NoError(t, err)
			profile := &edi.EDIPartnerDocumentProfile{
				TransactionSet:    tt.transactionSet,
				FunctionalGroupID: edi.FunctionalGroupDefault(tt.transactionSet),
				Envelope:          edi.DefaultX12EnvelopeSettings(),
				ValidationMode:    edi.ValidationModeWarnOnly,
				PartnerSettings: map[string]any{
					"carrier": map[string]any{"scac": "TEST"},
				},
			}
			runtime := RuntimeValues(profile, "004010")
			runtime["isaControlNumber"] = "000000001"
			runtime["groupControlNumber"] = "1"
			runtime["transactionControlNumber"] = "0001"

			result, err := RenderX12(&RenderInput{
				Profile: profile,
				TemplateVersion: &edi.EDITemplateVersion{
					Segments: segments,
				},
				DocumentPayload: tt.payload,
				Runtime:         runtime,
			})
			require.NoError(t, err)
			assert.Contains(t, result.RawX12, "ST*"+string(tt.transactionSet)+"*0001")
			assert.Contains(t, result.RawX12, "GE*1*1")
			assert.Contains(t, result.RawX12, "IEA*1*000000001")
			assert.Contains(t, result.RawX12, "SE*")
			assert.Positive(t, result.SegmentCount)
		})
	}
}

func TestRenderX12_StarlarkReadsDocumentRoots(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		transactionSet edi.TransactionSet
		payload        edi.DocumentPayload
		script         string
		want           string
	}{
		{
			name:           "invoice",
			transactionSet: edi.TransactionSet210,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet210,
				FreightInvoice: &edi.FreightInvoicePayload{
					InvoiceNumber: "INV-ROOT",
				},
			},
			script: "def value(ctx):\n    return ctx[\"invoice\"][\"invoiceNumber\"]",
			want:   "TST*INV-ROOT~",
		},
		{
			name:           "shipment status",
			transactionSet: edi.TransactionSet214,
			payload: edi.DocumentPayload{
				TransactionSet: edi.TransactionSet214,
				ShipmentStatus: &edi.ShipmentStatusPayload{
					StatusCode: "D1",
				},
			},
			script: "def value(ctx):\n    return ctx[\"shipmentStatus\"][\"statusCode\"]",
			want:   "TST*D1~",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			profile := &edi.EDIPartnerDocumentProfile{
				TransactionSet:    tt.transactionSet,
				FunctionalGroupID: edi.FunctionalGroupDefault(tt.transactionSet),
				Envelope:          edi.DefaultX12EnvelopeSettings(),
				ValidationMode:    edi.ValidationModeStrict,
				PartnerSettings:   map[string]any{},
			}
			runtime := RuntimeValues(profile, edi.DefaultX12204Version)
			SetProvisionalControlNumbers(runtime)

			result, err := RenderX12(&RenderInput{
				Profile: profile,
				TemplateVersion: &edi.EDITemplateVersion{
					Segments: []*edi.EDITemplateSegment{
						{
							SegmentID: "TST",
							Sequence:  1,
							Required:  true,
							Elements: []edi.TemplateElement{
								{
									Position:         1,
									Name:             "Value",
									Source:           edi.TemplateElementSourceStarlark,
									StarlarkFunction: "value",
									StarlarkScript:   tt.script,
								},
							},
						},
					},
				},
				DocumentPayload: tt.payload,
				Runtime:         runtime,
			})

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			assert.Contains(t, result.RawX12, tt.want)
		})
	}
}

func TestRender204_StarlarkRuntimeDiagnosticPreservesMetadata(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "explode"
	element.StarlarkScript = `def explode(ctx):
    return 1 / 0`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
	assert.Equal(t, "starlark_runtime_error", diagnostic.Code)
	assert.Equal(t, "L11", diagnostic.SegmentID)
	assert.Equal(t, 1, diagnostic.ElementPosition)
	assert.Equal(t, "starlark:explode", diagnostic.Path)
	assert.NotContains(t, diagnostic.Path, element.StarlarkScript)
	assert.Contains(t, diagnostic.Message, "floating-point division by zero")
	assert.Equal(
		t,
		"Check the Starlark script, function name, helper arguments, and available context fields.",
		diagnostic.SuggestedFix,
	)
	assert.False(t, strings.Contains(result.RawX12, "L11**BM*"))
}

func TestRender204_StarlarkEmptyScriptReportsMissingDefaultFunction(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, "script_function_not_found", diagnostic.Code)
	assert.Equal(t, "starlark:value", diagnostic.Path)
	assert.Contains(t, diagnostic.Message, "starlark function name is required")
	assert.Equal(
		t,
		"Define the referenced Starlark function in the inline script or template script libraries.",
		diagnostic.SuggestedFix,
	)
}

func TestRender204_StarlarkMissingCustomFunctionReportsConfiguredPath(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "map_bol"
	element.StarlarkScript = `def value(ctx):
    return "BOL-1"`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, "script_function_not_found", diagnostic.Code)
	assert.Equal(t, "starlark:map_bol", diagnostic.Path)
	assert.Contains(t, diagnostic.Message, `required Starlark function "map_bol" is not defined`)
	assert.Equal(
		t,
		"Define the referenced Starlark function in the inline script or template script libraries.",
		diagnostic.SuggestedFix,
	)
}

func TestRender204_StarlarkDefaultFunctionDiagnosticUsesSafePath(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkScript = `def value(ctx):
    return missing`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, "starlark_runtime_error", diagnostic.Code)
	assert.Equal(t, "starlark:value", diagnostic.Path)
	assert.NotContains(t, diagnostic.Path, element.StarlarkScript)
	assert.Equal(
		t,
		"Check the Starlark script, function name, helper arguments, and available context fields.",
		diagnostic.SuggestedFix,
	)
}

func TestRender204_DisabledValidationPreservesStarlarkDiagnostics(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeDisabled)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "explode"
	element.StarlarkScript = `def explode(ctx):
    return 1 / 0`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, "starlark_runtime_error", result.Diagnostics[0].Code)
	assert.Equal(t, "starlark:explode", result.Diagnostics[0].Path)
}

func TestRender204_StarlarkStepLimitDiagnosticPropagates(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "loop"
	element.StarlarkScript = `def loop(ctx):
    while True:
        pass`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	diagnostic := result.Diagnostics[0]
	assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
	assert.Equal(t, "starlark_step_limit", diagnostic.Code)
	assert.Equal(t, "L11", diagnostic.SegmentID)
	assert.Equal(t, 1, diagnostic.ElementPosition)
	assert.Equal(t, "starlark:loop", diagnostic.Path)
	assert.NotContains(t, diagnostic.Path, element.StarlarkScript)
	assert.NotEmpty(t, diagnostic.Message)
	assert.Equal(t, "Reduce loop work or simplify the Starlark script.", diagnostic.SuggestedFix)
}

func TestRender204_StarlarkElementUsesLibraryFunction(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = "BOL-1"
	input.TemplateVersion.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "refs",
			Language: edi.ScriptLanguageStarlark,
			Script: `def bol_ref(ctx):
    return "LIB-" + ctx["shipment"]["bol"]`,
		},
	}
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "bol_ref"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*LIB-BOL-1*BM")
}

func TestRender204_InlineStarlarkCallsLibraryHelper(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = "BOL-1"
	input.TemplateVersion.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "refs",
			Language: edi.ScriptLanguageStarlark,
			Script: `def prefix(value):
    return "HELP-" + value`,
		},
	}
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkScript = `def value(ctx):
    return prefix(ctx["shipment"]["bol"])`

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*HELP-BOL-1*BM")
}

func TestRender204_LibraryBackedCondition(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = "BOL-1"
	input.TemplateVersion.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "conditions",
			Language: edi.ScriptLanguageStarlark,
			Script: `def include_bol(ctx):
    return ctx["shipment"]["bol"] != ""`,
		},
	}
	findSegment(t, input, "L11").Condition = "starlark:include_bol"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*BOL-1*BM")
}

func TestRender204_RepeatLibraryConditionReceivesItem(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.Moves[0].Stops = []edi.LoadTenderStop{
		{Type: "LD", Sequence: 1, LocationName: "Load Dock"},
		{Type: "UL", Sequence: 2, LocationName: "Unload Dock"},
	}
	input.TemplateVersion.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "conditions",
			Language: edi.ScriptLanguageStarlark,
			Script: `def is_load(ctx, item):
    return item["type"] == "LD"`,
		},
	}
	findSegment(t, input, "N1").Condition = "starlark:is_load"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "N1*LD*Load Dock")
	assert.NotContains(t, result.RawX12, "N1*UL*Unload Dock")
}

func TestRender204_MissingLibraryFunctionDiagnostic(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "missing_ref"

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, "script_function_not_found", result.Diagnostics[0].Code)
	assert.Equal(t, "starlark:missing_ref", result.Diagnostics[0].Path)
}

func TestRender204_DuplicateLibraryFunctionsDiagnosticPropagates(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	element := findElement(t, input, "L11", 0)
	element.Source = edi.TemplateElementSourceStarlark
	element.FieldPath = ""
	element.StarlarkFunction = "ref"
	input.TemplateVersion.ScriptLibraries = []*edi.EDITemplateScriptLibrary{
		{
			Name:     "a",
			Language: edi.ScriptLanguageStarlark,
			Script:   "def ref(ctx):\n    return 'a'",
		},
		{
			Name:     "b",
			Language: edi.ScriptLanguageStarlark,
			Script:   "def ref(ctx):\n    return 'b'",
		},
	}

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 1)
	assert.Equal(t, "script_library_duplicate_function", result.Diagnostics[0].Code)
}

func TestHasBlockingDiagnostics_BlocksStrictStarlarkDiagnostics(t *testing.T) {
	t.Parallel()

	diagnostics := []Diagnostic{
		{
			Severity: edi.ValidationSeverityError,
			Code:     "starlark_runtime_error",
		},
	}

	assert.True(t, HasBlockingDiagnostics(diagnostics, edi.ValidationModeStrict))
	assert.False(t, HasBlockingDiagnostics(diagnostics, edi.ValidationModeWarnOnly))
}

func renderInput(mode edi.ValidationMode) *RenderInput {
	envelope := edi.DefaultX12EnvelopeSettings()
	profile := &edi.EDIPartnerDocumentProfile{
		Envelope:          envelope,
		FunctionalGroupID: "SM",
		ValidationMode:    mode,
		PartnerSettings:   map[string]any{},
	}
	version := &edi.EDITemplateVersion{
		X12Version:        edi.DefaultX12204Version,
		FunctionalGroupID: "SM",
		Segments: editemplates.Base204Segments(
			pagination.TenantInfo{},
			pulid.MustNew("editv_"),
		),
	}
	runtime := RuntimeValues(profile, edi.DefaultX12204Version)
	SetProvisionalControlNumbers(runtime)
	return &RenderInput{
		Profile:         profile,
		TemplateVersion: version,
		Payload: edi.LoadTenderPayload{
			Moves:       []edi.LoadTenderMove{},
			Commodities: []edi.LoadTenderCommodity{},
		},
		X12Version: edi.DefaultX12204Version,
		Runtime:    runtime,
	}
}

func validRenderInput(mode edi.ValidationMode) *RenderInput {
	input := renderInput(mode)
	input.Payload.ShipmentID = pulid.MustNew("shp_")
	input.Payload.Moves = []edi.LoadTenderMove{
		{
			Sequence: 1,
			Stops: []edi.LoadTenderStop{
				{
					Type:         "LD",
					Sequence:     1,
					LocationName: "Chicago Dock",
				},
			},
		},
	}
	return input
}

func findSegment(
	t *testing.T,
	input *RenderInput,
	segmentID string,
) *edi.EDITemplateSegment {
	t.Helper()

	for _, segment := range input.TemplateVersion.Segments {
		if segment.SegmentID == segmentID {
			return segment
		}
	}
	require.Failf(t, "segment not found", "segment %s not found", segmentID)
	return nil
}

func findElement(
	t *testing.T,
	input *RenderInput,
	segmentID string,
	index int,
) *edi.TemplateElement {
	t.Helper()

	segment := findSegment(t, input, segmentID)
	require.Less(t, index, len(segment.Elements))
	return &segment.Elements[index]
}
