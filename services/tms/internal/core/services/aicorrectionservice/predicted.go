package aicorrectionservice

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/aicorrection"
	"github.com/emoss08/trenova/shared/boolutils"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	maxValueRunes    = 500
	maxStopsPerRole  = 25
	maxSnapshotField = 100
	rolePickup       = "pickup"
	roleDelivery     = "delivery"
)

type fieldMeta struct {
	source     string
	confidence float64
}

type prediction struct {
	snapshot   *aicorrection.Snapshot
	fieldMeta  map[string]fieldMeta
	stopMeta   []fieldMeta
	kind       string
	confidence float64
	issuer     string
}

func (p *prediction) empty() bool {
	return len(p.snapshot.Fields) == 0 && len(p.snapshot.Stops) == 0
}

func readPrediction(data map[string]any) *prediction {
	p := &prediction{
		snapshot:  &aicorrection.Snapshot{Fields: map[string]string{}, Stops: []aicorrection.StopSnapshot{}},
		fieldMeta: map[string]fieldMeta{},
	}
	if data == nil {
		return p
	}

	p.kind = sliceutils.StringValue(data["kind"])
	p.issuer = sliceutils.StringValue(data["providerFingerprint"])
	p.confidence = confidenceValue(data["overallConfidence"])

	if fields, ok := data["fields"].(map[string]any); ok {
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			if len(p.snapshot.Fields) >= maxSnapshotField {
				break
			}
			field, isMap := fields[key].(map[string]any)
			if !isMap {
				continue
			}
			value := boundedText(sliceutils.StringValue(field["value"]))
			if value == "" {
				continue
			}
			p.snapshot.Fields[key] = value
			p.fieldMeta[key] = fieldMeta{
				source:     sliceutils.StringValue(field["source"]),
				confidence: confidenceValue(field["confidence"]),
			}
		}
	}

	if stops, ok := data["stops"].([]any); ok {
		perRole := map[string]int{}
		for _, raw := range stops {
			stop, isMap := raw.(map[string]any)
			if !isMap {
				continue
			}
			role := predictedRole(sliceutils.StringValue(stop["role"]))
			if perRole[role] >= maxStopsPerRole {
				continue
			}
			perRole[role]++
			p.snapshot.Stops = append(p.snapshot.Stops, aicorrection.StopSnapshot{
				Role:                role,
				Sequence:            intutils.IntValue(stop["sequence"]),
				Name:                boundedText(sliceutils.StringValue(stop["name"])),
				AddressLine1:        boundedText(sliceutils.StringValue(stop["addressLine1"])),
				AddressLine2:        boundedText(sliceutils.StringValue(stop["addressLine2"])),
				City:                boundedText(sliceutils.StringValue(stop["city"])),
				State:               boundedText(sliceutils.StringValue(stop["state"])),
				PostalCode:          boundedText(sliceutils.StringValue(stop["postalCode"])),
				Date:                boundedText(sliceutils.StringValue(stop["date"])),
				TimeWindow:          boundedText(sliceutils.StringValue(stop["timeWindow"])),
				AppointmentRequired: boolutils.BooleanValue(stop["appointmentRequired"]),
			})
			p.stopMeta = append(p.stopMeta, fieldMeta{
				source:     sliceutils.StringValue(stop["source"]),
				confidence: confidenceValue(stop["confidence"]),
			})
		}
	}

	return p
}

func predictedRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "delivery", "consignee", "receiver", "drop", "destination":
		return roleDelivery
	default:
		return rolePickup
	}
}

func confidenceValue(v any) float64 {
	return floatutils.Clamp(floatutils.FloatValue(v), 0, 1)
}

func boundedText(value string) string {
	return stringutils.TruncateRunes(stringutils.CollapseWhitespace(value), maxValueRunes)
}
