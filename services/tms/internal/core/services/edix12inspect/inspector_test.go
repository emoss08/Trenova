package edix12inspect

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetectSeparators(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		rawX12      string
		envelope    *edi.X12EnvelopeSettings
		wantSource  SeparatorSource
		wantElement string
		wantSegment string
		wantCode    string
	}{
		{
			name:        "detects fixed width ISA separators",
			rawX12:      valid204("*", "~", ">", "^"),
			wantSource:  SeparatorSourceISA,
			wantElement: "*",
			wantSegment: "~",
		},
		{
			name:   "uses envelope when ISA is malformed",
			rawX12: "ST|204|0001!",
			envelope: &edi.X12EnvelopeSettings{
				ElementSeparator:    "|",
				SegmentTerminator:   "!",
				ComponentSeparator:  ":",
				RepetitionSeparator: "^",
			},
			wantSource:  SeparatorSourceEnvelope,
			wantElement: "|",
			wantSegment: "!",
			wantCode:    "x12.separator.isa_missing",
		},
		{
			name:        "falls back without ISA or envelope",
			rawX12:      "ST*204*0001~",
			wantSource:  SeparatorSourceFallback,
			wantElement: "*",
			wantSegment: "~",
			wantCode:    "x12.separator.fallback",
		},
		{
			name:   "warns when envelope conflicts with ISA",
			rawX12: valid204("*", "~", ">", "^"),
			envelope: &edi.X12EnvelopeSettings{
				ElementSeparator:    "|",
				SegmentTerminator:   "!",
				ComponentSeparator:  ":",
				RepetitionSeparator: "^",
			},
			wantSource:  SeparatorSourceISA,
			wantElement: "*",
			wantSegment: "~",
			wantCode:    "x12.separator.conflict",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			separators, diagnostics := DetectSeparators(tt.rawX12, tt.envelope)

			assert.Equal(t, tt.wantSource, separators.Source)
			assert.Equal(t, tt.wantElement, separators.Element)
			assert.Equal(t, tt.wantSegment, separators.Segment)
			if tt.wantCode == "" {
				assert.Empty(t, diagnostics)
				return
			}
			require.NotEmpty(t, diagnostics)
			assert.Equal(t, tt.wantCode, diagnostics[0].Code)
		})
	}
}

func TestInspectX12_ParsesSegmentsAndPreservesValues(t *testing.T) {
	t.Parallel()

	rawX12 := valid204("*", "~", ">", "^")
	result := InspectX12(&InspectX12Request{
		RawX12:         rawX12,
		TransactionSet: edi.TransactionSet204,
		X12Version:     edi.DefaultX12204Version,
	})

	require.Len(t, result.Segments, 7)
	assert.Equal(t, rawX12, result.RawX12)
	assert.Equal(t, "000000905", result.Envelope.ISAControlNumber)
	assert.Equal(t, "0001", result.Transactions[0].STControlNumber)
	assert.Equal(t, "0001", result.Transactions[0].SEControlNumber)
	assert.Equal(t, 4, result.Segments[3].Index)
	assert.Equal(t, "B2", result.Segments[3].SegmentID)
	assert.Equal(t, "", result.Segments[3].Elements[0].Value)
	assert.True(t, result.Segments[3].Elements[0].Empty)
	assert.Equal(t, "Standard Point Location Code", result.Segments[3].Elements[2].Label)
	assert.Contains(t, result.Formatted, "B2 - Beginning Segment for Shipment Information")
	assert.Empty(t, result.Diagnostics)
}

func TestInspectX12_ParsesCompositesAndMissingTerminator(t *testing.T) {
	t.Parallel()

	rawX12 := valid204("*", "~", ">", "^") + "L11*AA>BB*BM"
	result := InspectX12(&InspectX12Request{RawX12: rawX12})

	last := result.Segments[len(result.Segments)-1]
	require.Len(t, last.Elements[0].Components, 2)
	assert.Equal(t, "AA", last.Elements[0].Components[0].Value)
	assert.Equal(t, "BB", last.Elements[0].Components[1].Value)
	assertDiagnostic(t, result.Diagnostics, "x12.segment.missing_terminator")
}

