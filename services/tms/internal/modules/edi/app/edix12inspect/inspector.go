package edix12inspect

import (
	"context"
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/edi/domain/edidocument"
	"github.com/emoss08/trenova/pkg/x12"
	"github.com/emoss08/trenova/shared/i18n"
)

const (
	segmentTypeInterchange = "interchange"
	segmentTypeGroup       = "group"
	segmentTypeTransaction = "transaction"
	segmentTypeSegment     = "segment"
)

func InspectX12(ctx context.Context, req *InspectX12Request) (*InspectX12Result, error) {
	if strings.TrimSpace(req.RawX12) == "" {
		result := &InspectX12Result{
			InspectOverview: InspectOverview{
				TransactionSet: req.TransactionSet,
				X12Version:     req.X12Version,
				Diagnostics: []NormalizedDiagnostic{
					{
						Severity:     edidocument.ValidationSeverityError,
						Code:         "x12.document.empty",
						Source:       DiagnosticSourceInspection,
						Message:      "X12 payload is empty.",
						SuggestedFix: "Paste or generate a complete X12 interchange before inspecting it.",
					},
				},
			},
			RawX12:   req.RawX12,
			Segments: []X12Segment{},
		}
		result.Summary = buildSummary(result)
		return result, nil
	}

	reading, err := x12.Default().Read(ctx, &x12.ReadRequest{
		Text:            req.RawX12,
		Separators:      envelopeSeparators(req.Envelope),
		IncludeSegments: true,
		Fragment:        IsFragment(req.RawX12),
	})
	if err != nil {
		return nil, fmt.Errorf("inspect X12: %w", err)
	}
	return resultFromReading(req, reading, i18n.FromContext(ctx)), nil
}

func IsFragment(rawX12 string) bool {
	return !strings.HasPrefix(strings.TrimLeft(rawX12, " \t\r\n"), isaSegmentID)
}

func DetectSeparators(
	ctx context.Context,
	rawX12 string,
	envelope *edidocument.X12EnvelopeSettings,
) (X12Separators, error) {
	reading, err := x12.Default().Read(ctx, &x12.ReadRequest{
		Text:       rawX12,
		Separators: envelopeSeparators(envelope),
		Fragment:   IsFragment(rawX12),
	})
	if err != nil {
		return X12Separators{}, fmt.Errorf("detect X12 separators: %w", err)
	}
	separators, _ := separatorsFromReading(reading, envelope)
	return separators, nil
}

func resultFromReading(
	req *InspectX12Request,
	reading *x12.Reading,
	locale i18n.Locale,
) *InspectX12Result {
	separators, separatorDiagnostics := separatorsFromReading(reading, req.Envelope)
	segments := segmentsFromReading(req.RawX12, reading, &separators)
	diagnostics := make([]NormalizedDiagnostic, 0, len(separatorDiagnostics)+8)
	diagnostics = append(diagnostics, separatorDiagnostics...)
	diagnostics = append(diagnostics, findingDiagnostics(reading, locale)...)
	diagnostics = append(diagnostics, normalizeRenderDiagnostics(req.Diagnostics, segments)...)

	result := &InspectX12Result{
		InspectOverview: InspectOverview{
			TransactionSet: req.TransactionSet,
			X12Version:     req.X12Version,
			Separators:     separators,
			Diagnostics:    diagnostics,
			Envelope:       envelopeFromReading(reading),
			Groups:         groupsFromReading(reading),
			Transactions:   transactionsFromReading(reading),
		},
		RawX12:    req.RawX12,
		Segments:  segments,
		Formatted: formatSegments(segments, diagnostics),
		Reading:   reading,
	}
	if result.TransactionSet == "" && len(result.Transactions) > 0 {
		result.TransactionSet = edidocument.TransactionSet(result.Transactions[0].TransactionSet)
	}
	if result.X12Version == "" {
		result.X12Version = readingVersion(reading)
	}
	result.Summary = buildSummary(result)
	return result
}

func envelopeSeparators(envelope *edidocument.X12EnvelopeSettings) *x12.Separators {
	if !envelopeSeparatorsPresent(envelope) {
		return nil
	}
	return &x12.Separators{
		Element:    envelope.ElementSeparator,
		Component:  envelope.ComponentSeparator,
		Repetition: envelope.RepetitionSeparator,
		Segment:    envelope.SegmentTerminator,
	}
}

