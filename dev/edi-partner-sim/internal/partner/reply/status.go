package reply

import (
	"math/rand/v2"
	"time"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
)

const (
	partnerPickup   = partner.StopPickup
	partnerDelivery = partner.StopDelivery
	reasonNormal    = "NS"
)

const (
	StatusArrivedPickup   = "X3"
	StatusDepartedPickup  = "AF"
	StatusArrivedDelivery = "X1"
	StatusDelivered       = "D1"
	StatusDelayed         = "SD"
)

var delayReasons = []string{"AO", "AI", "AH", "AF", "AG"}

type statusEvent struct {
	code   string
	reason string
	at     time.Time
	stop   *TenderStop
}

func (c *Carrier) Statuses(tender *Tender, scenario Scenario) ([]Reply, error) {
	rng := tender.rng(scenario.Seed, "214")
	trailer := c.Definition.Partner.Them.SCAC[:2] + digits(tender.rng(scenario.Seed, "trailer"), 5)
	pro := c.number(tender, scenario.Seed, "pro")
	events := sequence(tender, scenario, rng)
	replies := make([]Reply, 0, len(events))
	for index := range events {
		reply, err := c.status(tender.Key, tender.view(), pro, trailer, &events[index], rng)
		if err != nil {
			return nil, err
		}
		replies = append(replies, reply)
	}
	return replies, nil
}

func (c *Carrier) Stray(tender *Tender, scenario Scenario) (Reply, error) {
	rng := tender.rng(scenario.Seed, "stray")
	pickup := tender.first(partnerPickup)
	event := statusEvent{
		code:   StatusArrivedPickup,
		reason: reasonNormal,
		at:     pickup.Start.Add(minutes(rng.IntN(60))),
		stop:   pickup,
	}
	key := c.number(tender, scenario.Seed, "stray")
	trailer := c.Definition.Partner.Them.SCAC[:2] + digits(rng, 5)
	reply, err := c.status(
		key,
		map[string]any{"bol": nil, "po": nil},
		c.number(tender, scenario.Seed, "stray pro"),
		trailer,
		&event,
		rng,
	)
	if err != nil {
		return Reply{}, err
	}
	reply.Want = OutcomeRejected
	reply.Note = "a status for " + key + ", a load Trenova never tendered"
	return reply, nil
}

func (c *Carrier) status(
	key string,
	references map[string]any,
	pro, trailer string,
	event *statusEvent,
	rng *rand.Rand,
) (Reply, error) {
	vars := map[string]any{
		"scac":    c.Definition.Partner.Them.SCAC,
		"key":     key,
		"pro":     pro,
		"trailer": trailer,
		"tender":  references,
		"event": map[string]any{
			"code":    event.code,
			"reason":  event.reason,
			"at":      event.at,
			"stop":    event.stop.Sequence,
			"city":    event.stop.City,
			"state":   event.stop.State,
			"country": countryOf(event.stop),
		},
	}
	transaction, err := c.render("214", vars)
	if err != nil {
		return Reply{}, err
	}
	return Reply{
		Set:         "214",
		At:          event.at.Add(minutes(2 + rng.IntN(14))),
		Transaction: transaction,
		Want:        OutcomeApplied,
	}, nil
}

func sequence(tender *Tender, scenario Scenario, rng *rand.Rand) []statusEvent {
	var events []statusEvent
	var clock time.Time
	for index := range tender.Stops {
		stop := &tender.Stops[index]
		arrive := stop.Start.Add(minutes(rng.IntN(91) - 30))
		if !clock.IsZero() && arrive.Before(clock.Add(30*time.Minute)) {
			arrive = clock.Add(30 * time.Minute)
		}
		if scenario.Delay > 0 && rng.Float64() < scenario.Delay {
			delayed := arrive.Add(-45 * time.Minute)
			if !clock.IsZero() && delayed.Before(clock.Add(10*time.Minute)) {
				delayed = clock.Add(10 * time.Minute)
			}
			events = append(events, statusEvent{
				code:   StatusDelayed,
				reason: delayReasons[rng.IntN(len(delayReasons))],
				at:     delayed,
				stop:   stop,
			})
			arrive = delayed.Add(minutes(60 + rng.IntN(120)))
		}
		leave := arrive.Add(minutes(30 + rng.IntN(61)))
		arrived, left := StatusArrivedDelivery, StatusDelivered
		if stop.Kind == partnerPickup {
			arrived, left = StatusArrivedPickup, StatusDepartedPickup
			leave = arrive.Add(minutes(45 + rng.IntN(76)))
		}
		events = append(events,
			statusEvent{code: arrived, reason: reasonNormal, at: arrive, stop: stop},
			statusEvent{code: left, reason: reasonNormal, at: leave, stop: stop},
		)
		clock = leave
	}
	return events
}

func countryOf(stop *TenderStop) string {
	if stop.Country == "" {
		return "US"
	}
	return stop.Country
}
