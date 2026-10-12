package shipment

import "time"

type PickupWindow string

const (
	PickupWindowUnderTwoHours   = PickupWindow("UnderTwoHours")
	PickupWindowTwoToSixHours   = PickupWindow("TwoToSixHours")
	PickupWindowLaterToday      = PickupWindow("LaterToday")
	PickupWindowTomorrowOrLater = PickupWindow("TomorrowOrLater")
)

const (
	twoHoursInMinutes = 120
	sixHoursInMinutes = 360
)

func (w PickupWindow) String() string {
	return string(w)
}

func (w PickupWindow) IsValid() bool {
	switch w {
	case PickupWindowUnderTwoHours,
		PickupWindowTwoToSixHours,
		PickupWindowLaterToday,
		PickupWindowTomorrowOrLater:
		return true
	default:
		return false
	}
}

type PickupWindowBounds struct {
	Window       PickupWindow
	StartMinutes int
	EndMinutes   *int
}

func (b PickupWindowBounds) Spec() QuickFilterSpec {
	return PickupWindowFilter(b.StartMinutes, b.EndMinutes)
}

func (b PickupWindowBounds) Empty() bool {
	return b.EndMinutes != nil && *b.EndMinutes <= b.StartMinutes
}

func MinutesToLocalMidnight(now time.Time, loc *time.Location) int {
	local := now.In(loc)
	midnight := time.Date(local.Year(), local.Month(), local.Day()+1, 0, 0, 0, 0, loc)
	return int(midnight.Sub(local) / time.Minute)
}

func PickupWindowsAt(now time.Time, loc *time.Location) []PickupWindowBounds {
	midnight := MinutesToLocalMidnight(now, loc)
	twoHours := min(twoHoursInMinutes, midnight)
	sixHours := min(sixHoursInMinutes, midnight)

	return []PickupWindowBounds{
		{Window: PickupWindowUnderTwoHours, StartMinutes: 0, EndMinutes: &twoHours},
		{Window: PickupWindowTwoToSixHours, StartMinutes: twoHours, EndMinutes: &sixHours},
		{Window: PickupWindowLaterToday, StartMinutes: sixHours, EndMinutes: &midnight},
		{Window: PickupWindowTomorrowOrLater, StartMinutes: midnight},
	}
}
