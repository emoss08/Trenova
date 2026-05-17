package edix12

import (
	"testing"
	"time"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRender204_TransformPipelineRendersValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*testing.T, *RenderInput)
		wantRaw   string
	}{
		{
			name: "field path trim and upper",
			configure: func(t *testing.T, input *RenderInput) {
				input.Payload.BOL = " bol-low "
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source:    edi.TemplateElementSourceFieldPath,
						FieldPath: "bol",
					},
					edi.TemplateTransformStep{Operation: "trim"},
					edi.TemplateTransformStep{Operation: "upper"},
				)
			},
			wantRaw: "L11*BOL-LOW*BM",
		},
		{
			name: "constant left pad",
			configure: func(t *testing.T, input *RenderInput) {
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
						Value:  "7",
					},
					edi.TemplateTransformStep{
						Operation: "left_pad",
						Arguments: map[string]any{
							"length": 4,
							"pad":    "0",
						},
					},
				)
			},
			wantRaw: "L11*0007*BM",
		},
		{
			name: "repeat state normalize",
			configure: func(t *testing.T, input *RenderInput) {
				input.Payload.Moves[0].Stops[0].LocationStateCode = " il."
				setTransform(
					findElement(t, input, "N4", 1),
					edi.TemplateElementBaseSource{
						Source:     edi.TemplateElementSourceRepeat,
						RepeatPath: "locationStateCode",
					},
					edi.TemplateTransformStep{Operation: "normalize_state"},
				)
			},
			wantRaw: "N4**IL",
		},
		{
			name: "empty base source default",
			configure: func(t *testing.T, input *RenderInput) {
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
					},
					edi.TemplateTransformStep{
						Operation: "default",
						Arguments: map[string]any{"value": "FALLBACK"},
					},
				)
			},
			wantRaw: "L11*FALLBACK*BM",
		},
		{
			name: "empty runtime slice uses default",
			configure: func(t *testing.T, input *RenderInput) {
				input.Runtime["emptySlice"] = []any{}
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source:     edi.TemplateElementSourceRuntime,
						RuntimeKey: "emptySlice",
					},
					edi.TemplateTransformStep{
						Operation: "default",
						Arguments: map[string]any{"value": "FALLBACK"},
					},
				)
			},
			wantRaw: "L11*FALLBACK*BM",
		},
		{
			name: "empty runtime map uses default",
			configure: func(t *testing.T, input *RenderInput) {
				input.Runtime["emptyMap"] = map[string]any{}
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source:     edi.TemplateElementSourceRuntime,
						RuntimeKey: "emptyMap",
					},
					edi.TemplateTransformStep{
						Operation: "default",
						Arguments: map[string]any{"value": "FALLBACK"},
					},
				)
			},
			wantRaw: "L11*FALLBACK*BM",
		},
		{
			name: "false runtime bool uses default",
			configure: func(t *testing.T, input *RenderInput) {
				input.Runtime["emptyFlag"] = false
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source:     edi.TemplateElementSourceRuntime,
						RuntimeKey: "emptyFlag",
					},
					edi.TemplateTransformStep{
						Operation: "default",
						Arguments: map[string]any{"value": "FALLBACK"},
					},
				)
			},
			wantRaw: "L11*FALLBACK*BM",
		},
		{
			name: "coalesce references",
			configure: func(t *testing.T, input *RenderInput) {
				input.Payload.BOL = ""
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
					},
					edi.TemplateTransformStep{
						Operation: "coalesce",
						Arguments: map[string]any{
							"values": []any{"$shipment.bol", "$shipment.shipmentId"},
						},
					},
				)
			},
			wantRaw: "L11*shp_",
		},
		{
			name: "coalesce skips empty runtime collections",
			configure: func(t *testing.T, input *RenderInput) {
				input.Runtime["emptySlice"] = []string{}
				input.Runtime["emptyMap"] = map[string]string{}
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
					},
					edi.TemplateTransformStep{
						Operation: "coalesce",
						Arguments: map[string]any{
							"values": []any{
								"$runtime.emptySlice",
								"$runtime.emptyMap",
								"NEXT",
							},
						},
					},
				)
			},
			wantRaw: "L11*NEXT*BM",
		},
		{
			name: "unix timestamp format date",
			configure: func(t *testing.T, input *RenderInput) {
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
						Value:  "1778941800",
					},
					edi.TemplateTransformStep{Operation: "format_date"},
				)
			},
			wantRaw: "L11*20260516*BM",
		},
		{
			name: "decimal format",
			configure: func(t *testing.T, input *RenderInput) {
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
						Value:  "12.3",
					},
					edi.TemplateTransformStep{
						Operation: "format_decimal",
						Arguments: map[string]any{"places": 2},
					},
				)
			},
			wantRaw: "L11*12.30*BM",
		},
		{
			name: "qualifier mapping from repeat",
			configure: func(t *testing.T, input *RenderInput) {
				setTransform(
					findElement(t, input, "N1", 0),
					edi.TemplateElementBaseSource{
						Source:     edi.TemplateElementSourceRepeat,
						RepeatPath: "type",
					},
					edi.TemplateTransformStep{
						Operation: "qualifier",
						Arguments: map[string]any{
							"mapping": map[string]any{
								"LD": "SH",
							},
						},
					},
				)
			},
			wantRaw: "N1*SH*Chicago Dock",
		},
		{
			name: "conditional reference",
			configure: func(t *testing.T, input *RenderInput) {
				input.Payload.BOL = "BOL-COND"
				setTransform(
					findElement(t, input, "L11", 0),
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
					},
					edi.TemplateTransformStep{
						Operation: "conditional",
						Arguments: map[string]any{
							"when": "$shipment.bol",
							"then": "HAS",
							"else": "NONE",
						},
					},
				)
			},
			wantRaw: "L11*HAS*BM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			tt.configure(t, input)

			result, err := Render204(input)

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			assert.Contains(t, result.RawX12, tt.wantRaw)
		})
	}
}

