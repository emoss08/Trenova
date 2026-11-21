package partner

import (
	"fmt"
	"slices"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/geo"
)

type Profile struct {
	Start       string          `yaml:"start"`
	Days        int             `yaml:"days"`
	Number      Pattern         `yaml:"number"`
	BOL         Pattern         `yaml:"bol"`
	PO          Pattern         `yaml:"po"`
	PickupRef   Pattern         `yaml:"pickupRef"`
	Appointment Pattern         `yaml:"appointment"`
	Payment     []Weighted      `yaml:"payment"`
	Stops       StopPlan        `yaml:"stops"`
	Origins     []PartyDef      `yaml:"origins"`
	BillTo      PartyDef        `yaml:"billTo"`
	Consignees  Consignees      `yaml:"consignees"`
	Equipment   []EquipmentDef  `yaml:"equipment"`
	Weight      Range           `yaml:"weight"`
	Pieces      Range           `yaml:"pieces"`
	Lines       Range           `yaml:"lines"`
	Commodities []CommodityDef  `yaml:"commodities"`
	Hazmat      float64         `yaml:"hazmat"`
	TempMix     float64         `yaml:"temperatureMix"`
	Rate        RatePlan        `yaml:"rate"`
	Window      WindowPlan      `yaml:"window"`
	Zip4        float64         `yaml:"zip4"`
	Contact     float64         `yaml:"contact"`
	Notes       NotePlan        `yaml:"notes"`
	start       time.Time       `yaml:"-"`
	origins     []resolvedParty `yaml:"-"`
	billTo      resolvedParty   `yaml:"-"`
	oneRange    bool            `yaml:"-"`
}

type Pattern struct {
	Prefix string   `yaml:"prefix"`
	Digits int      `yaml:"digits"`
	Chance *float64 `yaml:"chance"`
}

type Weighted struct {
	Code   string `yaml:"code"`
	Weight int    `yaml:"weight"`
}

type StopPlan struct {
	Min         int     `yaml:"min"`
	Max         int     `yaml:"max"`
	ExtraPickup float64 `yaml:"extraPickup"`
}

type PartyDef struct {
	Code   string `yaml:"code"`
	Name   string `yaml:"name"`
	Street string `yaml:"street"`
	City   string `yaml:"city"`
	State  string `yaml:"state"`
	Zip    string `yaml:"zip"`
}

type Consignees struct {
	Code  string   `yaml:"code"`
	Names []string `yaml:"names"`
}

type EquipmentDef struct {
	Code     string `yaml:"code"`
	Weight   int    `yaml:"weight"`
	LengthFt int    `yaml:"lengthFt"`
	Reefer   *Range `yaml:"reefer"`
}

type Range struct {
	Min int `yaml:"min"`
	Max int `yaml:"max"`
}

type CommodityDef struct {
	Description string  `yaml:"description"`
	Code        string  `yaml:"code"`
	Hazmat      *Hazmat `yaml:"hazmat"`
	Temperature *Range  `yaml:"temperature"`
}

type RatePlan struct {
	PerMileCents Range `yaml:"perMileCents"`
	MinimumCents int64 `yaml:"minimumCents"`
	StopOffCents int64 `yaml:"stopOffCents"`
}

type WindowPlan struct {
	LeadDays    Range `yaml:"leadDays"`
	PickupHours Range `yaml:"pickupHours"`
	Hours       int   `yaml:"hours"`
	Mph         int   `yaml:"mph"`
	DwellHours  int   `yaml:"dwellHours"`
}

type NotePlan struct {
	Chance   float64  `yaml:"chance"`
	Load     []string `yaml:"load"`
	Delivery []string `yaml:"delivery"`
}

type resolvedParty struct {
	party Party
	city  geo.City
}