func separatorsFromReading(
	reading *x12.Reading,
	envelope *edidocument.X12EnvelopeSettings,
) (X12Separators, []NormalizedDiagnostic) {
	separators := X12Separators{
		Element:    reading.Separators.Element,
		Segment:    reading.Separators.Segment,
		Component:  reading.Separators.Component,
		Repetition: reading.Separators.Repetition,
	}
	separators.Source = SeparatorSourceFallback
	switch reading.SeparatorSource {
	case x12.SeparatorSourceISA:
		separators.Source = SeparatorSourceISA
	case x12.SeparatorSourceRequest:
		separators.Source = SeparatorSourceEnvelope
	case x12.SeparatorSourceDetected:
	}
	if separators.Source != SeparatorSourceISA || !envelopeSeparatorsPresent(envelope) {
		return separators, nil
	}
	hints := X12Separators{
		Element:    envelope.ElementSeparator,
		Segment:    envelope.SegmentTerminator,
		Component:  envelope.ComponentSeparator,
		Repetition: envelope.RepetitionSeparator,
	}
	if separatorsEqual(&separators, &hints) {
		return separators, nil
	}
	separators.HasConflict = true
	return separators, []NormalizedDiagnostic{{
		Severity:  edidocument.ValidationSeverityWarning,
		Code:      "x12.separator.conflict",
		Source:    DiagnosticSourceInspection,
		SegmentID: isaSegmentID,
		Message: "Envelope separator hints differ from the ISA separators; " +
			"ISA separators were used.",
		SuggestedFix: "Align the partner envelope settings with the delimiters present in ISA.",
	}}
}

func segmentsFromReading(
	rawX12 string,
	reading *x12.Reading,
	separators *X12Separators,
) []X12Segment {
	segments := make([]X12Segment, 0, len(reading.Segments))
	for index := range reading.Segments {
		source := &reading.Segments[index]
		definition, known := reading.Definitions[source.ID]
		segment := X12Segment{
			Index:             source.Index + 1,
			SegmentID:         source.ID,
			Name:              segmentName(source.ID, &definition, known),
			Type:              segmentType(source.ID),
			Loop:              source.LoopPath,
			Raw:               rawX12[source.Start:source.End],
			RawWithTerminator: withTerminator(rawX12, source.Start, source.End, separators.Segment),
			StartOffset:       source.Start,
			EndOffset:         source.End,
			Malformed:         !validSegmentID(source.ID),
		}
		segment.EndOffset = source.Start + len(segment.RawWithTerminator)
		if source.Transaction != nil {
			segment.TransactionIndex = *source.Transaction + 1
		}
		segment.Elements = elementsFromSegment(source, &definition, separators)
		segments = append(segments, segment)
	}
	return segments
}

func elementsFromSegment(
	source *x12.Segment,
	definition *x12.SegmentSummary,
	separators *X12Separators,
) []X12Element {
	elements := make([]X12Element, 0, len(source.Elements))
	cursor := source.Start + len(source.ID)
	for offset, value := range source.Elements {
		cursor += len(separators.Element)
		position := offset + 1
		element := X12Element{
			Position:    position,
			Label:       "Element " + twoDigit(position),
			Value:       value,
			Empty:       value == "",
			StartOffset: cursor,
			EndOffset:   cursor + len(value),
			Components:  splitComponents(value, separators.Component),
		}
		if offset < len(definition.Elements) {
			defined := &definition.Elements[offset]
			element.Label = defined.Name
			element.DataType = defined.Type
			element.Required = defined.Usage == x12.UsageMandatory
			element.Known = true
		}
		elements = append(elements, element)
		cursor += len(value)
	}
	return elements
}

func withTerminator(rawX12 string, start, end int, terminator string) string {
	if terminator != "" && strings.HasPrefix(rawX12[end:], terminator) {
		return rawX12[start : end+len(terminator)]
	}
	return rawX12[start:end]
}

func segmentName(id string, definition *x12.SegmentSummary, known bool) string {
	if known && definition.Name != "" {
		return definition.Name
	}
	return "Segment " + id
}

func segmentType(id string) string {
	switch id {
	case "ISA", "IEA":
		return segmentTypeInterchange
	case "GS", "GE":
		return segmentTypeGroup
	case "ST", "SE":
		return segmentTypeTransaction
	default:
		return segmentTypeSegment
	}
}

func validSegmentID(id string) bool {
	if len(id) < 2 || len(id) > 3 || id[0] < 'A' || id[0] > 'Z' {
		return false
	}
	for index := range len(id) {
		c := id[index]
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}
