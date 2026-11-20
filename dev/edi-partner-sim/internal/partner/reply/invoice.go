package reply

import (
	"errors"
	"fmt"
	"time"
)

const stopOffCents = 7_500

var errNoLinehaul = errors.New("reply: the invoice delta leaves no linehaul to bill")

const (
	ChargeLinehaul  = "400"
	ChargeFuel      = "FUE"
	ChargeStopOff   = "SOC"
	ChargeDetention = "DET"
)

type InvoiceLine struct {
	Description string
	Code        string
	Qualifier   string
	RateCents   int64
	AmountCents int64
}

func (c *Carrier) Invoice(tender *Tender, scenario Scenario) (Reply, error) {
	rng := tender.rng(scenario.Seed, "210")
	lines, err := charges(tender, scenario, 12+rng.IntN(11))
	if err != nil {
		return Reply{}, err
	}
	billed := int64(0)
	for _, line := range lines {
		billed += line.AmountCents
	}
	total := billed
	if scenario.Misfooted {
		total += int64(100 * (10 + rng.IntN(90)))
	}
	delivery := tender.last(partnerDelivery)
	delivered := delivery.End
	if delivered.IsZero() {
		delivered = delivery.Start
	}
	at := delivered.AddDate(0, 0, 1+rng.IntN(3)).In(c.Definition.Partner.Location())
	at = time.Date(at.Year(), at.Month(), at.Day(), 6+rng.IntN(10), 5*rng.IntN(12), 0, 0,
		at.Location())
	views := make([]any, 0, len(lines))
	for index, line := range lines {
		view := map[string]any{
			"sequence":    index + 1,
			"description": line.Description,
			"code":        line.Code,
			"qualifier":   line.Qualifier,
			"rate":        nil,
			"amount":      line.AmountCents,
		}
		if line.RateCents > 0 {
			view["rate"] = line.RateCents
		}
		views = append(views, view)
	}
	vars := map[string]any{
		"scac":   c.Definition.Partner.Them.SCAC,
		"pro":    c.number(tender, scenario.Seed, "pro"),
		"tender": tender.view(),
		"invoice": map[string]any{
			"number":    c.Definition.Partner.Them.SCAC[:2] + "F" + digits(rng, 7),
			"billed":    at,
			"delivered": delivered,
			"total":     total,
		},
		"parties": []any{
			partyView("SH", tender.first(partnerPickup)),
			partyView("CN", delivery),
		},
		"lines": views,
	}
	transaction, err := c.render("210", vars)
	if err != nil {
		return Reply{}, err
	}
	reply := Reply{Set: "210", At: at, Transaction: transaction, Want: OutcomeApplied}
	switch {
	case scenario.Misfooted:
		reply.Want = OutcomeHeld
		reply.Note = fmt.Sprintf("lines add to %d cents, the total says %d", billed, total)
	case total != tender.TotalCents:
		reply.Want = OutcomeHeld
		reply.Note = fmt.Sprintf("bills %d cents against a tendered %d", total, tender.TotalCents)
	}
	return reply, nil
}

func charges(tender *Tender, scenario Scenario, fuelPercent int) ([]InvoiceLine, error) {
	total := tender.TotalCents
	fuel := (total*int64(fuelPercent) + 50) / 100
	stopOffs := int64(max(len(tender.Stops)-2, 0)) * stopOffCents
	linehaul := total - fuel - stopOffs
	if linehaul <= 0 {
		fuel, stopOffs, linehaul = 0, 0, total
	}
	if scenario.DeltaCents < 0 {
		linehaul += scenario.DeltaCents
	}
	if linehaul <= 0 {
		return nil, fmt.Errorf("%w: %s %+d", errNoLinehaul, tender.Key, scenario.DeltaCents)
	}
	lines := []InvoiceLine{{
		Description: "LINEHAUL",
		Code:        ChargeLinehaul,
		Qualifier:   "FR",
		RateCents:   linehaul,
		AmountCents: linehaul,
	}}
	if fuel > 0 {
		lines = append(lines, InvoiceLine{
			Description: "FUEL SURCHARGE",
			Code:        ChargeFuel,
			AmountCents: fuel,
		})
	}
	for stopOffs > 0 {
		lines = append(lines, InvoiceLine{
			Description: "STOP OFF",
			Code:        ChargeStopOff,
			AmountCents: stopOffCents,
		})
		stopOffs -= stopOffCents
	}
	if scenario.DeltaCents > 0 {
		lines = append(lines, InvoiceLine{
			Description: "DETENTION",
			Code:        ChargeDetention,
			AmountCents: scenario.DeltaCents,
		})
	}
	return lines, nil
}

func partyView(role string, stop *TenderStop) map[string]any {
	return map[string]any{
		"role":    role,
		"name":    stop.Name,
		"city":    stop.City,
		"state":   stop.State,
		"zip":     stop.Zip,
		"country": countryOf(stop),
	}
}