func TestRender204_TransformConditionalUsesRuntimeTruthiness(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value any
	}{
		{
			name:  "false bool",
			value: false,
		},
		{
			name:  "empty array",
			value: [0]string{},
		},
		{
			name:  "empty slice",
			value: []any{},
		},
		{
			name:  "empty map",
			value: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			input.Runtime["condition"] = tt.value
			setTransform(
				findElement(t, input, "L11", 0),
				edi.TemplateElementBaseSource{Source: edi.TemplateElementSourceConstant},
				edi.TemplateTransformStep{
					Operation: "conditional",
					Arguments: map[string]any{
						"when": "$runtime.condition",
						"then": "HAS",
						"else": "NONE",
					},
				},
			)

			result, err := Render204(input)

			require.NoError(t, err)
			require.Empty(t, result.Diagnostics)
			assert.Contains(t, result.RawX12, "L11*NONE*BM")
		})
	}
}

func TestRender204_TransformPipelineReportsConfigurationErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		configure func(*edi.TemplateElement)
		message   string
	}{
		{
			name: "unknown operation",
			configure: func(element *edi.TemplateElement) {
				setTransform(
					element,
					edi.TemplateElementBaseSource{
						Source: edi.TemplateElementSourceConstant,
						Value:  "value",
					},
					edi.TemplateTransformStep{Operation: "explode"},
				)
			},
			message: "unsupported transform operation",
		},
		{
			name: "nil base source",
			configure: func(element *edi.TemplateElement) {
				element.Source = edi.TemplateElementSourceTransform
				element.BaseSource = nil
				element.TransformPipeline = []edi.TemplateTransformStep{{Operation: "trim"}}
			},
			message: "base source is required",
		},
		{
			name: "recursive transform base source",
			configure: func(element *edi.TemplateElement) {
				setTransform(
					element,
					edi.TemplateElementBaseSource{Source: edi.TemplateElementSourceTransform},
					edi.TemplateTransformStep{Operation: "trim"},
				)
			},
			message: "cannot be another transform",
		},
		{
			name: "starlark base source",
			configure: func(element *edi.TemplateElement) {
				setTransform(
					element,
					edi.TemplateElementBaseSource{Source: edi.TemplateElementSourceStarlark},
					edi.TemplateTransformStep{Operation: "trim"},
				)
			},
			message: "cannot be starlark",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			element := findElement(t, input, "L11", 0)
			element.FieldPath = ""
			tt.configure(element)

			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, 1)
			diagnostic := result.Diagnostics[0]
			assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
			assert.Equal(t, "transform_error", diagnostic.Code)
			assert.Equal(t, "L11", diagnostic.SegmentID)
			assert.Equal(t, 1, diagnostic.ElementPosition)
			assert.Contains(t, diagnostic.Message, tt.message)
			assert.Equal(t, transformSuggestedFix, diagnostic.SuggestedFix)
		})
	}
}

