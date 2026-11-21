package conform

import (
	"strings"

	"github.com/emoss08/trenova/edi-partner-sim/internal/partner"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/reply"
	"github.com/emoss08/trenova/edi-partner-sim/internal/partner/spoken"
)

var statusPhrases = map[string]string{
	"X3": "saying it arrived at pickup",
	"AF": "saying it left pickup",
	"X1": "saying it arrived at delivery",
	"CD": "saying it left delivery",
	"D1": "saying it delivered",
	"X6": "saying it is en route",
	"AG": "with an estimated delivery",
	"SD": "saying the load is delayed",
}

func possessive(name string) string {
	if strings.HasSuffix(name, "s") {
		return name + "'"
	}
	return name + "'s"
}

func tenderText(def *partner.Definition) string {
	return "Write Trenova's 204 tendering the load to " + def.Partner.Display
}

func replyText(display string, r *reply.Reply, scenario reply.Scenario) string {
	whose := "Read " + possessive(display) + " " + r.Set
	switch r.Set {
	case "990":
		if scenario.Decline != "" {
			return whose + " declining the load"
		}
		return whose + " accepting the load"
	case "214":
		if r.Want == reply.OutcomeRejected {
			return whose + " for a load Trenova never tendered"
		}
		code := reply.StatusCode(*r)
		if phrase, ok := statusPhrases[code]; ok {
			return whose + " " + phrase
		}
		if code != "" {
			return whose + " with status " + code
		}
		return whose
	case "210":
		switch {
		case scenario.Misfooted:
			return whose + ", its lines not adding up to its total"
		case scenario.DeltaCents > 0:
			return whose + ", " + spoken.Dollars(scenario.DeltaCents) + " over the tender"
		case scenario.DeltaCents < 0:
			return whose + ", " + spoken.Dollars(scenario.DeltaCents) + " under the tender"
		}
		return whose + " at the tendered total"
	}
	return whose
}
