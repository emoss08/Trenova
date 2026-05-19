package edix12inspect

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/edi"
	"github.com/emoss08/trenova/internal/core/services/edix12"
)

const (
	defaultElementSeparator    = "*"
	defaultSegmentTerminator   = "~"
	defaultComponentSeparator  = ">"
	defaultRepetitionSeparator = "^"
	isaSegmentLength           = 105
	isaSegmentTerminatorOffset = 105
)

var segmentIDPattern = regexp.MustCompile(`^[A-Z0-9]{2,3}$`)

func InspectX12(req *InspectX12Request) InspectX12Result {
	separators, separatorDiagnostics := DetectSeparators(req.RawX12, req.Envelope)
	segments, parseDiagnostics := parseSegments(req.RawX12, &separators)
	diagnostics := make(
		[]NormalizedDiagnostic,
		0,
		len(separatorDiagnostics)+len(parseDiagnostics)+len(req.Diagnostics)+8,
	)
	diagnostics = append(diagnostics, separatorDiagnostics...)
	diagnostics = append(diagnostics, parseDiagnostics...)
	diagnostics = append(diagnostics, validateStructure(req, segments)...)
	diagnostics = append(diagnostics, normalizeRenderDiagnostics(req.Diagnostics, segments)...)

	result := InspectX12Result{
		RawX12:         req.RawX12,
		TransactionSet: req.TransactionSet,
		X12Version:     req.X12Version,
		Separators:     separators,
		Segments:       segments,
		Formatted:      formatSegments(segments, diagnostics),
		Diagnostics:    diagnostics,
	}
	result.Envelope = buildEnvelope(segments)
	result.Groups = buildGroups(segments)
	result.Transactions = buildTransactions(segments)
	result.Summary = buildSummary(&result)
	return result
}

func DetectSeparators(
	rawX12 string,
	envelope *edi.X12EnvelopeSettings,
) (X12Separators, []NormalizedDiagnostic) {
	fallback := X12Separators{
		Element:    defaultElementSeparator,
		Segment:    defaultSegmentTerminator,
		Component:  defaultComponentSeparator,
		Repetition: defaultRepetitionSeparator,
		Source:     SeparatorSourceFallback,
	}
	isa, ok := separatorsFromISA(rawX12)
	if !ok {
		if envelopeSeparatorsPresent(envelope) {
			return separatorsFromEnvelope(envelope), []NormalizedDiagnostic{
				{
					Severity:  edi.ValidationSeverityWarning,
					Code:      "x12.separator.isa_missing",
					Source:    DiagnosticSourceInspection,
					SegmentID: "ISA",
					Message: "Unable to read fixed-width ISA separators; " +
						"envelope separators were used.",
					SuggestedFix: "Confirm the ISA segment is present and exactly 105 " +
						"characters before its terminator.",
				},
			}
		}
		return fallback, []NormalizedDiagnostic{
			{
				Severity: edi.ValidationSeverityWarning,
				Code:     "x12.separator.fallback",
				Source:   DiagnosticSourceInspection,
				Message: "Unable to read X12 separators from ISA; default X12 " +
					"separators were used.",
				SuggestedFix: "Provide a valid ISA segment or envelope separators.",
			},
		}
	}
	if !envelopeSeparatorsPresent(envelope) {
		return isa, nil
	}

	envSeparators := separatorsFromEnvelope(envelope)
	if separatorsEqual(&isa, &envSeparators) {
		return isa, nil
	}
	isa.HasConflict = true
	return isa, []NormalizedDiagnostic{
		{
			Severity:  edi.ValidationSeverityWarning,
			Code:      "x12.separator.conflict",
			Source:    DiagnosticSourceInspection,
			SegmentID: "ISA",
			Message: "Envelope separator hints differ from the ISA separators; " +
				"ISA separators were used.",
			SuggestedFix: "Align the partner envelope settings with the delimiters present in ISA.",
		},
	}
}

