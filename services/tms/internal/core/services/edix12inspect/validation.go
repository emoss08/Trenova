package edix12inspect

import (
	"fmt"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
)

var requiredEnvelopeOrder = []string{"ISA", "GS", "ST", "SE", "GE", "IEA"}

func validateStructure(req *InspectX12Request, segments []X12Segment) []NormalizedDiagnostic {
	diagnostics := make([]NormalizedDiagnostic, 0)
	if len(segments) == 0 {
		return diagnostics
	}
	diagnostics = append(diagnostics, validateRequiredEnvelope(segments)...)
	diagnostics = append(diagnostics, validateEnvelopeOrder(segments)...)
	diagnostics = append(diagnostics, validateISA(segments)...)
	diagnostics = append(diagnostics, validateControls(req, segments)...)
	return diagnostics
}

func validateRequiredEnvelope(segments []X12Segment) []NormalizedDiagnostic {
	diagnostics := []NormalizedDiagnostic{}
	for _, segmentID := range requiredEnvelopeOrder {
		if firstSegment(segments, segmentID) != nil {
			continue
		}
		diagnostics = append(diagnostics, NormalizedDiagnostic{
			Severity:     edi.ValidationSeverityError,
			Code:         "x12.envelope.missing_" + strings.ToLower(segmentID),
			Source:       DiagnosticSourceInspection,
			SegmentID:    segmentID,
			Message:      fmt.Sprintf("Required envelope segment %s is missing.", segmentID),
			SuggestedFix: "Include a complete ISA/GS/ST/SE/GE/IEA envelope.",
		})
	}
	return diagnostics
}

func validateEnvelopeOrder(segments []X12Segment) []NormalizedDiagnostic {
	diagnostics := []NormalizedDiagnostic{}
	lastOrder := -1
	for i := range segments {
		segment := &segments[i]
		order := slices.Index(requiredEnvelopeOrder, segment.SegmentID)
		if order < 0 {
			continue
		}
		if order < lastOrder {
			diagnostics = append(diagnostics, NormalizedDiagnostic{
				Severity:     edi.ValidationSeverityError,
				Code:         "x12.envelope.order",
				Source:       DiagnosticSourceInspection,
				SegmentID:    segment.SegmentID,
				SegmentIndex: segment.Index,
				Message: fmt.Sprintf(
					"Envelope segment %s appears out of order.",
					segment.SegmentID,
				),
				SuggestedFix: "Order envelope controls as ISA, GS, ST, SE, GE, IEA.",
			})
		}
		lastOrder = max(lastOrder, order)
	}
	return diagnostics
}

func validateISA(segments []X12Segment) []NormalizedDiagnostic {
	isa := firstSegment(segments, "ISA")
	if isa == nil {
		return nil
	}
	diagnostics := []NormalizedDiagnostic{}
	if len(isa.Elements) != 16 {
		diagnostics = append(diagnostics, NormalizedDiagnostic{
			Severity:     edi.ValidationSeverityError,
			Code:         "x12.isa.element_count",
			Source:       DiagnosticSourceInspection,
			SegmentID:    "ISA",
			SegmentIndex: isa.Index,
			Message: fmt.Sprintf(
				"ISA contains %d elements; X12 ISA requires 16 fixed-position elements.",
				len(isa.Elements),
			),
			SuggestedFix: "Pad empty ISA positions and include ISA16 before the segment terminator.",
		})
	}
	if len(isa.Raw) != isaSegmentLength {
		diagnostics = append(diagnostics, NormalizedDiagnostic{
			Severity:     edi.ValidationSeverityWarning,
			Code:         "x12.isa.fixed_width",
			Source:       DiagnosticSourceInspection,
			SegmentID:    "ISA",
			SegmentIndex: isa.Index,
			Message: fmt.Sprintf(
				"ISA is %d characters before the terminator; fixed-width ISA is normally 105 characters.",
				len(isa.Raw),
			),
			SuggestedFix: "Verify ISA element padding and delimiter placement.",
		})
	}
	return diagnostics
}

func validateControls(req *InspectX12Request, segments []X12Segment) []NormalizedDiagnostic {
	diagnostics := []NormalizedDiagnostic{}
	diagnostics = append(diagnostics, validateTransactionControls(req, segments)...)
	diagnostics = append(diagnostics, validateGroupControls(segments)...)
	diagnostics = append(diagnostics, validateInterchangeControls(segments)...)
	return diagnostics
}

