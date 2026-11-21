package partner

import (
	"fmt"
	"hash/fnv"
	"math/rand/v2"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/geo"
)

var unPattern = regexp.MustCompile(`^(UN|NA)[0-9]{4}$`)

type generator struct {
	profile *Profile
	rng     *rand.Rand
	zone    *time.Location
	cold    *TempRange
}

func (p *Profile) Generate(name string, zone *time.Location, seed uint64) *Load {
	g := &generator{
		profile: p,
		rng:     rand.New(rand.NewPCG(seed, hash64(name))),
		zone:    zone,
	}
	return g.load(seed)
}

func (g *generator) load(seed uint64) *Load {
	p := g.profile
	day := p.start.AddDate(0, 0, g.rng.IntN(p.Days))
	tendered := time.Date(
		day.Year(), day.Month(), day.Day(),
		6+g.rng.IntN(11), 5*g.rng.IntN(12), 0, 0, g.zone,
	)
	load := &Load{
		Seed:       seed,
		Number:     p.Number.Prefix + sequenced(seed, p.Number.Digits),
		TenderedAt: tendered,
		Payment:    g.weighted(p.Payment),
		BOL:        g.reference(p.BOL),
		PO:         g.reference(p.PO),
		BillTo:     p.billTo.party,
		Equipment:  g.equipment(),
	}
	if g.chance(p.Notes.Chance) && len(p.Notes.Load) > 0 {
		load.Note = g.pick(p.Notes.Load)
	}
	origins, deliveries := g.places()
	g.stops(load, origins, deliveries)
	g.freight(load)
	g.schedule(load, origins, deliveries)
	g.rate(load)
	temperatures(load)
	return load
}

func (g *generator) equipment() Equipment {
	options := g.profile.Equipment
	weights := make([]int, len(options))
	for index := range options {
		weights[index] = options[index].Weight
	}
	chosen := options[g.weightedIndex(weights)]
	equipment := Equipment{Code: chosen.Code, LengthFt: chosen.LengthFt}
	if chosen.Reefer != nil {
		equipment.Reefer = true
		equipment.TempF = g.between(*chosen.Reefer)
	}
	return equipment
}

func (g *generator) places() ([]resolvedParty, []resolvedParty) {
	p := g.profile
	count := p.Stops.Min + g.rng.IntN(p.Stops.Max-p.Stops.Min+1)
	pickups := 1
	if count >= 3 && len(p.origins) > 1 && g.chance(p.Stops.ExtraPickup) {
		pickups = 2
	}
	order := g.rng.Perm(len(p.origins))
	origins := make([]resolvedParty, 0, pickups)
	for _, index := range order[:pickups] {
		origins = append(origins, p.origins[index])
	}
	taken := map[string]bool{}
	for _, origin := range origins {
		taken[origin.city.Name] = true
	}
	deliveries := make([]resolvedParty, 0, count-pickups)
	for len(deliveries) < count-pickups {
		cityIndex := g.rng.IntN(len(geo.Cities))
		place := geo.Cities[cityIndex]
		if taken[place.Name] {
			continue
		}
		taken[place.Name] = true
		nameIndex := g.rng.IntN(len(p.Consignees.Names))
		deliveries = append(deliveries, g.consignee(nameIndex, cityIndex))
	}
	return origins, nearestFirst(origins[len(origins)-1].city, deliveries)
}

func (g *generator) consignee(nameIndex, cityIndex int) resolvedParty {
	p := g.profile
	place := geo.Cities[cityIndex]
	code := fmt.Sprintf("%s%02d%02d", p.Consignees.Code, nameIndex, cityIndex)
	fixed := rand.New(rand.NewPCG(hash64(code), 0))
	street := fmt.Sprintf(
		"%d %s %s",
		100+fixed.IntN(9_800),
		geo.StreetNames[fixed.IntN(len(geo.StreetNames))],
		geo.StreetKinds[fixed.IntN(len(geo.StreetKinds))],
	)
	return resolvedParty{
		party: Party{
			Code:    code,
			Name:    p.Consignees.Names[nameIndex] + " " + place.Name,
			Street:  street,
			City:    place.Name,
			State:   place.State,
			Zip:     place.Zip,
			Country: "US",
			Zone:    place.Zone,
		},
		city: place,
	}
}

func nearestFirst(from geo.City, places []resolvedParty) []resolvedParty {
	remaining := slices.Clone(places)
	ordered := make([]resolvedParty, 0, len(places))
	current := from
	for len(remaining) > 0 {
		best := 0
		for index := range remaining {
			if geo.RoadMiles(
				current,
				remaining[index].city,
			) < geo.RoadMiles(
				current,
				remaining[best].city,
			) {
				best = index
			}
		}
		ordered = append(ordered, remaining[best])
		current = remaining[best].city
		remaining = slices.Delete(remaining, best, best+1)
	}
	return ordered
}