func separatorsFromISA(rawX12 string) (X12Separators, bool) {
	start := strings.Index(rawX12, "ISA")
	if start < 0 || len(rawX12) <= start+isaSegmentTerminatorOffset {
		return X12Separators{}, false
	}
	terminator := rawX12[start+isaSegmentTerminatorOffset : start+isaSegmentTerminatorOffset+1]
	if isAlphaNumeric(terminator) {
		return X12Separators{}, false
	}
	return X12Separators{
		Element:    rawX12[start+3 : start+4],
		Repetition: rawX12[start+82 : start+83],
		Component:  rawX12[start+104 : start+105],
		Segment:    terminator,
		Source:     SeparatorSourceISA,
	}, true
}

func separatorsFromEnvelope(envelope *edi.X12EnvelopeSettings) X12Separators {
	return X12Separators{
		Element:    firstNonEmpty(envelope.ElementSeparator, defaultElementSeparator),
		Segment:    firstNonEmpty(envelope.SegmentTerminator, defaultSegmentTerminator),
		Component:  firstNonEmpty(envelope.ComponentSeparator, defaultComponentSeparator),
		Repetition: firstNonEmpty(envelope.RepetitionSeparator, defaultRepetitionSeparator),
		Source:     SeparatorSourceEnvelope,
	}
}

func parseSegments(
	rawX12 string,
	separators *X12Separators,
) ([]X12Segment, []NormalizedDiagnostic) {
	trimmed := strings.TrimSpace(rawX12)
	if trimmed == "" {
		return []X12Segment{}, []NormalizedDiagnostic{{
			Severity:     edi.ValidationSeverityError,
			Code:         "x12.document.empty",
			Source:       DiagnosticSourceInspection,
			Message:      "X12 payload is empty.",
			SuggestedFix: "Paste or generate a complete X12 interchange before inspecting it.",
		}}
	}

	segments := make([]X12Segment, 0, strings.Count(rawX12, separators.Segment)+1)
	diagnostics := []NormalizedDiagnostic{}
	searchFrom := 0
	chunks := strings.Split(rawX12, separators.Segment)
	transactionIndex := 0
	for chunkIndex, chunk := range chunks {
		start := strings.Index(rawX12[searchFrom:], chunk)
		if start < 0 {
			start = 0
		}
		startOffset := searchFrom + start
		searchFrom = startOffset + len(chunk)

		raw := strings.Trim(chunk, "\r\n\t ")
		leadingTrim := len(chunk) - len(strings.TrimLeft(chunk, "\r\n\t "))
		rawStartOffset := startOffset + leadingTrim
		if raw == "" {
			if chunkIndex == len(chunks)-1 {
				continue
			}
			searchFrom += len(separators.Segment)
			continue
		}

		hasTerminator := chunkIndex < len(chunks)-1
		rawWithTerminator := raw
		endOffset := rawStartOffset + len(raw)
		if hasTerminator {
			rawWithTerminator += separators.Segment
			endOffset += len(separators.Segment)
			searchFrom += len(separators.Segment)
		}

		segment := parseSegment(raw, rawWithTerminator, rawStartOffset, endOffset, separators)
		segment.Index = len(segments) + 1
		if segment.SegmentID == "ST" {
			transactionIndex++
		}
		segment.TransactionIndex = transactionIndex
		if segment.SegmentID == "SE" {
			segment.TransactionIndex = transactionIndex
		}
		segments = append(segments, segment)

		if segment.Malformed {
			diagnostics = append(diagnostics, NormalizedDiagnostic{
				Severity:     edi.ValidationSeverityError,
				Code:         "x12.segment.id_malformed",
				Source:       DiagnosticSourceInspection,
				SegmentID:    segment.SegmentID,
				SegmentIndex: segment.Index,
				Message: fmt.Sprintf(
					"Segment ID %q is not a valid X12 segment identifier.",
					segment.SegmentID,
				),
				SuggestedFix: "Use a two or three character uppercase alphanumeric segment ID.",
			})
		}
		if !hasTerminator {
			diagnostics = append(diagnostics, NormalizedDiagnostic{
				Severity:     edi.ValidationSeverityWarning,
				Code:         "x12.segment.missing_terminator",
				Source:       DiagnosticSourceInspection,
				SegmentID:    segment.SegmentID,
				SegmentIndex: segment.Index,
				Message:      "The final segment is missing the detected segment terminator.",
				SuggestedFix: "Append the segment terminator to the final X12 segment.",
			})
		}
	}

	return segments, diagnostics
}

