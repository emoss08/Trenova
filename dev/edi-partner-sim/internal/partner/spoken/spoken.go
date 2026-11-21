package spoken

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

var kindWords = map[partner.Kind]string{
	partner.KindOriginal:  "Original tender",
	partner.KindChange:    "Change",
	partner.KindReplace:   "Replacement",
	partner.KindCancel:    "Cancellation",
	partner.KindDuplicate: "Exact resend",
}

func Variation(v partner.Variation) string {
	text, ok := kindWords[v.Kind]
	if !ok {
		text = "Tender"
	}
	var parts []string
	if v.Delta.RatePercent != 0 {
		parts = append(parts, fmt.Sprintf("total %+g%%", v.Delta.RatePercent))
	}
	if v.Delta.AddStop {
		parts = append(parts, "a stop added")
	}
	if move := v.Delta.Move; move != nil {
		stop := "the last stop"
		if move.Stop > 0 {
			stop = fmt.Sprintf("stop %d", move.Stop)
		}
		way := "later"
		if move.By < 0 {
			way = "earlier"
		}
		parts = append(parts, fmt.Sprintf("%s moved %s %s", stop, Duration(move.By), way))
	}
	if v.Sparse {
		parts = append(parts, "only the required elements")
	}
	if v.Need != nil {
		parts = append(parts, "a load of "+v.Need.String())
	}
	if len(parts) == 0 {
		return text
	}
	return text + ": " + strings.Join(parts, ", ")
}

func Duration(d time.Duration) string {
	if d < 0 {
		d = -d
	}
	hours, minutes := int(d/time.Hour), int(d%time.Hour/time.Minute)
	switch {
	case hours > 0 && minutes > 0:
		return fmt.Sprintf("%d h %d min", hours, minutes)
	case hours > 0:
		return fmt.Sprintf("%d h", hours)
	case minutes > 0:
		return fmt.Sprintf("%d min", minutes)
	}
	return fmt.Sprintf("%d s", int(d/time.Second))
}

func Dollars(cents int64) string {
	if cents < 0 {
		cents = -cents
	}
	whole := fmt.Sprintf("%d", cents/100)
	for index := len(whole) - 3; index > 0; index -= 3 {
		whole = whole[:index] + "," + whole[index:]
	}
	return fmt.Sprintf("$%s.%02d", whole, cents%100)
}