func (g *generator) stops(load *Load, origins, deliveries []resolvedParty) {
	p := g.profile
	sequence := 0
	for _, origin := range origins {
		sequence++
		stop := Stop{
			Sequence:  sequence,
			Kind:      StopPickup,
			Party:     origin.party,
			PickupRef: g.reference(p.PickupRef),
		}
		g.contact(&stop, "SHIPPING", origin.city)
		load.Stops = append(load.Stops, stop)
	}
	for _, delivery := range deliveries {
		sequence++
		stop := Stop{
			Sequence:    sequence,
			Kind:        StopDelivery,
			Party:       delivery.party,
			Appointment: g.reference(p.Appointment),
		}
		if g.chance(p.Zip4) {
			stop.Party.Zip += fmt.Sprintf("%04d", 1+g.rng.IntN(9_998))
		}
		if g.chance(p.Notes.Chance) && len(p.Notes.Delivery) > 0 {
			stop.Note = g.pick(p.Notes.Delivery)
		}
		g.contact(&stop, "RECEIVING", delivery.city)
		load.Stops = append(load.Stops, stop)
	}
}

func (g *generator) contact(stop *Stop, name string, place geo.City) {
	if !g.chance(g.profile.Contact) {
		return
	}
	stop.Party.Contact = name
	stop.Party.Phone = place.Area + "555" + fmt.Sprintf("%04d", g.rng.IntN(10_000))
}

func (g *generator) freight(load *Load) {
	p := g.profile
	weight := roundTo(g.between(p.Weight), 10)
	pieces := g.between(p.Pieces)
	pickups, deliveries := splitKinds(load.Stops)
	pickupWeights := split(weight, len(pickups), g.rng)
	pickupPieces := split(pieces, len(pickups), g.rng)
	hazmatLine := g.chance(p.Hazmat)
	g.chill()
	for index, stop := range pickups {
		stop.Weight = pickupWeights[index]
		stop.Pieces = pickupPieces[index]
		g.lines(stop, hazmatLine && index == 0)
	}
	deliveryWeights := split(weight, len(deliveries), g.rng)
	deliveryPieces := split(pieces, len(deliveries), g.rng)
	for index, stop := range deliveries {
		stop.Weight = deliveryWeights[index]
		stop.Pieces = deliveryPieces[index]
	}
}

func (g *generator) lines(stop *Stop, hazardous bool) {
	p := g.profile
	count := min(g.between(p.Lines), stop.Pieces)
	weights := split(stop.Weight, count, g.rng)
	pieces := split(stop.Pieces, count, g.rng)
	plain, dangerous := splitCommodities(g.compatible())
	for index := range count {
		commodity := g.pickCommodity(plain, dangerous)
		if hazardous && index == count-1 && len(dangerous) > 0 {
			commodity = dangerous[g.rng.IntN(len(dangerous))]
		}
		line := Line{
			Number:      index + 1,
			Description: commodity.Description,
			Code:        commodity.Code,
			Weight:      weights[index],
			Pieces:      pieces[index],
		}
		if commodity.Hazmat != nil {
			hazmat := *commodity.Hazmat
			line.Hazmat = &hazmat
		}
		if commodity.Temperature != nil {
			line.Temperature = &TempRange{
				MinF: commodity.Temperature.Min,
				MaxF: commodity.Temperature.Max,
			}
		}
		stop.Lines = append(stop.Lines, line)
	}
}

func (g *generator) chill() {
	p := g.profile
	if !p.chilled() {
		return
	}
	var chilled []CommodityDef
	for _, commodity := range p.Commodities {
		if commodity.Temperature != nil {
			chilled = append(chilled, commodity)
		}
	}
	anchor := chilled[g.rng.IntN(len(chilled))].Temperature
	if !g.chance(p.TempMix) {
		g.cold = &TempRange{MinF: anchor.Min, MaxF: anchor.Max}
	}
}

func (g *generator) compatible() []CommodityDef {
	if g.cold == nil {
		return g.profile.Commodities
	}
	middle := (g.cold.MinF + g.cold.MaxF) / 2
	var out []CommodityDef
	for _, commodity := range g.profile.Commodities {
		held := TempRange{MinF: middle, MaxF: middle}
		if !g.profile.oneRange {
			held = *g.cold
		}
		if commodity.Temperature == nil || held.overlaps(TempRange{
			MinF: commodity.Temperature.Min,
			MaxF: commodity.Temperature.Max,
		}) {
			out = append(out, commodity)
		}
	}
	return out
}

func temperatures(load *Load) {
	var lines []TempRange
	pickups, deliveries := splitKinds(load.Stops)
	for _, stop := range pickups {
		for _, line := range stop.Lines {
			if line.Temperature != nil {
				lines = append(lines, *line.Temperature)
				stop.Temperatures = appendDistinct(stop.Temperatures, *line.Temperature)
			}
		}
	}
	if len(lines) == 0 {
		return
	}
	for index, stop := range deliveries {
		for line := index; line < len(lines); line += len(deliveries) {
			stop.Temperatures = appendDistinct(stop.Temperatures, lines[line])
		}
		if len(stop.Temperatures) == 0 {
			stop.Temperatures = []TempRange{lines[index%len(lines)]}
		}
	}
}

func appendDistinct(ranges []TempRange, value TempRange) []TempRange {
	if slices.Contains(ranges, value) {
		return ranges
	}
	return append(ranges, value)
}

