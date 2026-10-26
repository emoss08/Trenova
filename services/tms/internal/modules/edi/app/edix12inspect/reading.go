package edix12inspect

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edi"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/i18n"
)

func findingDiagnostics(reading *x12.Reading, locale i18n.Locale) []NormalizedDiagnostic {
	diagnostics := make([]NormalizedDiagnostic, 0, reading.Summary.Errors+reading.Summary.Warnings)
	add := func(findings []x12.Finding) {
		for index := range findings {
			diagnostics = append(diagnostics, FindingDiagnostic(&findings[index], locale))
		}
	}
	add(reading.Findings)
	for interchangeIndex := range reading.Interchanges {
		interchange := &reading.Interchanges[interchangeIndex]
		add(interchange.Findings)
		for groupIndex := range interchange.Groups {
			group := &interchange.Groups[groupIndex]
			add(group.Findings)
			for transactionIndex := range group.Transactions {
				add(group.Transactions[transactionIndex].Findings)
			}
		}
	}
	return diagnostics
}

const FindingCodePrefix = "x12."

func FindingDiagnostic(finding *x12.Finding, locale i18n.Locale) NormalizedDiagnostic {
	diagnostic := NormalizedDiagnostic{
		Severity:  FindingSeverity(finding.Severity),
		Code:      FindingCodePrefix + finding.Kind,
		Source:    DiagnosticSourceValidation,
		SegmentID: finding.SegmentID,
		Path:      FindingPath(finding),
		Message:   finding.Message.Localize(locale),
	}
	if finding.Segment != nil {
		diagnostic.SegmentIndex = *finding.Segment + 1
	}
	if finding.Element != nil {
		diagnostic.ElementPosition = *finding.Element
	}
	return diagnostic
}

func FindingSeverity(severity x12.Severity) edi.ValidationSeverity {
	switch severity {
	case x12.SeverityError:
		return edi.ValidationSeverityError
	case x12.SeverityWarning:
		return edi.ValidationSeverityWarning
	case x12.SeverityInfo:
	}
	return edi.ValidationSeverityInfo
}

func FindingPath(finding *x12.Finding) string {
	parts := make([]string, 0, 3)
	if finding.LoopID != "" {
		parts = append(parts, "loop "+finding.LoopID)
	}
	if finding.Position != nil {
		parts = append(parts, "segment "+strconv.Itoa(*finding.Position))
	}
	if finding.Element != nil && finding.SegmentID != "" {
		label := finding.SegmentID + twoDigit(*finding.Element)
		if finding.Component != nil {
			label += "-" + twoDigit(*finding.Component)
		}
		parts = append(parts, label)
	}
	return strings.Join(parts, " · ")
}

func envelopeFromReading(reading *x12.Reading) X12Envelope {
	envelope := X12Envelope{}
	for index := range reading.Interchanges {
		envelope.ActualGroups += len(reading.Interchanges[index].Groups)
	}
	if len(reading.Interchanges) == 0 {
		return envelope
	}
	first := &reading.Interchanges[0]
	if first.Header != nil {
		envelope.ISAControlNumber = first.Header.ControlNumber
	}
	last := &reading.Interchanges[len(reading.Interchanges)-1]
	if last.Trailer != nil {
		envelope.IEAControlNumber = last.Trailer.ControlNumber
		envelope.ExpectedGroups = parsePositiveInt(last.Trailer.Count)
	}
	return envelope
}

func groupsFromReading(reading *x12.Reading) []X12FunctionalGroup {
	groups := make([]X12FunctionalGroup, 0, reading.Summary.Groups)
	for interchangeIndex := range reading.Interchanges {
		interchange := &reading.Interchanges[interchangeIndex]
		for groupIndex := range interchange.Groups {
			group := &interchange.Groups[groupIndex]
			if group.Header == nil {
				continue
			}
			converted := X12FunctionalGroup{
				Index:             len(groups) + 1,
				FunctionalIDCode:  group.Header.FunctionalID,
				GSControlNumber:   group.Header.ControlNumber,
				ActualCount:       len(group.Transactions),
				StartSegmentIndex: group.FirstSegment + 1,
			}
			if group.Trailer != nil {
				converted.GEControlNumber = group.Trailer.ControlNumber
				converted.ExpectedCount = parsePositiveInt(group.Trailer.Count)
				converted.EndSegmentIndex = group.Trailer.Segment + 1
			}
			groups = append(groups, converted)
		}
	}
	return groups
}

