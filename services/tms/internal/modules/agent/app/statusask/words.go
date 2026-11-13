package statusask

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtower"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/haulerports"
)

// loadName names the load the way a person does. A load with no PRO is
// "this load": its record id is never put in front of anyone.
func loadName(hauler *haulerports.Hauler, bare string) string {
	if pro := strings.TrimSpace(hauler.ProNumber); pro != "" {
		return "PRO " + pro
	}

	return bare
}

// at is where the stop is, as it fits a sentence; empty when the item does
// not say.
func at(item *watchtower.Item) string {
	if item.Panel == nil || item.Panel.Stop == nil {
		return ""
	}
	if place := strings.TrimSpace(item.Panel.Stop.Place); place != "" {
		return " at " + place
	}

	return ""
}

// driverWords is the message to the driver for the kind: what happened, on
// which load and where, and the one thing they are asked to do.
func driverWords(item *watchtower.Item, hauler *haulerports.Hauler) (title, message string) {
	name := firstName(hauler.Driver.Name)
	load := loadName(hauler, "your load")
	where := at(item)

	switch item.SourceKind { //nolint:exhaustive // only the kinds this playbook works
	case watchtower.SourceMissedPickup:
		title = "Pickup status needed: " + load
		message = fmt.Sprintf(
			"Hi %s, the pickup window%s for %s has closed and we have no arrival recorded. "+
				"Are you on site? Please mark your arrival in Dash, or reply with where you "+
				"are and when you expect to be there.", name, where, load)
	case watchtower.SourceMissedDelivery:
		title = "Delivery status needed: " + load
		message = fmt.Sprintf(
			"Hi %s, the delivery window%s for %s has closed and we have no arrival recorded. "+
				"Are you on site? Please mark your arrival in Dash, or reply with where you "+
				"are and when you expect to be there.", name, where, load)
	case watchtower.SourceEtaAtRisk:
		title = "ETA needed: " + load
		message = fmt.Sprintf(
			"Hi %s, %s is projected to arrive%s after its window. Please reply with where "+
				"you are and your best arrival time, and tell us if anything is holding you up.",
			name, load, where)
	default:
		title = "Check call needed: " + load
		message = fmt.Sprintf(
			"Hi %s, we have not heard from you on %s within the check call window. Please "+
				"send a check call in Dash, or reply with where you are.", name, load)
	}

	return title, clip(message, drivermessage.MaxChars)
}

// emailWords is the request as an email: the template greets whoever it
// goes to and signs it, so the body is only the question.
func emailWords(item *watchtower.Item, hauler *haulerports.Hauler) (subject, body string) {
	load := loadName(hauler, "this load")
	where := at(item)
	subject = "Status needed: " + loadName(hauler, "your load")

	switch item.SourceKind { //nolint:exhaustive // only the kinds this playbook works
	case watchtower.SourceMissedPickup:
		body = fmt.Sprintf(
			"The pickup window%s for %s has closed and we have no arrival recorded. Where "+
				"is the truck now, and when do you expect it to arrive?", where, load)
	case watchtower.SourceMissedDelivery:
		body = fmt.Sprintf(
			"The delivery window%s for %s has closed and we have no arrival recorded. Where "+
				"is the truck now, and when do you expect it to arrive?", where, load)
	case watchtower.SourceEtaAtRisk:
		body = fmt.Sprintf(
			"%s is projected to arrive%s after its window. Where is the truck now, and what "+
				"is your best arrival time? Please tell us if anything is holding it up.",
			capitalized(load), where)
	default:
		body = fmt.Sprintf(
			"We have not had a check call on %s within the agreed window. Where is the truck "+
				"now, and is it still on schedule?", load)
	}

	return subject, body
}

// asked is the request in a sentence, as the item keeps it once it is sent.
func asked(name string, hauler *haulerports.Hauler, channel watchtowerplay.Channel) string {
	how := "by email"
	switch channel { //nolint:exhaustive // only the ways a question is sent
	case watchtowerplay.ChannelDash:
		how = "in Dash"
	case watchtowerplay.ChannelText:
		how = "by text"
	}

	return "Asked " + name + " where " + loadName(hauler, "the load") + " is " + how
}

func capitalized(text string) string {
	if text == "" {
		return text
	}

	return strings.ToUpper(text[:1]) + text[1:]
}

func firstName(name string) string {
	if fields := strings.Fields(name); len(fields) > 0 {
		return fields[0]
	}

	return "there"
}

// clip keeps text within limit bytes, the bound the message tool checks,
// without cutting a character in two.
func clip(text string, limit int) string {
	if len(text) <= limit {
		return text
	}
	cut := 0
	for i := range text {
		if i > limit {
			break
		}
		cut = i
	}

	return text[:cut]
}
