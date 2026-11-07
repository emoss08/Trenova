package demandforecast

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/operations/domain/dispatcheconomics"
)

func Explain(reason *Reason, location *time.Location) string {
	if reason == nil {
		return ""
	}
	if location == nil {
		location = time.UTC
	}

	var b strings.Builder
	fmt.Fprintf(&b, "Ending near %s is worth %s: ",
		areaName(reason.AreaLabel), dispatcheconomics.Dollars(reason.ValueCents))
	b.WriteString(typicalLoads(reason.TypicalLow, reason.TypicalHigh))
	fmt.Fprintf(&b, " usually appear there %s", windowText(reason, location))
	if names := customerText(reason.Customers); names != "" {
		fmt.Fprintf(&b, " (%s)", names)
	}
	b.WriteString(", ")
	b.WriteString(bookedText(reason.Booked))
	b.WriteString(", ")
	b.WriteString(driversText(reason.DriversExpected))
	b.WriteString(".")

	return b.String()
}

func areaName(label string) string {
	if label == "" {
		return "this area"
	}

	return label
}

func typicalLoads(low, high int) string {
	switch {
	case high <= 0:
		return "few loads"
	case low == high && high == 1:
		return "about 1 load"
	case low == high:
		return fmt.Sprintf("about %d loads", high)
	case low <= 0 && high == 1:
		return "up to 1 load"
	case low <= 0:
		return fmt.Sprintf("up to %d loads", high)
	default:
		return fmt.Sprintf("%d–%d loads", low, high)
	}
}

func windowText(reason *Reason, location *time.Location) string {
	start := time.Unix(reason.WindowStart, 0).In(location)
	end := time.Unix(reason.WindowEnd, 0).In(location)

	if start.YearDay() != end.YearDay() && end.Format("15:04") != "00:00" {
		return fmt.Sprintf("%s %s–%s %s",
			start.Format("Mon"), start.Format("15:04"), end.Format("Mon"), end.Format("15:04"))
	}

	return fmt.Sprintf("%s %s–%s", start.Format("Mon"), start.Format("15:04"), end.Format("15:04"))
}

func customerText(customers []CustomerShare) string {
	parts := make([]string, 0, len(customers))
	for _, customer := range customers {
		if customer.Name == "" {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s about %d", customer.Name,
			int(math.Round(customer.Expected))))
	}

	return strings.Join(parts, ", ")
}

func bookedText(booked int) string {
	switch booked {
	case 0:
		return "none booked yet"
	case 1:
		return "1 already booked"
	default:
		return fmt.Sprintf("%d already booked", booked)
	}
}

func driversText(drivers int) string {
	switch drivers {
	case 0:
		return "no other driver expected"
	case 1:
		return "1 other driver expected"
	default:
		return fmt.Sprintf("%d other drivers expected", drivers)
	}
}