func (g *generator) pickCommodity(plain, dangerous []CommodityDef) CommodityDef {
	if len(plain) > 0 {
		return plain[g.rng.IntN(len(plain))]
	}
	return dangerous[g.rng.IntN(len(dangerous))]
}

func (g *generator) schedule(load *Load, origins, deliveries []resolvedParty) {
	p := g.profile
	places := slices.Concat(origins, deliveries)
	lead := g.between(p.Window.LeadDays)
	if lead == 0 && load.TenderedAt.Hour() >= 12 {
		lead = 1
	}
	day := load.TenderedAt.AddDate(0, 0, lead)
	start := time.Date(
		day.Year(),
		day.Month(),
		day.Day(),
		g.between(p.Window.PickupHours),
		0,
		0,
		0,
		g.zone,
	)
	length := time.Duration(p.Window.Hours) * time.Hour
	miles := 0
	for index := range load.Stops {
		if index > 0 {
			leg := geo.RoadMiles(places[index-1].city, places[index].city)
			miles += leg
			drive := time.Duration(float64(leg)/float64(p.Window.Mph)*float64(time.Hour)) +
				time.Duration(p.Window.DwellHours)*time.Hour
			start = workingHour(
				load.Stops[index-1].Window.Start.Add(drive).Truncate(time.Hour).Add(time.Hour),
			)
		}
		load.Stops[index].Window = Window{Start: start, End: start.Add(length)}
	}
	load.Miles = miles
}

func workingHour(at time.Time) time.Time {
	switch {
	case at.Hour() < 6:
		return time.Date(at.Year(), at.Month(), at.Day(), 7, 0, 0, 0, at.Location())
	case at.Hour() >= 18:
		next := at.AddDate(0, 0, 1)
		return time.Date(next.Year(), next.Month(), next.Day(), 7, 0, 0, 0, at.Location())
	default:
		return at
	}
}

func (g *generator) rate(load *Load) {
	p := g.profile
	load.RateCents = int64(g.between(p.Rate.PerMileCents))
	total := load.RateCents * int64(load.Miles)
	total = max(total, p.Rate.MinimumCents)
	total += p.Rate.StopOffCents * int64(max(len(load.Stops)-2, 0))
	load.TotalCents = total
}

func (g *generator) reference(pattern Pattern) string {
	if pattern.Chance != nil && !g.chance(*pattern.Chance) {
		return ""
	}
	return pattern.Prefix + g.digits(pattern.Digits)
}

func (g *generator) digits(count int) string {
	out := make([]byte, count)
	for index := range out {
		out[index] = byte('0' + g.rng.IntN(10))
	}
	if count > 1 && out[0] == '0' {
		out[0] = '1'
	}
	return string(out)
}

func (g *generator) weighted(options []Weighted) string {
	weights := make([]int, len(options))
	for index := range options {
		weights[index] = options[index].Weight
	}
	return options[g.weightedIndex(weights)].Code
}

func (g *generator) weightedIndex(weights []int) int {
	total := 0
	for _, weight := range weights {
		total += weight
	}
	roll := g.rng.IntN(total)
	for index, weight := range weights {
		if roll < weight {
			return index
		}
		roll -= weight
	}
	return len(weights) - 1
}

func (g *generator) chance(probability float64) bool {
	return probability > 0 && g.rng.Float64() < probability
}

func (g *generator) between(value Range) int {
	return value.Min + g.rng.IntN(value.Max-value.Min+1)
}

func (g *generator) pick(options []string) string {
	return options[g.rng.IntN(len(options))]
}

func sequenced(seed uint64, digits int) string {
	modulus := uint64(1)
	for range digits {
		modulus *= 10
	}
	floor := modulus / 10
	value := floor + seed%(modulus-floor)
	return strconv.FormatUint(value, 10)
}

func split(total, parts int, rng *rand.Rand) []int {
	if parts <= 1 {
		return []int{total}
	}
	shares := make([]int, parts)
	remaining := total
	for index := range parts - 1 {
		left := parts - index
		even := remaining / left
		low := max(even/2, 1)
		high := max(even+even/2, low)
		share := min(low+rng.IntN(high-low+1), remaining-(left-1))
		shares[index] = max(share, 1)
		remaining -= shares[index]
	}
	shares[parts-1] = remaining
	return shares
}

func splitKinds(stops []Stop) ([]*Stop, []*Stop) {
	var pickups, deliveries []*Stop
	for index := range stops {
		if stops[index].Kind == StopPickup {
			pickups = append(pickups, &stops[index])
		} else {
			deliveries = append(deliveries, &stops[index])
		}
	}
	return pickups, deliveries
}

func splitCommodities(all []CommodityDef) ([]CommodityDef, []CommodityDef) {
	var plain, dangerous []CommodityDef
	for _, commodity := range all {
		if commodity.Hazmat == nil {
			plain = append(plain, commodity)
		} else {
			dangerous = append(dangerous, commodity)
		}
	}
	return plain, dangerous
}

func roundTo(value, step int) int {
	return max(value/step*step, step)
}

func hash64(text string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(text))
	return h.Sum64()
}
