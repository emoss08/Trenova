package partner

import (
	"fmt"
	"math"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/geo"
)

type Kind string

const (
	KindOriginal  Kind = "original"
	KindChange    Kind = "change"
	KindReplace   Kind = "replace"
	KindCancel    Kind = "cancel"
	KindDuplicate Kind = "duplicate"
)

var Kinds = []Kind{KindOriginal, KindChange, KindReplace, KindCancel, KindDuplicate}

type Variation struct {
	Kind   Kind      `yaml:"kind"`
	Sparse bool      `yaml:"sparse"`
	Delta  Delta     `yaml:"delta"`
	Need   *LoadNeed `yaml:"load"`
}

type Delta struct {
	RatePercent float64 `yaml:"ratePercent"`
	AddStop     bool    `yaml:"addStop"`
	Move        *Move   `yaml:"move"`
}

type Move struct {
	Stop int           `yaml:"stop"`
	By   time.Duration `yaml:"by"`
}

func (v Variation) String() string {
	text := string(v.Kind)
	if v.Sparse {
		text += " sparse"
	}
	if v.Delta.RatePercent != 0 {
		text += fmt.Sprintf(" rate %+g%%", v.Delta.RatePercent)
	}
	if v.Delta.AddStop {
		text += " add-stop"
	}
	if v.Delta.Move != nil {
		text += fmt.Sprintf(" move stop %d by %s", v.Delta.Move.Stop, v.Delta.Move.By)
	}
	if v.Need != nil {
		text += " (" + v.Need.String() + ")"
	}
	return text
}

func (v Variation) validate(check *problems, path string) {
	known := make([]string, 0, len(Kinds))
	for _, kind := range Kinds {
		known = append(known, string(kind))
	}
	check.oneOf(path+".kind", string(v.Kind), known...)
	if v.Delta.RatePercent <= -100 {
		check.add(
			path+".delta.ratePercent",
			"%v would make the rate zero or negative",
			v.Delta.RatePercent,
		)
	}
	for _, problem := range v.Need.Problems() {
		check.add(path+".load", "%s", problem)
	}
	if v.Delta.Move != nil && v.Delta.Move.Stop < 0 {
		check.add(
			path+".delta.move.stop",
			"%d is not a stop sequence (0 is the last stop)",
			v.Delta.Move.Stop,
		)
	}
}

func (p *Profile) apply(load *Load, delta Delta) (*Load, error) {
	varied := load.Clone()
	if delta.AddStop {
		p.addStop(varied)
	}
	if delta.Move != nil {
		index := len(varied.Stops) - 1
		if delta.Move.Stop > 0 {
			index = delta.Move.Stop - 1
		}
		if index < 0 || index >= len(varied.Stops) {
			return nil, fmt.Errorf("load %s has no stop %d to move", varied.Number, delta.Move.Stop)
		}
		window := &varied.Stops[index].Window
		window.Start = window.Start.Add(delta.Move.By)
		window.End = window.End.Add(delta.Move.By)
	}
	if delta.RatePercent != 0 {
		varied.TotalCents = int64(
			math.Round(float64(varied.TotalCents) * (100 + delta.RatePercent) / 100),
		)
	}
	return varied, nil
}

func (p *Profile) addStop(load *Load) {
	rng := rand.New(rand.NewPCG(load.Seed, hash64("add-stop "+load.Number)))
	taken := map[string]bool{}
	for index := range load.Stops {
		taken[load.Stops[index].Party.City] = true
	}
	cityIndex := rng.IntN(len(geo.Cities))
	for taken[geo.Cities[cityIndex].Name] {
		cityIndex = (cityIndex + 1) % len(geo.Cities)
	}
	g := &generator{profile: p, rng: rng, zone: load.TenderedAt.Location()}
	added := g.consignee(rng.IntN(len(p.Consignees.Names)), cityIndex)
	last := &load.Stops[len(load.Stops)-1]
	stop := Stop{
		Sequence:     last.Sequence + 1,
		Kind:         StopDelivery,
		Party:        added.party,
		Appointment:  g.reference(p.Appointment),
		Temperatures: slices.Clone(last.Temperatures),
	}
	if last.Kind == StopDelivery && last.Pieces > 1 {
		stop.Weight = last.Weight / 2
		stop.Pieces = last.Pieces / 2
		last.Weight -= stop.Weight
		last.Pieces -= stop.Pieces
	}
	from, _ := geo.Find(last.Party.City, last.Party.State)
	leg := geo.RoadMiles(from, added.city)
	drive := time.Duration(float64(leg)/float64(p.Window.Mph)*float64(time.Hour)) +
		time.Duration(p.Window.DwellHours)*time.Hour
	start := workingHour(last.Window.Start.Add(drive).Truncate(time.Hour).Add(time.Hour))
	stop.Window = Window{Start: start, End: start.Add(time.Duration(p.Window.Hours) * time.Hour)}
	load.Stops = append(load.Stops, stop)
	load.Miles += leg
	load.TotalCents += p.Rate.StopOffCents
}
