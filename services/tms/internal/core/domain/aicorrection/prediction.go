package aicorrection

import (
	"maps"
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/boolutils"
	"github.com/emoss08/trenova/shared/floatutils"
	"github.com/emoss08/trenova/shared/intutils"
	"github.com/emoss08/trenova/shared/sliceutils"
	"github.com/emoss08/trenova/shared/stringutils"
)

const (
	MaxValueRunes    = 500
	MaxStopsPerRole  = 25
	maxSnapshotField = 100
	RolePickup       = "pickup"
	RoleDelivery     = "delivery"
)

type FieldMeta struct {
	Source     string
	Confidence float64
}

type Prediction struct {
	Snapshot   *Snapshot
	FieldMeta  map[string]FieldMeta
	StopMeta   []FieldMeta
	Kind       string
	Confidence float64
	Issuer     string
}

func (p *Prediction) Empty() bool {
	return len(p.Snapshot.Fields) == 0 && len(p.Snapshot.Stops) == 0
}

func ReadPrediction(data map[string]any) *Prediction {
	p := &Prediction{
		Snapshot:  &Snapshot{Fields: map[string]string{}, Stops: []StopSnapshot{}},
		FieldMeta: map[string]FieldMeta{},
	}
	if data == nil {
		return p
	}

	p.Kind = sliceutils.StringValue(data["kind"])
	p.Issuer = sliceutils.StringValue(data["providerFingerprint"])
	p.Confidence = confidenceValue(data["overallConfidence"])

	if fields, ok := data["fields"].(map[string]any); ok {
		for _, key := range slices.Sorted(maps.Keys(fields)) {
			if len(p.Snapshot.Fields) >= maxSnapshotField {
				break
			}
			field, isMap := fields[key].(map[string]any)
			if !isMap {
				continue
			}
			value := BoundedText(sliceutils.StringValue(field["value"]))
			if value == "" {
				continue
			}
			p.Snapshot.Fields[key] = value
			p.FieldMeta[key] = FieldMeta{
				Source:     sliceutils.StringValue(field["source"]),
				Confidence: confidenceValue(field["confidence"]),
			}
		}
	}

	if stops := stopMaps(data["stops"]); len(stops) > 0 {
		perRole := map[string]int{}
		for _, stop := range stops {
			role := predictedRole(sliceutils.StringValue(stop["role"]))
			if perRole[role] >= MaxStopsPerRole {
				continue
			}
			perRole[role]++
			p.Snapshot.Stops = append(p.Snapshot.Stops, StopSnapshot{
				Role:                role,
				Sequence:            intutils.IntValue(stop["sequence"]),
				Name:                BoundedText(sliceutils.StringValue(stop["name"])),
				AddressLine1:        BoundedText(sliceutils.StringValue(stop["addressLine1"])),
				AddressLine2:        BoundedText(sliceutils.StringValue(stop["addressLine2"])),
				City:                BoundedText(sliceutils.StringValue(stop["city"])),
				State:               BoundedText(sliceutils.StringValue(stop["state"])),
				PostalCode:          BoundedText(sliceutils.StringValue(stop["postalCode"])),
				Date:                BoundedText(sliceutils.StringValue(stop["date"])),
				TimeWindow:          BoundedText(sliceutils.StringValue(stop["timeWindow"])),
				AppointmentRequired: boolutils.BooleanValue(stop["appointmentRequired"]),
			})
			p.StopMeta = append(p.StopMeta, FieldMeta{
				Source:     sliceutils.StringValue(stop["source"]),
				Confidence: confidenceValue(stop["confidence"]),
			})
		}
	}

	return p
}

func stopMaps(raw any) []map[string]any {
	switch stops := raw.(type) {
	case []map[string]any:
		return stops
	case []any:
		out := make([]map[string]any, 0, len(stops))
		for _, item := range stops {
			if stop, ok := item.(map[string]any); ok {
				out = append(out, stop)
			}
		}
		return out
	default:
		return nil
	}
}

func predictedRole(role string) string {
	switch strings.ToLower(strings.TrimSpace(role)) {
	case "delivery", "consignee", "receiver", "drop", "destination":
		return RoleDelivery
	default:
		return RolePickup
	}
}

func confidenceValue(v any) float64 {
	return floatutils.Clamp(floatutils.FloatValue(v), 0, 1)
}

func BoundedText(value string) string {
	return stringutils.TruncateRunes(stringutils.CollapseWhitespace(value), MaxValueRunes)
}
