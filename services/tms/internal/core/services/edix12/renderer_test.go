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
