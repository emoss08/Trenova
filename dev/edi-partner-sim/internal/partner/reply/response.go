package reply

import "time"

const (
	decisionAccept  = "A"
	decisionDecline = "D"
)

func (c *Carrier) Response(tender *Tender, scenario Scenario) (Reply, error) {
	rng := tender.rng(scenario.Seed, "990")
	pickup := tender.first(partnerPickup)
	at := pickup.Start.Add(-18*time.Hour + minutes(rng.IntN(240))).
		In(c.Definition.Partner.Location())
	vars := map[string]any{
		"scac":        c.Definition.Partner.Them.SCAC,
		"tender":      tender.view(),
		"respondedAt": at,
		"decision":    decisionAccept,
		"pro":         c.number(tender, scenario.Seed, "pro"),
		"reason":      nil,
	}
	if scenario.Decline != "" {
		vars["decision"] = decisionDecline
		vars["pro"] = nil
		vars["reason"] = scenario.Decline
	}
	transaction, err := c.render("990", vars)
	if err != nil {
		return Reply{}, err
	}
	return Reply{
		Set:         "990",
		At:          at,
		Transaction: transaction,
		Want:        OutcomeApplied,
	}, nil
}
