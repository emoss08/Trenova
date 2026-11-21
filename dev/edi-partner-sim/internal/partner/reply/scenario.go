package reply

import (
	"fmt"
	"math/rand/v2"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/spoken"
)

type Outcome string

const (
	OutcomeApplied  Outcome = "applied"
	OutcomeHeld     Outcome = "held"
	OutcomeRejected Outcome = "rejected"
)

type Scenario struct {
	Seed       uint64  `yaml:"seed"`
	Decline    string  `yaml:"decline"`
	Delay      float64 `yaml:"delay"`
	DeltaCents int64   `yaml:"deltaCents"`
	Misfooted  bool    `yaml:"misfooted"`
	Stray      bool    `yaml:"stray"`
}

type Reply struct {
	Set         string
	At          time.Time
	Transaction partner.Transaction
	Want        Outcome
	Note        string
}

var declineReasons = []string{
	"No driver near the pickup",
	"Equipment unavailable that day",
	"Rate below our lane minimum",
	"Consignee hours do not fit",
	"Lane outside our service area",
}

func DrawScenario(seed uint64) Scenario {
	rng := rand.New(rand.NewPCG(seed, 990214210))
	scenario := Scenario{Seed: seed, Delay: 0.3}
	if rng.IntN(5) == 0 {
		scenario.Decline = declineReasons[rng.IntN(len(declineReasons))]
		return scenario
	}
	switch roll := rng.IntN(20); {
	case roll < 8:
	case roll < 13:
		scenario.DeltaCents = int64(5_000 + 500*rng.IntN(51))
	case roll < 16:
		scenario.DeltaCents = -int64(2_500 + 500*rng.IntN(26))
	default:
		scenario.Misfooted = true
	}
	scenario.Stray = rng.IntN(8) == 0
	return scenario
}

func (s Scenario) String() string {
	text := fmt.Sprintf("seed %d", s.Seed)
	switch {
	case s.Decline != "":
		return text + " decline"
	case s.Misfooted:
		text += " misfooted invoice"
	case s.DeltaCents != 0:
		text += fmt.Sprintf(" invoice %+d cents", s.DeltaCents)
	}
	if s.Stray {
		text += " stray status"
	}
	return text
}

func (c *Carrier) Converse(tender *Tender, scenario Scenario) ([]Reply, error) {
	response, err := c.Response(tender, scenario)
	if err != nil {
		return nil, err
	}
	replies := []Reply{response}
	if scenario.Decline != "" {
		return replies, nil
	}
	statuses, err := c.Statuses(tender, scenario)
	if err != nil {
		return nil, err
	}
	replies = append(replies, statuses...)
	if scenario.Stray {
		stray, strayErr := c.Stray(tender, scenario)
		if strayErr != nil {
			return nil, strayErr
		}
		replies = append(replies, stray)
	}
	invoice, err := c.Invoice(tender, scenario)
	if err != nil {
		return nil, err
	}
	return append(replies, invoice), nil
}

func digits(rng *rand.Rand, count int) string {
	out := make([]byte, count)
	for index := range out {
		out[index] = byte('0' + rng.IntN(10))
	}
	if out[0] == '0' {
		out[0] = '1'
	}
	return string(out)
}

func (c *Carrier) number(tender *Tender, seed uint64, what string) string {
	return c.Definition.Partner.Them.SCAC[:2] + digits(tender.rng(seed, what), 8)
}

func minutes(count int) time.Duration {
	return time.Duration(count) * time.Minute
}

func (s Scenario) Describe() string {
	if s.Decline != "" {
		return "Declined: " + s.Decline
	}
	text := "Accepted, delivered and invoiced"
	switch {
	case s.Misfooted:
		text = "Invoice whose lines do not add up to its total"
	case s.DeltaCents > 0:
		text = "Invoice " + spoken.Dollars(s.DeltaCents) + " over the tender"
	case s.DeltaCents < 0:
		text = "Invoice " + spoken.Dollars(s.DeltaCents) + " under the tender"
	}
	if s.Stray {
		text += ", and a status for a load never tendered"
	}
	return text
}
