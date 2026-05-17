package edix12

import (
	"strings"
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	editemplates "github.com/emoss08/trenova/internal/core/domain/edi/templates"
	"github.com/emoss08/trenova/pkg/pagination"
	"github.com/emoss08/trenova/shared/pulid"
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

func TestRender204_ReportsUnsupportedAdvancedRenderingFeatures(t *testing.T) {
	t.Parallel()

	input := renderInput(edi.ValidationModeStrict)
	input.Payload.ShipmentID = pulid.MustNew("shp_")
	input.Payload.Moves = []edi.LoadTenderMove{
		{
			Sequence: 1,
			Stops: []edi.LoadTenderStop{
				{Type: "LD", Sequence: 1},
			},
		},
	}
	for _, segment := range input.TemplateVersion.Segments {
		switch segment.SegmentID {
		case "B2":
			segment.Elements[2].Source = edi.TemplateElementSourceTransform
			segment.Elements[2].BaseSource = &edi.TemplateElementBaseSource{
				Source:    edi.TemplateElementSourceFieldPath,
				FieldPath: "ratingDetail.paymentMethod",
			}
			segment.Elements[2].TransformPipeline = []edi.TemplateTransformStep{
				{
					Operation: "uppercase",
					Arguments: map[string]any{},
				},
			}
		case "NTE":
			segment.Condition = "shipment.ratingDetail.note != ''"
		}
	}

	result, err := Render204(input)

	require.NoError(t, err)
	require.Len(t, result.Diagnostics, 2)
	assert.Equal(t, "render_error", result.Diagnostics[0].Code)
	assert.NotEmpty(t, result.Diagnostics[0].SuggestedFix)
	assert.Contains(t, result.Diagnostics[0].Message, "not supported")
	assert.Equal(t, "render_error", result.Diagnostics[1].Code)
	assert.NotEmpty(t, result.Diagnostics[1].SuggestedFix)
	assert.Contains(t, result.Diagnostics[1].Message, "not supported")
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
	assert.Contains(t, result.RawX12, "B2**"+input.Payload.ShipmentID.String()+"*BOL-STARK")
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
	assert.Equal(t, "starlark_runtime_error", diagnostic.Code)
	assert.Equal(t, "starlark:value", diagnostic.Path)
	assert.Contains(t, diagnostic.Message, `required Starlark function "value" is not defined`)
	assert.Equal(
		t,
		"Check the Starlark script, function name, helper arguments, and available context fields.",
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
	assert.Equal(t, "starlark_runtime_error", diagnostic.Code)
	assert.Equal(t, "starlark:map_bol", diagnostic.Path)
	assert.Contains(t, diagnostic.Message, `required Starlark function "map_bol" is not defined`)
	assert.Equal(
		t,
		"Check the Starlark script, function name, helper arguments, and available context fields.",
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
