package conform

import (
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/oracle"
)

const (
	trenovaTextMax = 100
	dstSearchDays  = 200
	edgeOneIn      = 5
)

var separatorMarks = []string{"*", "~", "|", "^", ">", ":"}

var fillerWords = []string{
	"NORTH", "INDUSTRIAL", "PARK", "DISTRIBUTION", "CENTER", "ANNEX", "BUILDING", "DOCK",
}

var (
	accentedCities = []string{
		"Cañon City", "Española", "La Cañada Flintridge", "Doña Ana", "Peñasco",
		"Coeur d'Alene", "Mayagüez", "Bayamón",
	}
	accentedNames = []string{
		"Société Générale de Fret", "Müller & Söhne Logística", "Peña Hermanos Distribución",
		"Ñandú Imports", "José Martí Café Supply", "Øresund Trading Co.",
	}
	accentedStreets = []string{
		"1200 Calle Peñuelas", "45 Rue Saint-Jérôme", "8 Straße der Einheit",
		"77 Avenida Ñuñoa, Suite 4",
	}
	accentedNotes = []string{
		"Sin camión disponible — lo siento", "Pas de remorque frigorifique à Joliet",
		"Détention – 2½ h @ $75/h", "Fuel surcharge (4.5 %) · March",
	}
)

type field struct {
	set      string
	segment  string
	position int
	fallback int
	accented []string
}

var (
	shipmentRef    = field{statusSet, "L11", 1, 30, nil}
	statusCity     = field{statusSet, "MS1", 1, 30, accentedCities}
	invoiceName    = field{invoiceSet, "N1", 2, 60, accentedNames}
	invoiceStreet  = field{invoiceSet, "N3", 1, 55, accentedStreets}
	invoiceSuite   = field{invoiceSet, "N3", 2, 55, accentedStreets}
	invoiceCity    = field{invoiceSet, "N4", 1, 30, accentedCities}
	invoiceLineFor = field{invoiceSet, "L5", 2, 50, accentedNotes}
	declineReason  = field{responseSet, "N9", 2, 30, accentedNotes}
	tenderName     = field{tenderSet, "N1", 2, 60, accentedNames}
	tenderStreet   = field{tenderSet, "N3", 1, 55, accentedStreets}
	tenderCity     = field{tenderSet, "N4", 1, 30, accentedCities}
)

type variety struct {
	rng    *rand.Rand
	guides *oracle.GuideSet
}

func newVariety(name string, guides *oracle.GuideSet, seed uint64) *variety {
	return &variety{
		rng: rand.New( //nolint:gosec // a seeded generator: the same seed must replay the same case
			rand.NewPCG(seed, hash64("trenova writes "+name)),
		),
		guides: guides,
	}
}

func (v *variety) chance(oneIn int) bool {
	return v != nil && v.rng.IntN(oneIn) == 0
}

func (v *variety) instant(base time.Time, zone *time.Location) time.Time {
	if v == nil {
		return base
	}
	at := base.Add(time.Duration(v.rng.IntN(90))*time.Minute +
		time.Duration(v.rng.IntN(60))*time.Second)
	if !v.chance(edgeOneIn) {
		return at
	}
	local := at.In(zone)
	midnight := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, zone)
	second := time.Duration(v.rng.IntN(60)) * time.Second
	switch v.rng.IntN(4) {
	case 0:
		return midnight.Add(second)
	case 1:
		return midnight.Add(24*time.Hour - time.Minute + second)
	case 2:
		if moment, ok := nearestShift(at, zone); ok {
			offsets := []time.Duration{-time.Second, 0, second, 30*time.Minute + second}
			return moment.Add(offsets[v.rng.IntN(len(offsets))])
		}
	}
	return v.dateRoll(local, midnight)
}

func (v *variety) dateRoll(local, midnight time.Time) time.Time {
	_, offset := local.Zone()
	span := time.Duration(offset) * time.Second
	switch {
	case span < 0:
		return midnight.Add(24*time.Hour - time.Duration(1+v.rng.Int64N(int64(-span))))
	case span > 0:
		return midnight.Add(time.Duration(v.rng.Int64N(int64(span))))
	}
	return midnight
}

func nearestShift(around time.Time, zone *time.Location) (time.Time, bool) {
	var best time.Time
	found := false
	_, before := around.AddDate(0, 0, -dstSearchDays).In(zone).Zone()
	for day := -dstSearchDays + 1; day <= dstSearchDays; day++ {
		at := around.AddDate(0, 0, day)
		if _, offset := at.In(zone).Zone(); offset != before {
			moment := shiftWithin(at.Add(-24*time.Hour), at, zone)
			if !found || absolute(moment.Sub(around)) < absolute(best.Sub(around)) {
				best, found = moment, true
			}
			before = offset
		}
	}
	return best, found
}

func shiftWithin(low, high time.Time, zone *time.Location) time.Time {
	from, to := low.Unix(), high.Unix()
	_, first := time.Unix(from, 0).In(zone).Zone()
	for to-from > 1 {
		middle := from + (to-from)/2
		if _, offset := time.Unix(middle, 0).In(zone).Zone(); offset == first {
			from = middle
		} else {
			to = middle
		}
	}
	return time.Unix(to, 0).UTC()
}