func parseSegment(
	raw string,
	rawWithTerminator string,
	startOffset int,
	endOffset int,
	separators *X12Separators,
) X12Segment {
	parts := strings.Split(raw, separators.Element)
	segmentID := ""
	if len(parts) > 0 {
		segmentID = parts[0]
	}
	elements := make([]X12Element, 0, max(0, len(parts)-1))
	cursor := startOffset + len(segmentID)
	for position, value := range parts[1:] {
		cursor += len(separators.Element)
		label, known := elementLabel(segmentID, position+1)
		element := X12Element{
			Position:    position + 1,
			Label:       label,
			Value:       value,
			Empty:       value == "",
			Required:    elementRequired(segmentID, position+1),
			Known:       known,
			StartOffset: cursor,
			EndOffset:   cursor + len(value),
			Components:  splitComponents(value, separators.Component),
		}
		elements = append(elements, element)
		cursor += len(value)
	}

	return X12Segment{
		SegmentID:         segmentID,
		Name:              segmentName(segmentID),
		Type:              segmentType(segmentID),
		Loop:              segmentLoop(segmentID),
		Raw:               raw,
		RawWithTerminator: rawWithTerminator,
		StartOffset:       startOffset,
		EndOffset:         endOffset,
		Elements:          elements,
		Malformed:         !segmentIDPattern.MatchString(segmentID),
	}
}

func splitComponents(value, componentSeparator string) []X12Component {
	if componentSeparator == "" || !strings.Contains(value, componentSeparator) {
		return []X12Component{{
			Position: 1,
			Value:    value,
			Empty:    value == "",
		}}
	}
	parts := strings.Split(value, componentSeparator)
	components := make([]X12Component, 0, len(parts))
	for i, part := range parts {
		components = append(components, X12Component{
			Position: i + 1,
			Value:    part,
			Empty:    part == "",
		})
	}
	return components
}

func normalizeRenderDiagnostics(
	diagnostics []edix12.Diagnostic,
	segments []X12Segment,
) []NormalizedDiagnostic {
	normalized := make([]NormalizedDiagnostic, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		normalized = append(normalized, NormalizedDiagnostic{
			Severity:        diagnostic.Severity,
			Code:            diagnostic.Code,
			Source:          sourceForDiagnostic(&diagnostic),
			SegmentID:       diagnostic.SegmentID,
			SegmentIndex:    segmentIndexForDiagnostic(segments, &diagnostic),
			ElementPosition: diagnostic.ElementPosition,
			Path:            diagnostic.Path,
			Message:         diagnostic.Message,
			SuggestedFix:    diagnostic.SuggestedFix,
		})
	}
	return normalized
}

func sourceForDiagnostic(diagnostic *edix12.Diagnostic) DiagnosticSource {
	code := strings.ToLower(diagnostic.Code)
	path := strings.ToLower(diagnostic.Path)
	switch {
	case strings.Contains(code, "source_context") || strings.Contains(path, "sourcecontext"):
		return DiagnosticSourceSourceContext
	case strings.Contains(code, "partner_setting") || strings.Contains(path, "partnersettings"):
		return DiagnosticSourcePartnerSetting
	case strings.Contains(code, "starlark"):
		return DiagnosticSourceStarlark
	case strings.Contains(code, "condition"):
		return DiagnosticSourceCondition
	case strings.Contains(code, "transform"):
		return DiagnosticSourceTransform
	case strings.Contains(code, "validation"):
		return DiagnosticSourceValidation
	default:
		return DiagnosticSourceRender
	}
}

func segmentIndexForDiagnostic(segments []X12Segment, diagnostic *edix12.Diagnostic) int {
	if diagnostic.SegmentID == "" {
		return 0
	}
	for i := range segments {
		segment := &segments[i]
		if segment.SegmentID == diagnostic.SegmentID {
			return segment.Index
		}
	}
	return 0
}

func buildEnvelope(segments []X12Segment) X12Envelope {
	envelope := X12Envelope{ActualGroups: countSegments(segments, "GS")}
	if isa := firstSegment(segments, "ISA"); isa != nil {
		envelope.ISAControlNumber = elementValue(isa, 13)
	}
	if iea := lastSegment(segments, "IEA"); iea != nil {
		envelope.IEAControlNumber = elementValue(iea, 2)
		envelope.ExpectedGroups = parsePositiveInt(elementValue(iea, 1))
	}
	return envelope
}