func (p *Profile) validate(check *problems, zone *time.Location) {
	start, err := time.ParseInLocation(time.DateOnly, p.Start, zone)
	if err != nil {
		check.add("load.start", "%q is not a date such as 2026-03-02", p.Start)
	}
	p.start = start
	if p.Days < 1 {
		check.add("load.days", "must be at least 1")
	}
	validatePattern(check, "load.number", p.Number, false)
	validatePattern(check, "load.bol", p.BOL, true)
	validatePattern(check, "load.po", p.PO, true)
	validatePattern(check, "load.pickupRef", p.PickupRef, true)
	validatePattern(check, "load.appointment", p.Appointment, true)
	validateWeighted(check, "load.payment", p.Payment)
	if p.Stops.Min < 2 || p.Stops.Max < p.Stops.Min || p.Stops.Max > 12 {
		check.add("load.stops", "needs 2 <= min <= max <= 12, has %d..%d", p.Stops.Min, p.Stops.Max)
	}
	validateChance(check, "load.stops.extraPickup", p.Stops.ExtraPickup)
	if len(p.Origins) == 0 {
		check.add("load.origins", "lists no origin")
	}
	p.origins = p.origins[:0]
	for index, origin := range p.Origins {
		resolved, ok := resolveParty(check, fmt.Sprintf("load.origins[%d]", index), origin)
		if ok {
			p.origins = append(p.origins, resolved)
		}
	}
	p.billTo, _ = resolveParty(check, "load.billTo", p.BillTo)
	check.required("load.consignees.code", p.Consignees.Code)
	if len(p.Consignees.Names) == 0 {
		check.add("load.consignees.names", "lists no name")
	}
	if len(p.Equipment) == 0 {
		check.add("load.equipment", "lists no equipment")
	}
	for index, equipment := range p.Equipment {
		path := fmt.Sprintf("load.equipment[%d]", index)
		check.required(path+".code", equipment.Code)
		if equipment.Weight < 1 {
			check.add(path+".weight", "must be at least 1")
		}
		if equipment.LengthFt < 20 || equipment.LengthFt > 53 {
			check.add(
				path+".lengthFt",
				"%d is not a trailer length between 20 and 53",
				equipment.LengthFt,
			)
		}
		if equipment.Reefer != nil {
			validateRange(check, path+".reefer", *equipment.Reefer, -20, 70)
		}
	}
	validateRange(check, "load.weight", p.Weight, 100, 80_000)
	validateRange(check, "load.pieces", p.Pieces, 1, 9_999)
	if p.Pieces.Min < p.Stops.Max {
		check.add(
			"load.pieces.min",
			"%d is under load.stops.max: every stop carries a piece",
			p.Pieces.Min,
		)
	}
	validateRange(check, "load.lines", p.Lines, 1, 20)
	if len(p.Commodities) == 0 {
		check.add("load.commodities", "lists no commodity")
	}
	hazardous := false
	for index, commodity := range p.Commodities {
		path := fmt.Sprintf("load.commodities[%d]", index)
		check.required(path+".description", commodity.Description)
		check.required(path+".code", commodity.Code)
		if commodity.Temperature != nil {
			validateRange(check, path+".temperature", *commodity.Temperature, -40, 90)
		}
		if commodity.Hazmat == nil {
			continue
		}
		hazardous = true
		if !unPattern.MatchString(commodity.Hazmat.UN) {
			check.add(
				path+".hazmat.un",
				"%q is not a UN or NA number such as UN1760",
				commodity.Hazmat.UN,
			)
		}
		check.required(path+".hazmat.class", commodity.Hazmat.Class)
		check.required(path+".hazmat.name", commodity.Hazmat.Name)
	}
	validateChance(check, "load.hazmat", p.Hazmat)
	validateChance(check, "load.temperatureMix", p.TempMix)
	if p.TempMix > 0 && !p.chilled() {
		check.add("load.temperatureMix", "is above 0 but no commodity has a temperature")
	}
	if p.Hazmat > 0 && !hazardous {
		check.add("load.hazmat", "is above 0 but no commodity is hazardous")
	}
	validateRange(check, "load.rate.perMileCents", p.Rate.PerMileCents, 50, 2_000)
	if p.Rate.MinimumCents < 0 || p.Rate.StopOffCents < 0 {
		check.add("load.rate", "minimumCents and stopOffCents cannot be negative")
	}
	validateRange(check, "load.window.leadDays", p.Window.LeadDays, 0, 30)
	validateRange(check, "load.window.pickupHours", p.Window.PickupHours, 0, 22)
	if p.Window.Hours < 1 || p.Window.Hours > 24 {
		check.add("load.window.hours", "%d is not between 1 and 24", p.Window.Hours)
	}
	if p.Window.Mph < 20 || p.Window.Mph > 70 {
		check.add("load.window.mph", "%d is not between 20 and 70", p.Window.Mph)
	}
	if p.Window.DwellHours < 0 {
		check.add("load.window.dwellHours", "cannot be negative")
	}
	validateChance(check, "load.zip4", p.Zip4)
	validateChance(check, "load.contact", p.Contact)
	validateChance(check, "load.notes.chance", p.Notes.Chance)
}

func validatePattern(check *problems, path string, pattern Pattern, optional bool) {
	if pattern.Digits < 1 || pattern.Digits > 18 {
		check.add(path+".digits", "%d is not between 1 and 18", pattern.Digits)
	}
	if pattern.Chance != nil {
		if !optional {
			check.add(path+".chance", "is not allowed: this reference is always sent")
		}
		validateChance(check, path+".chance", *pattern.Chance)
	}
}

func validateWeighted(check *problems, path string, options []Weighted) {
	if len(options) == 0 {
		check.add(path, "lists no code")
	}
	for index, option := range options {
		check.required(fmt.Sprintf("%s[%d].code", path, index), option.Code)
		if option.Weight < 1 {
			check.add(fmt.Sprintf("%s[%d].weight", path, index), "must be at least 1")
		}
	}
}

func validateRange(check *problems, path string, value Range, low, high int) {
	if value.Min < low || value.Max > high || value.Min > value.Max {
		check.add(path, "needs %d <= min <= max <= %d, has %d..%d", low, high, value.Min, value.Max)
	}
}

func validateChance(check *problems, path string, value float64) {
	if value < 0 || value > 1 {
		check.add(path, "%v is not a probability between 0 and 1", value)
	}
}

func resolveParty(check *problems, path string, def PartyDef) (resolvedParty, bool) {
	check.required(path+".code", def.Code)
	check.required(path+".name", def.Name)
	check.required(path+".street", def.Street)
	place, ok := geo.Find(def.City, def.State)
	if !ok {
		check.add(path+".city", "%s, %s is not in the simulator's city table", def.City, def.State)
		return resolvedParty{}, false
	}
	zip := def.Zip
	if zip == "" {
		zip = place.Zip
	}
	return resolvedParty{
		party: Party{
			Code:    def.Code,
			Name:    def.Name,
			Street:  def.Street,
			City:    place.Name,
			State:   place.State,
			Zip:     zip,
			Country: "US",
			Zone:    place.Zone,
		},
		city: place,
	}, true
}

func (p *Profile) chilled() bool {
	return slices.ContainsFunc(p.Commodities, func(commodity CommodityDef) bool {
		return commodity.Temperature != nil
	})
}