func validateTransactionControls(
	req *InspectX12Request,
	segments []X12Segment,
) []NormalizedDiagnostic {
	diagnostics := []NormalizedDiagnostic{}
	var openST *X12Segment
	for i := range segments {
		segment := &segments[i]
		switch segment.SegmentID {
		case "ST":
			openST = segment
			transactionSet := elementValue(segment, 1)
			hasRequestedMismatch := req.TransactionSet != "" &&
				transactionSet != "" &&
				string(req.TransactionSet) != transactionSet
			if hasRequestedMismatch {
				diagnostics = append(diagnostics, NormalizedDiagnostic{
					Severity:        edi.ValidationSeverityWarning,
					Code:            "x12.transaction_set.unsupported",
					Source:          DiagnosticSourceInspection,
					SegmentID:       "ST",
					SegmentIndex:    segment.Index,
					ElementPosition: 1,
					Message: fmt.Sprintf(
						"ST01 is %s but %s was requested.",
						transactionSet,
						req.TransactionSet,
					),
					SuggestedFix: "Inspect with the matching transaction set or correct ST01.",
				})
			}
		case "SE":
			if openST == nil {
				diagnostics = append(diagnostics, controlDiagnostic(
					segment,
					"x12.se.without_st",
					"SE appears before an open ST transaction.",
					"Move SE after the matching ST transaction.",
				))
				continue
			}
			stControl := elementValue(openST, 2)
			seControl := elementValue(segment, 2)
			if stControl != "" && seControl != "" && stControl != seControl {
				diagnostics = append(diagnostics, controlDiagnostic(
					segment,
					"x12.control.st_se_mismatch",
					fmt.Sprintf("ST02 %s does not match SE02 %s.", stControl, seControl),
					"Set SE02 to the same transaction control number as ST02.",
				))
			}
			expected := parsePositiveInt(elementValue(segment, 1))
			actual := segment.Index - openST.Index + 1
			if expected > 0 && expected != actual {
				diagnostics = append(diagnostics, controlDiagnostic(
					segment,
					"x12.count.se01_mismatch",
					fmt.Sprintf(
						"SE01 reports %d segments but %d were found from ST through SE.",
						expected,
						actual,
					),
					"Set SE01 to the count of segments from ST through SE, inclusive.",
				))
			}
			openST = nil
		}
	}
	if openST != nil {
		diagnostics = append(diagnostics, controlDiagnostic(
			openST,
			"x12.transaction.missing_se",
			"ST transaction is missing its SE trailer.",
			"Add an SE segment with matching SE02 control number.",
		))
	}
	return diagnostics
}

func validateGroupControls(segments []X12Segment) []NormalizedDiagnostic {
	diagnostics := []NormalizedDiagnostic{}
	var openGS *X12Segment
	transactionCount := 0
	for i := range segments {
		segment := &segments[i]
		switch segment.SegmentID {
		case "GS":
			openGS = segment
			transactionCount = 0
		case "ST":
			if openGS != nil {
				transactionCount++
			}
		case "GE":
			if openGS == nil {
				diagnostics = append(diagnostics, controlDiagnostic(
					segment,
					"x12.ge.without_gs",
					"GE appears before an open GS functional group.",
					"Move GE after the matching GS functional group.",
				))
				continue
			}
			gsControl := elementValue(openGS, 6)
			geControl := elementValue(segment, 2)
			if gsControl != "" && geControl != "" && gsControl != geControl {
				diagnostics = append(diagnostics, controlDiagnostic(
					segment,
					"x12.control.gs_ge_mismatch",
					fmt.Sprintf("GS06 %s does not match GE02 %s.", gsControl, geControl),
					"Set GE02 to the same group control number as GS06.",
				))
			}
			expected := parsePositiveInt(elementValue(segment, 1))
			if expected > 0 && expected != transactionCount {
				diagnostics = append(diagnostics, controlDiagnostic(
					segment,
					"x12.count.ge01_mismatch",
					fmt.Sprintf(
						"GE01 reports %d transaction sets but %d were found.",
						expected,
						transactionCount,
					),
					"Set GE01 to the number of ST/SE transaction sets in the group.",
				))
			}
			openGS = nil
		}
	}
	if openGS != nil {
		diagnostics = append(diagnostics, controlDiagnostic(
			openGS,
			"x12.group.missing_ge",
			"GS functional group is missing its GE trailer.",
			"Add a GE segment with matching GE02 control number.",
		))
	}
	return diagnostics
}

func validateInterchangeControls(segments []X12Segment) []NormalizedDiagnostic {
	diagnostics := []NormalizedDiagnostic{}
	isa := firstSegment(segments, "ISA")
	iea := lastSegment(segments, "IEA")
	if isa == nil || iea == nil {
		return diagnostics
	}
	isaControl := elementValue(isa, 13)
	ieaControl := elementValue(iea, 2)
	if isaControl != "" && ieaControl != "" && isaControl != ieaControl {
		diagnostics = append(diagnostics, controlDiagnostic(
			iea,
			"x12.control.isa_iea_mismatch",
			fmt.Sprintf("ISA13 %s does not match IEA02 %s.", isaControl, ieaControl),
			"Set IEA02 to the same interchange control number as ISA13.",
		))
	}
	expected := parsePositiveInt(elementValue(iea, 1))
	actual := countSegments(segments, "GS")
	if expected > 0 && expected != actual {
		diagnostics = append(diagnostics, controlDiagnostic(
			iea,
			"x12.count.iea01_mismatch",
			fmt.Sprintf("IEA01 reports %d functional groups but %d were found.", expected, actual),
			"Set IEA01 to the number of GS/GE groups in the interchange.",
		))
	}
	return diagnostics
}

func controlDiagnostic(
	segment *X12Segment,
	code string,
	message string,
	suggestedFix string,
) NormalizedDiagnostic {
	return NormalizedDiagnostic{
		Severity:     edi.ValidationSeverityError,
		Code:         code,
		Source:       DiagnosticSourceInspection,
		SegmentID:    segment.SegmentID,
		SegmentIndex: segment.Index,
		Message:      message,
		SuggestedFix: suggestedFix,
	}
}
