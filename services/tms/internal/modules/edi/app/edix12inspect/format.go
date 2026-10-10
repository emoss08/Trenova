package edix12inspect

import (
	"strconv"
	"strings"
)

func formatSegments(segments []X12Segment, diagnostics []NormalizedDiagnostic) string {
	var builder strings.Builder
	for i := range segments {
		segment := &segments[i]
		if i > 0 {
			builder.WriteByte('\n')
		}
		prefix := indentationForSegment(segment.SegmentID)
		builder.WriteString(prefix)
		builder.WriteString(segment.SegmentID)
		builder.WriteString(" - ")
		builder.WriteString(segment.Name)
		for _, element := range segment.Elements {
			builder.WriteByte('\n')
			builder.WriteString(prefix)
			builder.WriteString("  ")
			builder.WriteString(segment.SegmentID)
			builder.WriteString(twoDigit(element.Position))
			builder.WriteByte(' ')
			builder.WriteString(element.Label)
			builder.WriteString(": ")
			if element.Empty {
				builder.WriteString("[empty]")
			} else {
				builder.WriteString(element.Value)
			}
			if len(element.Components) > 1 {
				builder.WriteString(" (")
				for j, component := range element.Components {
					if j > 0 {
						builder.WriteString(" > ")
					}
					if component.Empty {
						builder.WriteString("[empty]")
					} else {
						builder.WriteString(component.Value)
					}
				}
				builder.WriteByte(')')
			}
		}
		for _, diagnosticIndex := range diagnosticIndexesForSegment(diagnostics, segment.Index) {
			diagnostic := &diagnostics[diagnosticIndex]
			builder.WriteByte('\n')
			builder.WriteString(prefix)
			builder.WriteString("  ! ")
			builder.WriteString(string(diagnostic.Severity))
			builder.WriteString(": ")
			builder.WriteString(diagnostic.Message)
		}
	}
	return builder.String()
}

func diagnosticIndexesForSegment(
	diagnostics []NormalizedDiagnostic,
	segmentIndex int,
) []int {
	matches := make([]int, 0)
	for i := range diagnostics {
		if diagnostics[i].SegmentIndex == segmentIndex {
			matches = append(matches, i)
		}
	}
	return matches
}

func indentationForSegment(segmentID string) string {
	switch segmentID {
	case "ISA", "IEA":
		return ""
	case "GS", "GE":
		return "  "
	default:
		return "    "
	}
}

func twoDigit(value int) string {
	if value < 10 {
		return "0" + strconv.Itoa(value)
	}
	return strconv.Itoa(value)
}