func transactionsFromReading(reading *x12.Reading) []X12Transaction {
	transactions := make([]X12Transaction, 0, reading.Summary.Transactions)
	for _, transaction := range AllTransactions(reading) {
		converted := X12Transaction{
			Index:             transaction.Index + 1,
			TransactionSet:    transaction.TransactionSet,
			STControlNumber:   transaction.ControlNumber,
			ActualSegments:    transaction.LastSegment - transaction.FirstSegment + 1,
			StartSegmentIndex: transaction.FirstSegment + 1,
		}
		if transaction.Trailer != nil {
			converted.SEControlNumber = transaction.Trailer.ControlNumber
			converted.ExpectedSegments = parsePositiveInt(transaction.Trailer.Count)
			converted.EndSegmentIndex = transaction.Trailer.Segment + 1
		}
		transactions = append(transactions, converted)
	}
	return transactions
}

func AllTransactions(reading *x12.Reading) []*x12.Transaction {
	transactions := make([]*x12.Transaction, 0, reading.Summary.Transactions)
	for interchangeIndex := range reading.Interchanges {
		interchange := &reading.Interchanges[interchangeIndex]
		for groupIndex := range interchange.Groups {
			group := &interchange.Groups[groupIndex]
			for transactionIndex := range group.Transactions {
				transactions = append(transactions, &group.Transactions[transactionIndex])
			}
		}
	}
	return transactions
}

func readingVersion(reading *x12.Reading) string {
	for interchangeIndex := range reading.Interchanges {
		groups := reading.Interchanges[interchangeIndex].Groups
		for groupIndex := range groups {
			if header := groups[groupIndex].Header; header != nil && header.Version != "" {
				return header.Version
			}
		}
	}
	return ""
}

func buildSummary(result *InspectX12Result) InspectSummary {
	summary := InspectSummary{
		SegmentCount:     len(result.Segments),
		GroupCount:       len(result.Groups),
		TransactionCount: len(result.Transactions),
	}
	for index := range result.Diagnostics {
		switch result.Diagnostics[index].Severity {
		case edi.ValidationSeverityError:
			summary.ErrorCount++
		case edi.ValidationSeverityWarning:
			summary.WarningCount++
		case edi.ValidationSeverityInfo:
			summary.InfoCount++
		}
	}
	return summary
}

func splitComponents(value, componentSeparator string) []X12Component {
	if componentSeparator == "" || !strings.Contains(value, componentSeparator) {
		return []X12Component{{Position: 1, Value: value, Empty: value == ""}}
	}
	parts := strings.Split(value, componentSeparator)
	components := make([]X12Component, 0, len(parts))
	for index, part := range parts {
		components = append(components, X12Component{
			Position: index + 1,
			Value:    part,
			Empty:    part == "",
		})
	}
	return components
}

func normalizeRenderDiagnostics(
	diagnostics []edi.Diagnostic,
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

func sourceForDiagnostic(diagnostic *edi.Diagnostic) DiagnosticSource {
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

func segmentIndexForDiagnostic(segments []X12Segment, diagnostic *edi.Diagnostic) int {
	if diagnostic.SegmentID == "" {
		return 0
	}
	for index := range segments {
		if segments[index].SegmentID == diagnostic.SegmentID {
			return segments[index].Index
		}
	}
	return 0
}

func parsePositiveInt(value string) int {
	parsed, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil || parsed < 0 {
		return 0
	}
	return parsed
}

func envelopeSeparatorsPresent(envelope *edi.X12EnvelopeSettings) bool {
	return envelope != nil &&
		(envelope.ElementSeparator != "" || envelope.SegmentTerminator != "" ||
			envelope.ComponentSeparator != "" || envelope.RepetitionSeparator != "")
}

func separatorsEqual(left, right *X12Separators) bool {
	return left.Element == right.Element &&
		left.Segment == right.Segment &&
		left.Component == right.Component &&
		(right.Repetition == "" || left.Repetition == right.Repetition)
}
