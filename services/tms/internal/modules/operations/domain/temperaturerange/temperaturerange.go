package temperaturerange

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

const (
	Lowest  int16 = -100
	Highest int16 = 150
)

type Range struct {
	Min *int16
	Max *int16
}

func Of(low, high *int16) Range {
	return Range{Min: low, Max: high}
}

func FromDegrees(low, high *int) Range {
	return Range{Min: degrees(low), Max: degrees(high)}
}

func degrees(value *int) *int16 {
	if value == nil || *value < int(Lowest) || *value > int(Highest) {
		return nil
	}
	converted := int16(*value)
	return &converted
}

func (r Range) Set() bool {
	return r.Min != nil || r.Max != nil
}

func (r Range) Validate(multiErr *errortypes.MultiError) {
	for _, bound := range []struct {
		field string
		value *int16
	}{
		{field: "temperatureMin", value: r.Min},
		{field: "temperatureMax", value: r.Max},
	} {
		if bound.value != nil && (*bound.value < Lowest || *bound.value > Highest) {
			multiErr.Add(
				bound.field,
				errortypes.ErrInvalid,
				"Temperature must be between {0} and {1} °F",
				Lowest,
				Highest,
			)
		}
	}
	if r.Min != nil && r.Max != nil && *r.Max < *r.Min {
		multiErr.Add(
			"temperatureMax",
			errortypes.ErrInvalid,
			"Maximum temperature cannot be below the minimum temperature",
		)
	}
}

func (r Range) ValidateWithin(outer Range, name string, multiErr *errortypes.MultiError) {
	if outer.Min != nil && r.Min != nil && *r.Min < *outer.Min {
		multiErr.Add(
			"temperatureMin",
			errortypes.ErrInvalid,
			"{0} is kept no colder than {1} °F",
			name,
			*outer.Min,
		)
	}
	if outer.Max != nil && r.Max != nil && *r.Max > *outer.Max {
		multiErr.Add(
			"temperatureMax",
			errortypes.ErrInvalid,
			"{0} is kept no warmer than {1} °F",
			name,
			*outer.Max,
		)
	}
}

func Load(ranges []Range) Range {
	var overlap, envelope Range
	for _, each := range ranges {
		if !each.Set() {
			continue
		}
		overlap.Min = bound(overlap.Min, each.Min, func(a, b int16) bool { return a > b })
		overlap.Max = bound(overlap.Max, each.Max, func(a, b int16) bool { return a < b })
		for _, value := range []*int16{each.Min, each.Max} {
			envelope.Min = bound(envelope.Min, value, func(a, b int16) bool { return a < b })
			envelope.Max = bound(envelope.Max, value, func(a, b int16) bool { return a > b })
		}
	}
	if overlap.Min != nil && overlap.Max != nil && *overlap.Min > *overlap.Max {
		return envelope
	}
	return overlap
}

func bound(current, next *int16, better func(a, b int16) bool) *int16 {
	if next == nil {
		return current
	}
	if current == nil || better(*next, *current) {
		value := *next
		return &value
	}
	return current
}