func buildGroups(segments []X12Segment) []X12FunctionalGroup {
	groups := []X12FunctionalGroup{}
	var current *X12FunctionalGroup
	for i := range segments {
		segment := &segments[i]
		switch segment.SegmentID {
		case "GS":
			groups = append(groups, X12FunctionalGroup{
				Index:             len(groups) + 1,
				FunctionalIDCode:  elementValue(segment, 1),
				GSControlNumber:   elementValue(segment, 6),
				StartSegmentIndex: segment.Index,
			})
			current = &groups[len(groups)-1]
		case "ST":
			if current != nil {
				current.ActualCount++
			}
		case "GE":
			if current != nil {
				current.GEControlNumber = elementValue(segment, 2)
				current.ExpectedCount = parsePositiveInt(elementValue(segment, 1))
				current.EndSegmentIndex = segment.Index
				current = nil
			}
		}
	}
	return groups
}

func buildTransactions(segments []X12Segment) []X12Transaction {
	transactions := []X12Transaction{}
	var current *X12Transaction
	for i := range segments {
		segment := &segments[i]
		switch segment.SegmentID {
		case "ST":
			transactions = append(transactions, X12Transaction{
				Index:             len(transactions) + 1,
				TransactionSet:    elementValue(segment, 1),
				STControlNumber:   elementValue(segment, 2),
				StartSegmentIndex: segment.Index,
			})
			current = &transactions[len(transactions)-1]
		case "SE":
			if current != nil {
				current.SEControlNumber = elementValue(segment, 2)
				current.ExpectedSegments = parsePositiveInt(elementValue(segment, 1))
				current.EndSegmentIndex = segment.Index
				current.ActualSegments = segment.Index - current.StartSegmentIndex + 1
				current = nil
			}
		}
	}
	return transactions
}

func buildSummary(result *InspectX12Result) InspectSummary {
	summary := InspectSummary{
		SegmentCount:     len(result.Segments),
		GroupCount:       len(result.Groups),
		TransactionCount: len(result.Transactions),
	}
	for i := range result.Diagnostics {
		diagnostic := &result.Diagnostics[i]
		switch diagnostic.Severity {
		case edi.ValidationSeverityError:
			summary.ErrorCount++
		case edi.ValidationSeverityWarning:
			summary.WarningCount++
		case edi.ValidationSeverityInfo:
			summary.InfoCount++
		default:
			summary.InfoCount++
		}
	}
	return summary
}

func elementValue(segment *X12Segment, position int) string {
	if position <= 0 || position > len(segment.Elements) {
		return ""
	}
	return segment.Elements[position-1].Value
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed <= 0 {
		return 0
	}
	return parsed
}

func firstSegment(segments []X12Segment, segmentID string) *X12Segment {
	for i := range segments {
		if segments[i].SegmentID == segmentID {
			return &segments[i]
		}
	}
	return nil
}

func lastSegment(segments []X12Segment, segmentID string) *X12Segment {
	for i := len(segments) - 1; i >= 0; i-- {
		if segments[i].SegmentID == segmentID {
			return &segments[i]
		}
	}
	return nil
}

func countSegments(segments []X12Segment, segmentID string) int {
	count := 0
	for i := range segments {
		segment := &segments[i]
		if segment.SegmentID == segmentID {
			count++
		}
	}
	return count
}

func envelopeSeparatorsPresent(envelope *edi.X12EnvelopeSettings) bool {
	return envelope != nil &&
		(envelope.ElementSeparator != "" ||
			envelope.SegmentTerminator != "" ||
			envelope.ComponentSeparator != "" ||
			envelope.RepetitionSeparator != "")
}

func separatorsEqual(left, right *X12Separators) bool {
	return left.Element == right.Element &&
		left.Segment == right.Segment &&
		left.Component == right.Component &&
		left.Repetition == right.Repetition
}

func firstNonEmpty(value, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}

func isAlphaNumeric(value string) bool {
	if value == "" {
		return false
	}
	r := rune(value[0])
	return (r >= 'A' && r <= 'Z') ||
		(r >= 'a' && r <= 'z') ||
		(r >= '0' && r <= '9')
}
