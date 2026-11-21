package conform

import (
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const varietySeeds = 400

func TestTrenovasTimesCarrySecondsZonesShiftsAndDateRolls(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	base := time.Date(2026, time.March, 4, 9, 0, 0, 0, chicago)
	shift := time.Date(2026, time.March, 8, 8, 0, 0, 0, time.UTC)
	seen := map[string]int{}
	for seed := uint64(1); seed <= varietySeeds; seed++ {
		v := newVariety("manufacturer", nil, seed)
		at := v.instant(base, chicago)
		local := at.In(chicago)
		if at.Second() != 0 {
			seen["seconds"]++
		}
		if local.Hour() == 0 && local.Minute() == 0 {
			seen["midnight"]++
		}
		if local.Hour() == 23 && local.Minute() == 59 {
			seen["last minute"]++
		}
		if absolute(at.Sub(shift)) <= time.Hour {
			seen["daylight saving shift"]++
		}
		if at.UTC().Day() != local.Day() {
			seen["UTC on another day"]++
		}
		switch v.zone("America/New_York", "America/Chicago") {
		case "":
			seen["held in UTC"]++
		case "America/Chicago":
			seen["the fallback zone"]++
		}
	}
	for _, want := range []string{
		"seconds", "midnight", "last minute", "daylight saving shift", "UTC on another day",
		"held in UTC", "the fallback zone",
	} {
		if seen[want] == 0 {
			t.Errorf("no seed of %d gave a time with %s: %v", varietySeeds, want, seen)
		}
	}
	var plain *variety
	if plain.instant(base, chicago) != base || plain.zone("A", "B") != "A" {
		t.Error("no variety must leave times and zones as they are")
	}
}

func TestTheNearestShiftIsTheChangeOfClock(t *testing.T) {
	t.Parallel()

	chicago, err := time.LoadLocation("America/Chicago")
	if err != nil {
		t.Fatal(err)
	}
	moment, ok := nearestShift(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC), chicago)
	if !ok || !moment.Equal(time.Date(2026, time.March, 8, 8, 0, 0, 0, time.UTC)) {
		t.Errorf("the spring shift in Chicago: %s %v", moment, ok)
	}
	phoenix, err := time.LoadLocation("America/Phoenix")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok = nearestShift(time.Date(2026, time.March, 1, 12, 0, 0, 0, time.UTC), phoenix); ok {
		t.Error("Phoenix keeps one clock all year")
	}
}

func TestTextReachesAndPassesTheGuidesWidthAndLeavesASCII(t *testing.T) {
	t.Parallel()

	guides, err := oracle.LoadGuides("../../partners/manufacturer/guide")
	if err != nil {
		t.Fatal(err)
	}
	const name = "KESTREL VALVE WORKS ACCOUNTS PAYABLE"
	seen := map[string]int{}
	for seed := uint64(1); seed <= varietySeeds; seed++ {
		text := newVariety("manufacturer", guides, seed).text(name, invoiceName)
		length := utf8.RuneCountInString(text)
		switch {
		case length == 60:
			seen["at the width"]++
		case length > 60:
			seen["past the width"]++
		case text == strings.ToLower(name):
			seen["lowercase"]++
		case strings.ContainsAny(text, "*~|^>:"):
			seen["a separator"]++
		case !isASCII(text):
			seen["non-ASCII"]++
		}
	}
	for _, want := range []string{
		"at the width", "past the width", "lowercase", "a separator", "non-ASCII",
	} {
		if seen[want] == 0 {
			t.Errorf("no seed of %d gave text %s: %v", varietySeeds, want, seen)
		}
	}
}

func TestInvoiceLinesAlwaysAddUpToTheTotal(t *testing.T) {
	t.Parallel()

	load := &partner.Load{RateCents: 250}
	extra := 0
	for seed := uint64(1); seed <= varietySeeds; seed++ {
		total := cents(int64(45_000 + seed*137))
		lines := invoiceLines(load, total, newVariety("grocer", nil, seed))
		sum := int64(0)
		for _, line := range lines {
			amount := centsOf(line.Amount.Amount)
			if amount <= 0 {
				t.Fatalf("seed %d: a line of %s", seed, line.Amount.Amount)
			}
			sum += amount
		}
		if sum != centsOf(total.Amount) {
			t.Fatalf("seed %d: the lines add to %d, the total is %s", seed, sum, total.Amount)
		}
		extra += len(lines) - 1
	}
	if extra == 0 {
		t.Error("no invoice carried a charge besides the linehaul")
	}
}

func isASCII(text string) bool {
	for _, r := range text {
		if r > 127 {
			return false
		}
	}
	return true
}