func TestInspectX12_ValidatesControls(t *testing.T) {
	t.Parallel()

	rawX12 := isa("*", "~", ">", "^", "000000905") +
		"GS*SM*SENDER*RECEIVER*20260518*1200*77*X*004010~" +
		"ST*204*0001~" +
		"B2**SHIP123**PP~" +
		"SE*99*9999~" +
		"GE*2*88~" +
		"IEA*2*000000999~"

	result := InspectX12(&InspectX12Request{
		RawX12:         rawX12,
		TransactionSet: edi.TransactionSet210,
	})

	assertDiagnostic(t, result.Diagnostics, "x12.transaction_set.unsupported")
	assertDiagnostic(t, result.Diagnostics, "x12.control.st_se_mismatch")
	assertDiagnostic(t, result.Diagnostics, "x12.count.se01_mismatch")
	assertDiagnostic(t, result.Diagnostics, "x12.control.gs_ge_mismatch")
	assertDiagnostic(t, result.Diagnostics, "x12.count.ge01_mismatch")
	assertDiagnostic(t, result.Diagnostics, "x12.control.isa_iea_mismatch")
	assertDiagnostic(t, result.Diagnostics, "x12.count.iea01_mismatch")
	assert.Equal(t, 6, result.Summary.ErrorCount)
	assert.Equal(t, 1, result.Summary.WarningCount)
}

func TestInspectX12_MergesRenderDiagnostics(t *testing.T) {
	t.Parallel()

	result := InspectX12(&InspectX12Request{
		RawX12: valid204("*", "~", ">", "^"),
		Diagnostics: []edix12.Diagnostic{
			{
				Severity:        edi.ValidationSeverityError,
				Code:            "required",
				SegmentID:       "B2",
				ElementPosition: 4,
				Path:            "payload.shipment.id",
				Message:         "Shipment ID is required.",
				SuggestedFix:    "Provide shipment ID.",
			},
		},
	})

	var found NormalizedDiagnostic
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "required" {
			found = diagnostic
			break
		}
	}
	require.NotZero(t, found)
	assert.Equal(t, DiagnosticSourceRender, found.Source)
	assert.Equal(t, 4, found.SegmentIndex)
	assert.Equal(t, 4, found.ElementPosition)
}

func valid204(
	elementSeparator string,
	segmentTerminator string,
	componentSeparator string,
	repetitionSeparator string,
) string {
	return isa(
		elementSeparator,
		segmentTerminator,
		componentSeparator,
		repetitionSeparator,
		"000000905",
	) +
		joinSegments(
			elementSeparator,
			segmentTerminator,
			"GS",
			"SM",
			"SENDER",
			"RECEIVER",
			"20260518",
			"1200",
			"77",
			"X",
			"004010",
		) +
		joinSegments(
			elementSeparator,
			segmentTerminator,
			"ST",
			"204",
			"0001",
		) +
		joinSegments(
			elementSeparator,
			segmentTerminator,
			"B2",
			"",
			"SHIP123",
			"",
			"PP",
		) +
		joinSegments(
			elementSeparator,
			segmentTerminator,
			"SE",
			"3",
			"0001",
		) +
		joinSegments(
			elementSeparator,
			segmentTerminator,
			"GE",
			"1",
			"77",
		) +
		joinSegments(
			elementSeparator,
			segmentTerminator,
			"IEA",
			"1",
			"000000905",
		)
}

func isa(
	elementSeparator string,
	segmentTerminator string,
	componentSeparator string,
	repetitionSeparator string,
	controlNumber string,
) string {
	segment := strings.Join([]string{
		"ISA",
		"00",
		fmt.Sprintf("%-10s", ""),
		"00",
		fmt.Sprintf("%-10s", ""),
		"ZZ",
		fmt.Sprintf("%-15s", "SENDER"),
		"ZZ",
		fmt.Sprintf("%-15s", "RECEIVER"),
		"260518",
		"1200",
		repetitionSeparator,
		"00401",
		controlNumber,
		"0",
		"T",
		componentSeparator,
	}, elementSeparator)
	if len(segment) != isaSegmentLength {
		panic(fmt.Sprintf("invalid test ISA length %d", len(segment)))
	}
	return segment + segmentTerminator
}

func joinSegments(elementSeparator string, segmentTerminator string, values ...string) string {
	return strings.Join(values, elementSeparator) + segmentTerminator
}

func assertDiagnostic(t *testing.T, diagnostics []NormalizedDiagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected diagnostic %q in %#v", code, diagnostics)
}
