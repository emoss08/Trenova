package shipment

import (
	"github.com/emoss08/trenova/pkg/errortypes"
)

type QuickFilter string

const (
	QuickFilterLate            = QuickFilter("Late")
	QuickFilterUncovered       = QuickFilter("Uncovered")
	QuickFilterMoving          = QuickFilter("Moving")
	QuickFilterDeliveringToday = QuickFilter("DeliveringToday")
	QuickFilterReefer          = QuickFilter("Reefer")
	QuickFilterLowMargin       = QuickFilter("LowMargin")
	QuickFilterDeliveryHour    = QuickFilter("DeliveryHour")
	QuickFilterPickupWindow    = QuickFilter("PickupWindow")
	QuickFilterDetention       = QuickFilter("Detention")
	QuickFilterReadyToBill     = QuickFilter("ReadyToBill")
)

const (
	MaxQuickFilters   = 16
	hoursPerDay       = 24
	quickFilterPrefix = "quickFilters"
)

var countableQuickFilters = [...]QuickFilter{
	QuickFilterLate,
	QuickFilterUncovered,
	QuickFilterMoving,
	QuickFilterDeliveringToday,
	QuickFilterReefer,
	QuickFilterLowMargin,
	QuickFilterDetention,
	QuickFilterReadyToBill,
}

func CountableQuickFilters() []QuickFilter {
	return append([]QuickFilter(nil), countableQuickFilters[:]...)
}

func (q QuickFilter) String() string {
	return string(q)
}

func (q QuickFilter) IsValid() bool {
	switch q {
	case QuickFilterLate,
		QuickFilterUncovered,
		QuickFilterMoving,
		QuickFilterDeliveringToday,
		QuickFilterReefer,
		QuickFilterLowMargin,
		QuickFilterDeliveryHour,
		QuickFilterPickupWindow,
		QuickFilterDetention,
		QuickFilterReadyToBill:
		return true
	default:
		return false
	}
}

func (q QuickFilter) NeedsMargin() bool {
	return q == QuickFilterLowMargin
}

func (q QuickFilter) NeedsDetention() bool {
	return q == QuickFilterDetention
}

type QuickFilterSpec struct {
	Filter             QuickFilter `json:"filter"`
	Hour               *int        `json:"hour,omitempty"`
	WindowStartMinutes *int        `json:"windowStartMinutes,omitempty"`
	WindowEndMinutes   *int        `json:"windowEndMinutes,omitempty"`
}

func Quick(filter QuickFilter) QuickFilterSpec {
	return QuickFilterSpec{Filter: filter}
}

func DeliveryHourFilter(hour int) QuickFilterSpec {
	return QuickFilterSpec{Filter: QuickFilterDeliveryHour, Hour: &hour}
}

func PickupWindowFilter(startMinutes int, endMinutes *int) QuickFilterSpec {
	return QuickFilterSpec{
		Filter:             QuickFilterPickupWindow,
		WindowStartMinutes: &startMinutes,
		WindowEndMinutes:   endMinutes,
	}
}

func (s *QuickFilterSpec) Validate(multiErr *errortypes.MultiError) {
	if !s.Filter.IsValid() {
		multiErr.Add("filter", errortypes.ErrInvalid, "Unknown quick filter")
		return
	}

	switch s.Filter {
	case QuickFilterDeliveryHour:
		if s.Hour == nil {
			multiErr.Add(
				"hour",
				errortypes.ErrRequired,
				"Hour is required for a delivery hour filter",
			)
			return
		}
		if *s.Hour < 0 || *s.Hour >= hoursPerDay {
			multiErr.Add("hour", errortypes.ErrInvalid, "Hour must be between 0 and 23")
		}
	case QuickFilterPickupWindow:
		if s.WindowStartMinutes == nil {
			multiErr.Add(
				"windowStartMinutes",
				errortypes.ErrRequired,
				"Window start is required for a pickup window filter",
			)
			return
		}
		if *s.WindowStartMinutes < 0 {
			multiErr.Add(
				"windowStartMinutes",
				errortypes.ErrInvalid,
				"Window start cannot be negative",
			)
		}
		if s.WindowEndMinutes != nil && *s.WindowEndMinutes <= *s.WindowStartMinutes {
			multiErr.Add(
				"windowEndMinutes",
				errortypes.ErrInvalid,
				"Window end must be after the window start",
			)
		}
	case QuickFilterLate,
		QuickFilterUncovered,
		QuickFilterMoving,
		QuickFilterDeliveringToday,
		QuickFilterReefer,
		QuickFilterLowMargin,
		QuickFilterDetention,
		QuickFilterReadyToBill:
		if s.Hour != nil || s.WindowStartMinutes != nil || s.WindowEndMinutes != nil {
			multiErr.Add("filter", errortypes.ErrInvalid, "This quick filter takes no parameters")
		}
	}
}

func ValidateQuickFilters(specs []QuickFilterSpec, multiErr *errortypes.MultiError) {
	if len(specs) > MaxQuickFilters {
		multiErr.Add(quickFilterPrefix, errortypes.ErrInvalid, "Too many quick filters")
		return
	}
	for i := range specs {
		specs[i].Validate(multiErr.WithIndex(quickFilterPrefix, i))
	}
}

func QuickFiltersNeed(specs []QuickFilterSpec) (margin, detention bool) {
	for i := range specs {
		margin = margin || specs[i].Filter.NeedsMargin()
		detention = detention || specs[i].Filter.NeedsDetention()
	}
	return margin, detention
}