func absolute(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func (v *variety) zone(own, fallback string) string {
	if v == nil {
		return own
	}
	switch v.rng.IntN(20) {
	case 0:
		return ""
	case 1, 2:
		return fallback
	}
	return own
}

func (v *variety) text(base string, f field) string {
	if v == nil || base == "" {
		return base
	}
	roll := v.rng.IntN(100)
	switch {
	case roll < 60:
		return base
	case roll < 68:
		return strings.ToLower(base)
	case roll < 74:
		return titleCase(base)
	case roll < 82 && len(f.accented) > 0:
		return f.accented[v.rng.IntN(len(f.accented))]
	case roll < 86:
		return marked(base, separatorMarks[v.rng.IntN(len(separatorMarks))])
	case roll < 93:
		return fit(base, v.limit(f))
	}
	return fit(base, min(v.limit(f)+1+v.rng.IntN(20), trenovaTextMax))
}

func (v *variety) limit(f field) int {
	if limit := v.guides.ElementMax(f.set, f.segment, f.position); limit > 0 {
		return limit
	}
	return f.fallback
}

func titleCase(text string) string {
	words := strings.Fields(strings.ToLower(text))
	for index, word := range words {
		runes := []rune(word)
		runes[0] = []rune(strings.ToUpper(string(runes[0])))[0]
		words[index] = string(runes)
	}
	return strings.Join(words, " ")
}

func marked(text, mark string) string {
	if strings.Contains(text, " ") {
		return strings.Replace(text, " ", mark, 1)
	}
	return text + mark
}

func fit(text string, length int) string {
	runes := []rune(text)
	for index := 0; len(runes) < length; index++ {
		runes = append(runes, ' ')
		runes = append(runes, []rune(fillerWords[index%len(fillerWords)])...)
	}
	runes = runes[:length]
	if runes[length-1] == ' ' {
		runes[length-1] = 'X'
	}
	return string(runes)
}

type charge struct {
	code        string
	description string
}

var extraCharges = []charge{
	{"FUE", "Fuel surcharge"}, {"SOC", "Stop-off charges"}, {"DET", "Detention"},
	{"LUM", "Lumper"},
}

func (v *variety) charges(totalCents int64) ([]charge, []int64) {
	if v == nil || totalCents < 400 {
		return nil, nil
	}
	count := v.rng.IntN(3)
	picked := slices.Clone(extraCharges)
	v.rng.Shuffle(len(picked), func(i, j int) { picked[i], picked[j] = picked[j], picked[i] })
	amounts := make([]int64, 0, count)
	left := totalCents
	for range count {
		amount := 1 + v.rng.Int64N(max(totalCents/8, 2))
		if left-amount < totalCents/2 {
			break
		}
		amounts = append(amounts, amount)
		left -= amount
	}
	return picked[:len(amounts)], amounts
}

func (v *variety) tender(document loadTender) loadTender {
	if v == nil {
		return document
	}
	if document.Total != nil {
		varied := cents(centsOf(document.Total.Amount) + 1 + v.rng.Int64N(99))
		document.Total, document.FreightCharge = &varied, &varied
	}
	if v.chance(4) {
		document.PONumber = ""
	}
	stops := slices.Clone(document.Stops)
	for index := range stops {
		stop := &stops[index]
		stop.Name = v.text(stop.Name, tenderName)
		if stop.Location != nil {
			located := *stop.Location
			located.Label = stop.Name
			if v.chance(5) {
				located.Code, stop.Code = "", ""
			}
			stop.Location = &located
		}
		if stop.Address != nil {
			moved := *stop.Address
			moved.Line1 = v.text(moved.Line1, tenderStreet)
			moved.City = v.text(moved.City, tenderCity)
			stop.Address = &moved
		}
		stop.Window = v.window(stop.Window)
	}
	document.Stops = stops
	return document
}

func (v *variety) window(w window) window {
	start, err := time.Parse(time.RFC3339, w.Start.At)
	if err != nil {
		return w
	}
	zone := location(w.Start.Zone)
	moved := v.instant(start, zone)
	w.Start = at(moved, w.Start.Zone)
	if w.End == nil {
		return w
	}
	end, err := time.Parse(time.RFC3339, w.End.At)
	if err != nil || !end.After(start) {
		return w
	}
	closed := at(moved.Add(end.Sub(start)+time.Duration(v.rng.IntN(60))*time.Second),
		w.End.Zone)
	w.End = &closed
	return w
}

func (v *variety) shipment(tender *readTender) *readTender {
	if v == nil {
		return tender
	}
	built := *tender
	built.BOL = v.text(tender.BOL, shipmentRef)
	if v.chance(20) {
		built.BOL = ""
	}
	built.PONumber = v.text(strings.TrimSpace(tender.PONumber), shipmentRef)
	if v.chance(8) {
		built.Weight, built.Pieces = nil, nil
	}
	return &built
}
