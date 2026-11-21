package partner

import (
	"errors"
	"fmt"
	"strings"
)

const weightStep = 10

var errNeed = errors.New("no load the partner sends meets the need")

type LoadNeed struct {
	MinWeight int   `yaml:"minWeight" json:"minWeight,omitempty"`
	MaxWeight int   `yaml:"maxWeight" json:"maxWeight,omitempty"`
	Stops     int   `yaml:"stops"     json:"stops,omitempty"`
	Hazmat    *bool `yaml:"hazmat"    json:"hazmat,omitempty"`
	OneRange  bool  `yaml:"oneRange"  json:"oneRange,omitempty"`
}

func (n *LoadNeed) String() string {
	if n == nil {
		return ""
	}
	var parts []string
	switch {
	case n.MinWeight > 0 && n.MaxWeight > 0:
		parts = append(parts, fmt.Sprintf("%d-%d lb", n.MinWeight, n.MaxWeight))
	case n.MinWeight > 0:
		parts = append(parts, fmt.Sprintf("at least %d lb", n.MinWeight))
	case n.MaxWeight > 0:
		parts = append(parts, fmt.Sprintf("at most %d lb", n.MaxWeight))
	}
	if n.Stops > 0 {
		parts = append(parts, fmt.Sprintf("%d stops", n.Stops))
	}
	if n.Hazmat != nil {
		parts = append(parts, map[bool]string{true: "hazmat", false: "no hazmat"}[*n.Hazmat])
	}
	if n.OneRange {
		parts = append(parts, "one temperature range")
	}
	return strings.Join(parts, ", ")
}

func (n *LoadNeed) Problems() []string {
	if n == nil {
		return nil
	}
	var out []string
	if n.MinWeight < 0 || n.MaxWeight < 0 {
		out = append(out, "minWeight and maxWeight cannot be negative")
	}
	if n.MaxWeight > 0 && n.MinWeight > n.MaxWeight {
		out = append(out, fmt.Sprintf("minWeight %d is over maxWeight %d", n.MinWeight,
			n.MaxWeight))
	}
	if n.Stops != 0 && (n.Stops < 2 || n.Stops > 12) {
		out = append(out, fmt.Sprintf("stops %d is not between 2 and 12", n.Stops))
	}
	return out
}

func (p *Profile) Narrow(need *LoadNeed) (*Profile, error) {
	if need == nil {
		return p, nil
	}
	if problems := need.Problems(); len(problems) > 0 {
		return nil, fmt.Errorf("%w: %s", errNeed, strings.Join(problems, "; "))
	}
	narrowed := *p
	weight := Range{Min: max(p.Weight.Min, need.MinWeight), Max: p.Weight.Max}
	if need.MaxWeight > 0 {
		weight.Max = min(weight.Max, need.MaxWeight)
	}
	weight.Min = (weight.Min + weightStep - 1) / weightStep * weightStep
	weight.Max = weight.Max / weightStep * weightStep
	if weight.Min > weight.Max {
		return nil, fmt.Errorf("%w: its loads weigh %d to %d lb, the need is %s", errNeed,
			p.Weight.Min, p.Weight.Max, need)
	}
	narrowed.Weight = weight
	if need.Stops > 0 {
		if need.Stops < p.Stops.Min || need.Stops > p.Stops.Max {
			return nil, fmt.Errorf("%w: its loads have %d to %d stops, the need is %d", errNeed,
				p.Stops.Min, p.Stops.Max, need.Stops)
		}
		narrowed.Stops = StopPlan{
			Min:         need.Stops,
			Max:         need.Stops,
			ExtraPickup: p.Stops.ExtraPickup,
		}
	}
	if need.Hazmat != nil {
		if err := narrowed.narrowHazmat(*need.Hazmat); err != nil {
			return nil, err
		}
	}
	if need.OneRange {
		if !p.chilled() {
			return nil, fmt.Errorf("%w: none of its commodities has a temperature", errNeed)
		}
		narrowed.TempMix = 0
		narrowed.oneRange = true
	}
	return &narrowed, nil
}

func (p *Profile) narrowHazmat(hazardous bool) error {
	plain, dangerous := splitCommodities(p.Commodities)
	if hazardous {
		if len(dangerous) == 0 {
			return fmt.Errorf("%w: none of its commodities is hazardous", errNeed)
		}
		p.Hazmat = 1
		return nil
	}
	if len(plain) == 0 {
		return fmt.Errorf("%w: every one of its commodities is hazardous", errNeed)
	}
	p.Hazmat, p.Commodities = 0, plain
	return nil
}

func (d *Definition) CheckNeed(set string, need *LoadNeed) error {
	template, ok := d.Templates[set]
	if !ok {
		return fmt.Errorf("%s does not send %s", d.Partner.Name, set)
	}
	_, err := template.Load.Narrow(need)
	return err
}