func TestRender204_TransformRequiredReportsEmptyRuntimeCollections(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		runtimeKey string
		value      any
	}{
		{
			name:       "empty slice",
			runtimeKey: "emptySlice",
			value:      []any{},
		},
		{
			name:       "empty map",
			runtimeKey: "emptyMap",
			value:      map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			input.Runtime[tt.runtimeKey] = tt.value
			setTransform(
				findElement(t, input, "L11", 0),
				edi.TemplateElementBaseSource{
					Source:     edi.TemplateElementSourceRuntime,
					RuntimeKey: tt.runtimeKey,
				},
				edi.TemplateTransformStep{
					Operation: "required",
					Arguments: map[string]any{"message": "runtime collection is required"},
				},
			)

			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, 1)
			diagnostic := result.Diagnostics[0]
			assert.Equal(t, edi.ValidationSeverityError, diagnostic.Severity)
			assert.Equal(t, "transform_error", diagnostic.Code)
			assert.Equal(t, "L11", diagnostic.SegmentID)
			assert.Equal(t, 1, diagnostic.ElementPosition)
			assert.Contains(t, diagnostic.Message, "runtime collection is required")
			assert.Equal(t, transformSuggestedFix, diagnostic.SuggestedFix)
		})
	}
}

func TestRender204_TransformDiagnosticsRespectValidationMode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		mode     edi.ValidationMode
		severity edi.ValidationSeverity
	}{
		{
			name:     "disabled preserves transform diagnostics",
			mode:     edi.ValidationModeDisabled,
			severity: edi.ValidationSeverityError,
		},
		{
			name:     "warn only downgrades transform diagnostics",
			mode:     edi.ValidationModeWarnOnly,
			severity: edi.ValidationSeverityWarning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(tt.mode)
			setTransform(
				findElement(t, input, "L11", 0),
				edi.TemplateElementBaseSource{
					Source: edi.TemplateElementSourceConstant,
					Value:  "value",
				},
				edi.TemplateTransformStep{Operation: "unknown"},
			)

			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, 1)
			assert.Equal(t, "transform_error", result.Diagnostics[0].Code)
			assert.Equal(t, tt.severity, result.Diagnostics[0].Severity)
		})
	}
}

func TestHasBlockingDiagnostics_BlocksStrictTransformDiagnostics(t *testing.T) {
	t.Parallel()

	diagnostics := []Diagnostic{
		{
			Severity: edi.ValidationSeverityError,
			Code:     "transform_error",
		},
	}

	assert.True(t, HasBlockingDiagnostics(diagnostics, edi.ValidationModeStrict))
	assert.False(t, HasBlockingDiagnostics(diagnostics, edi.ValidationModeWarnOnly))
}

func TestRender204_TransformOutputUsesElementPostProcessing(t *testing.T) {
	t.Parallel()

	t.Run("required validation applies after empty transform", func(t *testing.T) {
		t.Parallel()

		input := validRenderInput(edi.ValidationModeStrict)
		element := findElement(t, input, "B2", 1)
		setTransform(
			element,
			edi.TemplateElementBaseSource{Source: edi.TemplateElementSourceConstant},
			edi.TemplateTransformStep{Operation: "empty_if_none"},
		)

		result, err := Render204(input)

		require.NoError(t, err)
		require.Len(t, result.Diagnostics, 1)
		assert.Equal(t, "required", result.Diagnostics[0].Code)
		assert.Contains(t, result.Diagnostics[0].Message, "Shipment Identification Number is required")
	})

	t.Run("max length validation applies and truncates after transform", func(t *testing.T) {
		t.Parallel()

		input := validRenderInput(edi.ValidationModeStrict)
		element := findElement(t, input, "L11", 0)
		element.Validation.MaxLength = 3
		setTransform(
			element,
			edi.TemplateElementBaseSource{
				Source: edi.TemplateElementSourceConstant,
				Value:  "abcdef",
			},
			edi.TemplateTransformStep{Operation: "upper"},
		)

		result, err := Render204(input)

		require.NoError(t, err)
		require.Len(t, result.Diagnostics, 1)
		assert.Equal(t, "max_length", result.Diagnostics[0].Code)
		assert.Contains(t, result.RawX12, "L11*ABC*BM")
	})

	t.Run("x12 separators are sanitized after transform", func(t *testing.T) {
		t.Parallel()

		input := validRenderInput(edi.ValidationModeStrict)
		setTransform(
			findElement(t, input, "L11", 0),
			edi.TemplateElementBaseSource{
				Source: edi.TemplateElementSourceConstant,
				Value:  "A*B~C>D",
			},
			edi.TemplateTransformStep{Operation: "trim"},
		)

		result, err := Render204(input)

		require.NoError(t, err)
		require.Empty(t, result.Diagnostics)
		assert.Contains(t, result.RawX12, "L11*A B C D*BM~")
	})
}

