package edix12inspect

import (
	"fmt"
	"strings"
	"testing"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const isaLength = 106

func inspect(t *testing.T, req *InspectX12Request) *InspectX12Result {
	t.Helper()
	result, err := InspectX12(t.Context(), req)
	require.NoError(t, err)
	return result
}

func TestInspectX12_ReadsSeparatorsFromTheISAAndWarnsWhenTheProfileDisagrees(t *testing.T) {
	t.Parallel()

	result := inspect(t, &InspectX12Request{RawX12: valid204("*", "~", ">", "^")})
	assert.Equal(t, SeparatorSourceISA, result.Separators.Source)
	assert.Equal(t, "*", result.Separators.Element)
	assert.Equal(t, "~", result.Separators.Segment)
	assert.False(t, result.Separators.HasConflict)

	conflicting := inspect(t, &InspectX12Request{
		RawX12: valid204("*", "~", ">", "^"),
		Envelope: &edidocument.X12EnvelopeSettings{
			ElementSeparator:   "|",
			SegmentTerminator:  "!",
			ComponentSeparator: ":",
		},
	})
	assert.True(t, conflicting.Separators.HasConflict)
	assertDiagnostic(t, conflicting.Diagnostics, "x12.separator.conflict")
}

func TestInspectX12_ReadsAFragmentWithTheProfilesSeparators(t *testing.T) {
	t.Parallel()

	result := inspect(t, &InspectX12Request{
		RawX12: "ST|990|0001!B1|SIML|SHIP-1|20260107|A!SE|3|0001!",
		Envelope: &edidocument.X12EnvelopeSettings{
			ElementSeparator:  "|",
			SegmentTerminator: "!",
		},
	})
	assert.Equal(t, SeparatorSourceEnvelope, result.Separators.Source)
	require.Len(t, result.Segments, 3)
	assert.Equal(t, "Beginning Segment for Booking or Pick-up/Delivery", result.Segments[1].Name)
	assert.Equal(t, edidocument.TransactionSet("990"), result.TransactionSet)
	assert.Empty(t, result.Diagnostics)
}

func TestInspectX12_NamesSegmentsAndElementsFromTheGuideAndPlacesLoops(t *testing.T) {
	t.Parallel()

	raw := valid204("*", "~", ">", "^")
	result := inspect(t, &InspectX12Request{RawX12: raw})

	assert.Empty(t, result.Diagnostics)
	assert.Equal(t, raw, result.RawX12)
	assert.Equal(t, "000000905", result.Envelope.ISAControlNumber)
	assert.Equal(t, "0001", result.Transactions[0].STControlNumber)
	assert.Equal(t, "0001", result.Transactions[0].SEControlNumber)

	b2 := result.Segments[3]
	assert.Equal(t, 4, b2.Index)
	assert.Equal(t, "B2", b2.SegmentID)
	assert.Equal(t, "Beginning Segment for Shipment Information Transaction", b2.Name)
	assert.True(t, b2.Elements[0].Empty)
	assert.Equal(t, "Standard Point Location Code", b2.Elements[2].Label)
	assert.Equal(t, raw[b2.Elements[1].StartOffset:b2.Elements[1].EndOffset], "SIML")
	assert.Equal(t, "B2**SIML**SHIP123**PP~", b2.RawWithTerminator)

	n1 := result.Segments[7]
	assert.Equal(t, "N1", n1.SegmentID)
	assert.Equal(t, "0300/0310", n1.Loop)
	assert.True(t, n1.Elements[0].Required)
	assert.Contains(t, result.Formatted, "        N1 - Name")
}

func TestInspectX12_ReportsGuideFindingsWithTheirPlace(t *testing.T) {
	t.Parallel()

	raw := strings.Replace(valid204("*", "~", ">", "^"), "N1*SH*", "N1**", 1)
	result := inspect(t, &InspectX12Request{RawX12: raw})

	var missing NormalizedDiagnostic
	for _, diagnostic := range result.Diagnostics {
		if diagnostic.Code == "x12.mandatoryElementMissing" {
			missing = diagnostic
		}
	}
	require.NotZero(t, missing)
	assert.Equal(t, edidocument.ValidationSeverityError, missing.Severity)
	assert.Equal(t, DiagnosticSourceValidation, missing.Source)
	assert.Equal(t, "N1", missing.SegmentID)
	assert.Equal(t, 8, missing.SegmentIndex)
	assert.Equal(t, 1, missing.ElementPosition)
	assert.Equal(t, "loop 0310 · segment 6 · N101", missing.Path)
	assert.Equal(t, "N101 (Entity Identifier Code) is mandatory but is empty", missing.Message)
	assert.Equal(t, 1, result.Summary.ErrorCount)
}

func TestInspectX12_ChecksControlNumbersAndCounts(t *testing.T) {
	t.Parallel()

	raw := strings.NewReplacer(
		"SE*7*0001", "SE*99*9999",
		"GE*1*77", "GE*2*88",
		"IEA*1*000000905", "IEA*2*000000999",
	).Replace(valid204("*", "~", ">", "^"))
	result := inspect(t, &InspectX12Request{RawX12: raw})

	for _, code := range []string{
		"x12.segmentCountMismatch",
		"x12.transactionControlNumberMismatch",
		"x12.transactionCountMismatch",
		"x12.groupControlNumberMismatch",
		"x12.groupCountMismatch",
		"x12.interchangeControlNumberMismatch",
	} {
		assertDiagnostic(t, result.Diagnostics, code)
	}
	assert.Equal(t, 6, result.Summary.ErrorCount)
}

func TestInspectX12_MergesRenderDiagnostics(t *testing.T) {
	t.Parallel()

	result := inspect(t, &InspectX12Request{
		RawX12: valid204("*", "~", ">", "^"),
		Diagnostics: []edidocument.Diagnostic{
			{
				Severity:        edidocument.ValidationSeverityError,
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

func TestInspectX12_AnEmptyPayloadIsAnError(t *testing.T) {
	t.Parallel()

	result := inspect(t, &InspectX12Request{RawX12: "  \n"})
	assertDiagnostic(t, result.Diagnostics, "x12.document.empty")
	assert.Equal(t, 1, result.Summary.ErrorCount)
}

func valid204(
	elementSeparator string,
	segmentTerminator string,
	componentSeparator string,
	repetitionSeparator string,
) string {
	segments := [][]string{
		{"GS", "SM", "SENDER", "RECEIVER", "20260518", "1200", "77", "X", "004010"},
		{"ST", "204", "0001"},
		{"B2", "", "SIML", "", "SHIP123", "", "PP"},
		{"B2A", "00"},
		{"S5", "1", "LD"},
		{"G62", "10", "20260518"},
		{"N1", "SH", "Shipper"},
		{"SE", "7", "0001"},
		{"GE", "1", "77"},
		{"IEA", "1", "000000905"},
	}
	var builder strings.Builder
	builder.WriteString(isa(
		elementSeparator,
		segmentTerminator,
		componentSeparator,
		repetitionSeparator,
		"000000905",
	))
	for _, segment := range segments {
		builder.WriteString(strings.Join(segment, elementSeparator) + segmentTerminator)
	}
	return builder.String()
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
	}, elementSeparator) + segmentTerminator
	if len(segment) != isaLength {
		panic(fmt.Sprintf("invalid test ISA length %d", len(segment)))
	}
	return segment
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
