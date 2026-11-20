package partner

import (
	"fmt"
	"strings"
)

type scope struct {
	vars   map[string]any
	parent *scope
}

func (s *scope) with(name string, value any) *scope {
	return &scope{vars: map[string]any{name: value}, parent: s}
}

func (s *scope) Resolve(path []string) (any, error) {
	var value any
	found := false
	for current := s; current != nil; current = current.parent {
		if value, found = current.vars[path[0]]; found {
			break
		}
	}
	if !found {
		return nil, fmt.Errorf("%s is not a value here", path[0])
	}
	for index, key := range path[1:] {
		if value == nil {
			return nil, nil
		}
		fields, ok := value.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s has no fields", strings.Join(path[:index+1], "."))
		}
		if value, ok = fields[key]; !ok {
			return nil, fmt.Errorf("%s is not a field", strings.Join(path[:index+2], "."))
		}
	}
	return value, nil
}

func tenderView(p *Partner, load *Load, kind Kind, purpose string) *scope {
	stops := make([]any, 0, len(load.Stops))
	for index := range load.Stops {
		stops = append(stops, stopView(&load.Stops[index]))
	}
	return &scope{vars: map[string]any{
		"scac":    p.Trenova.SCAC,
		"purpose": purpose,
		"kind":    string(kind),
		"load": map[string]any{
			"number":     load.Number,
			"bol":        load.BOL,
			"po":         load.PO,
			"note":       load.Note,
			"payment":    load.Payment,
			"weight":     load.Weight(),
			"pieces":     load.Pieces(),
			"miles":      load.Miles,
			"rate":       load.RateCents,
			"total":      load.TotalCents,
			"tenderedAt": load.TenderedAt,
			"hazmat":     load.Hazardous(),
			"stopCount":  len(load.Stops),
		},
		"billTo": partyView(&load.BillTo),
		"equipment": map[string]any{
			"code":     load.Equipment.Code,
			"lengthFt": load.Equipment.LengthFt,
			"reefer":   load.Equipment.Reefer,
			"temp":     temperature(load.Equipment),
		},
		"stops": stops,
	}}
}

func temperature(equipment Equipment) any {
	if !equipment.Reefer {
		return nil
	}
	return equipment.TempF
}

func stopView(stop *Stop) map[string]any {
	lines := make([]any, 0, len(stop.Lines))
	for index := range stop.Lines {
		lines = append(lines, lineView(&stop.Lines[index]))
	}
	return map[string]any{
		"sequence":     stop.Sequence,
		"kind":         string(stop.Kind),
		"pickup":       stop.Kind == StopPickup,
		"delivery":     stop.Kind == StopDelivery,
		"party":        partyView(&stop.Party),
		"window":       map[string]any{"start": stop.Window.Start, "end": stop.Window.End},
		"weight":       stop.Weight,
		"pieces":       stop.Pieces,
		"pickupRef":    stop.PickupRef,
		"appointment":  stop.Appointment,
		"note":         stop.Note,
		"lines":        lines,
		"temperatures": rangesView(stop.Temperatures),
	}
}

func rangesView(ranges []TempRange) []any {
	out := make([]any, 0, len(ranges))
	for _, value := range ranges {
		out = append(out, rangeView(value))
	}
	return out
}

func rangeView(value TempRange) map[string]any {
	return map[string]any{"min": value.MinF, "max": value.MaxF}
}

func lineView(line *Line) map[string]any {
	view := map[string]any{
		"number":      line.Number,
		"description": line.Description,
		"code":        line.Code,
		"weight":      line.Weight,
		"pieces":      line.Pieces,
		"hazmat":      nil,
		"temperature": nil,
	}
	if line.Temperature != nil {
		view["temperature"] = rangeView(*line.Temperature)
	}
	if line.Hazmat != nil {
		view["hazmat"] = map[string]any{
			"un":    line.Hazmat.UN,
			"class": line.Hazmat.Class,
			"name":  line.Hazmat.Name,
		}
	}
	return view
}

func partyView(party *Party) map[string]any {
	return map[string]any{
		"code":    party.Code,
		"name":    party.Name,
		"street":  party.Street,
		"city":    party.City,
		"state":   party.State,
		"zip":     party.Zip,
		"country": party.Country,
		"contact": party.Contact,
		"phone":   party.Phone,
	}
}