func TestRender204_TransformInvalidInputReportsError(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		operation string
		value     string
		args      map[string]any
		message   string
	}{
		{
			name:      "invalid date",
			operation: "format_date",
			value:     "not-a-date",
			message:   "not a valid time",
		},
		{
			name:      "invalid decimal",
			operation: "format_decimal",
			value:     "not-a-decimal",
			args:      map[string]any{"places": 2},
			message:   "not a valid decimal",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			input := validRenderInput(edi.ValidationModeStrict)
			setTransform(
				findElement(t, input, "L11", 0),
				edi.TemplateElementBaseSource{
					Source: edi.TemplateElementSourceConstant,
					Value:  tt.value,
				},
				edi.TemplateTransformStep{
					Operation: tt.operation,
					Arguments: tt.args,
				},
			)

			result, err := Render204(input)

			require.NoError(t, err)
			require.Len(t, result.Diagnostics, 1)
			assert.Equal(t, "transform_error", result.Diagnostics[0].Code)
			assert.Contains(t, result.Diagnostics[0].Message, tt.message)
		})
	}
}

func TestRender204_TransformFormatTimeReturnsEmptyForEmptyInput(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	setTransform(
		findElement(t, input, "L11", 0),
		edi.TemplateElementBaseSource{Source: edi.TemplateElementSourceConstant},
		edi.TemplateTransformStep{Operation: "format_date"},
	)

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11**BM")
}

func TestRender204_TransformFormatTime(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	setTransform(
		findElement(t, input, "L11", 0),
		edi.TemplateElementBaseSource{
			Source: edi.TemplateElementSourceConstant,
			Value:  time.Date(2026, 5, 16, 14, 30, 0, 0, time.UTC).Format(time.RFC3339),
		},
		edi.TemplateTransformStep{Operation: "format_time"},
	)

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*1430*BM")
}

func setTransform(
	element *edi.TemplateElement,
	baseSource edi.TemplateElementBaseSource,
	steps ...edi.TemplateTransformStep,
) {
	element.Source = edi.TemplateElementSourceTransform
	element.BaseSource = &baseSource
	element.TransformPipeline = steps
}

func TestRender204_TransformUsesRuntimeAndPartnerArgumentReferences(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Profile.PartnerSettings = map[string]any{
		"carrier": map[string]any{"scac": "ABCD"},
	}
	input.Runtime["transactionControlNumber"] = "1234"
	setTransform(
		findElement(t, input, "L11", 0),
		edi.TemplateElementBaseSource{
			Source: edi.TemplateElementSourceConstant,
			Value:  "REF",
		},
		edi.TemplateTransformStep{
			Operation: "concat",
			Arguments: map[string]any{
				"separator": "-",
				"values": []any{
					"$partner.carrier.scac",
					"$runtime.transactionControlNumber",
				},
			},
		},
	)

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*REF-ABCD-1234*BM")
}

func TestRender204_TransformContainsRendersBoolAsX12String(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = "BOL-123"
	setTransform(
		findElement(t, input, "L11", 0),
		edi.TemplateElementBaseSource{
			Source:    edi.TemplateElementSourceFieldPath,
			FieldPath: "bol",
		},
		edi.TemplateTransformStep{
			Operation: "contains",
			Arguments: map[string]any{"value": "BOL"},
		},
	)

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*Y*BM")
}

func TestRender204_TransformCoalesceFallsBackToShipmentID(t *testing.T) {
	t.Parallel()

	input := validRenderInput(edi.ValidationModeStrict)
	input.Payload.BOL = ""
	shipmentID := pulid.MustNew("shp_")
	input.Payload.ShipmentID = shipmentID
	setTransform(
		findElement(t, input, "L11", 0),
		edi.TemplateElementBaseSource{Source: edi.TemplateElementSourceConstant},
		edi.TemplateTransformStep{
			Operation: "coalesce",
			Arguments: map[string]any{
				"values": []any{"$shipment.bol", "$shipment.shipmentId"},
			},
		},
	)

	result, err := Render204(input)

	require.NoError(t, err)
	require.Empty(t, result.Diagnostics)
	assert.Contains(t, result.RawX12, "L11*"+shipmentID.String()+"*BM")
}
