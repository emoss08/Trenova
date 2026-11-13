package podchase

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/app/drivermessage"
	"github.com/emoss08/trenova/internal/modules/agent/domain/watchtowerplay"
	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/pkg/errortypes"
	"github.com/emoss08/trenova/shared/pulid"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolDriverMessage   = "worker_send_message"
	toolDocumentRequest = "shipment_request_documents"
	requestedDocument   = "Signed proof of delivery"
)

// ask is who a shipment's proof of delivery is asked of, how, and in what
// words. It is decided once, from the record, and is the same whether a
// person sends it from the Watchtower or the billing agent files it for an
// event.
type ask struct {
	channel watchtowerplay.Channel
	// name is who is asked; did is what they did on the load.
	name string
	did  string
	// workerID is the driver, for a Dash message or a text.
	workerID pulid.ID
	// email is the address, for an email. It stays inside the request.
	email string
	// title heads a driver's message; subject an email.
	title   string
	subject string
	message string
}

// reach is what the organization can send with, beyond what is on the
// record: an email profile to send from, and a plan with text messages.
type reach struct {
	email bool
	texts bool
}

// planAsk picks who is asked: the driver who delivered it, in Dash when they
// have a login and by email when they only have an address; otherwise the
// carrier that hauled it, by email; and last the driver by text, when a
// number is all there is and the plan includes text messages, since a text
// cannot carry the paperwork back. Nil when nobody can be asked: unreachable
// says why.
func planAsk(fact *podchaseports.Facts, can reach, timezone string) *ask {
	delivered := deliveredOn(fact.DeliveredAt, timezone)
	shipment := shipmentLabel(fact)
	customer := forCustomer(fact.CustomerName)
	driver, carrier := fact.Driver, fact.Carrier

	if driver != nil && driver.OnDash {
		return &ask{
			channel:  watchtowerplay.ChannelDash,
			name:     driver.Name,
			did:      "drove",
			workerID: driver.WorkerID,
			title:    "Signed POD needed: " + proLabel(fact),
			message: clipBytes(fmt.Sprintf(
				"Hi %s, please send the signed proof of delivery for %s, delivered %s%s. "+
					"A clear photo of the signed paperwork in Dash is fine.",
				firstName(driver.Name), shipment, delivered, customer,
			), drivermessage.MaxChars),
		}
	}

	if can.email {
		body := fmt.Sprintf(
			"We still need the signed proof of delivery for %s, delivered %s%s. "+
				"Please reply with a copy so we can bill the load.",
			shipment, delivered, customer,
		)
		subject := "Signed proof of delivery for " + proLabel(fact)
		if driver != nil && driver.Email != "" {
			return &ask{
				channel: watchtowerplay.ChannelEmail,
				name:    driver.Name,
				did:     "drove",
				email:   driver.Email,
				subject: subject,
				message: body,
			}
		}
		if carrier != nil && carrier.Email != "" {
			return &ask{
				channel: watchtowerplay.ChannelEmail,
				name:    carrier.Name,
				did:     "hauled",
				email:   carrier.Email,
				subject: subject,
				message: body,
			}
		}
	}

	if driver != nil && driver.HasPhone && can.texts {
		return &ask{
			channel:  watchtowerplay.ChannelText,
			name:     driver.Name,
			did:      "drove",
			workerID: driver.WorkerID,
			title:    "Signed POD needed: " + proLabel(fact),
			message: clipBytes(fmt.Sprintf(
				"Hi %s, we still need the signed proof of delivery for %s, delivered %s%s. "+
					"Please get a copy to the office today: hand it in, or send dispatch a clear "+
					"photo of the signed paperwork.",
				firstName(driver.Name), shipment, delivered, customer,
			), drivermessage.MaxChars),
		}
	}

	return nil
}

// asked is the request in a sentence, as the item keeps it once it is sent.
func (a *ask) asked() string {
	return "Asked " + a.name + " for the signed proof of delivery " + by(a.channel)
}

func by(channel watchtowerplay.Channel) string {
	switch channel { //nolint:exhaustive // only the ways a question is sent
	case watchtowerplay.ChannelDash:
		return "in Dash"
	case watchtowerplay.ChannelText:
		return "by text"
	default:
		return "by email"
	}
}

// chaseCall is the ask as a tool call, for the billing agent's own request:
// the tool, its arguments and why, in the words its proposal carries.
type chaseCall struct {
	tool      string
	arguments map[string]any
	rationale string
}

func (a *ask) call(fact *podchaseports.Facts, timezone string) *chaseCall {
	why := fmt.Sprintf(
		"No signed proof of delivery is on file for %s, delivered %s%s. %s %s the delivery",
		shipmentLabel(fact), deliveredOn(fact.DeliveredAt, timezone),
		forCustomer(fact.CustomerName), a.name, a.did,
	)

	switch a.channel { //nolint:exhaustive // only the ways a question is sent
	case watchtowerplay.ChannelEmail:
		return &chaseCall{
			tool: toolDocumentRequest,
			arguments: map[string]any{
				"to":                 []any{a.email},
				"subject":            a.subject,
				"body":               a.message,
				"customerName":       a.name,
				"shipmentProNumber":  fact.ProNumber,
				"requestedDocuments": []any{requestedDocument},
			},
			rationale: why + ", so the request goes to them by email.",
		}
	case watchtowerplay.ChannelText:
		return &chaseCall{
			tool:      toolDriverMessage,
			arguments: a.driverArguments(fact),
			rationale: why + " and has only a phone number on file, so the request goes " +
				"to them by text.",
		}
	default:
		return &chaseCall{
			tool:      toolDriverMessage,
			arguments: a.driverArguments(fact),
			rationale: why + " and is on Dash, so the request goes to them there.",
		}
	}
}

func (a *ask) driverArguments(fact *podchaseports.Facts) map[string]any {
	return map[string]any{
		"workerId":   a.workerID.String(),
		"title":      a.title,
		"message":    a.message,
		"priority":   "medium",
		"shipmentId": fact.ShipmentID.String(),
	}
}

// unreachable says, for a person, why nobody can be asked for a shipment's
// proof of delivery. It names people and what is missing, never an address.
// Each message is one literal so it reaches the translation catalog whole.
//
//nolint:lll // a message is one literal
func unreachable(fact *podchaseports.Facts, can reach) *errortypes.BusinessError {
	driver, carrier := fact.Driver, fact.Carrier
	driverEmail := driver != nil && driver.Email != ""
	carrierEmail := carrier != nil && carrier.Email != ""

	switch {
	case driver == nil && carrier == nil:
		return errortypes.NewBusinessError("Nobody is assigned to the delivering move, so there is nobody to ask for the proof of delivery. Upload it here.")
	case !can.email && (driverEmail || carrierEmail):
		return errortypes.NewBusinessError("No email profile is set up to send from, so the proof of delivery cannot be asked for by email. Upload it here, or ask an administrator to add one in the email settings.")
	case driver != nil && driver.HasPhone && !can.texts:
		return errortypes.NewBusinessError("{0} can only be reached by text, and the organization's plan does not include text messages. Upload the proof of delivery here, or add their email.", driver.Name)
	case driver != nil && carrier != nil:
		return errortypes.NewBusinessError("{0} has no Dash login, email or phone number on file and {1} has no email on file, so neither can be asked for the proof of delivery. Upload it here, or add a way to reach them.", driver.Name, carrier.Name)
	case driver != nil:
		return errortypes.NewBusinessError("{0} has no Dash login, email or phone number on file, so they cannot be asked for the proof of delivery. Upload it here, or add a way to reach them.", driver.Name)
	default:
		return errortypes.NewBusinessError("{0} has no email on file, so they cannot be asked for the proof of delivery. Upload it here, or add their email.", carrier.Name)
	}
}

func shipmentLabel(fact *podchaseports.Facts) string {
	label := proLabel(fact)
	if bol := strings.TrimSpace(fact.BOL); bol != "" {
		label += " (BOL " + bol + ")"
	}

	return label
}

// proLabel names the load the way a person does. A load with no PRO is
// "the load": its record id is never put in front of anyone.
func proLabel(fact *podchaseports.Facts) string {
	if pro := strings.TrimSpace(fact.ProNumber); pro != "" {
		return "PRO " + pro
	}

	return "the load"
}

func forCustomer(customer string) string {
	if customer = strings.TrimSpace(customer); customer == "" {
		return ""
	}

	return " for " + customer
}

func firstName(name string) string {
	if fields := strings.Fields(name); len(fields) > 0 {
		return fields[0]
	}

	return "there"
}

func deliveredOn(at int64, timezone string) string {
	if at <= 0 {
		return "recently"
	}

	return "on " + time.Unix(at, 0).In(timeutils.LoadLocation(timezone)).Format("Jan 2, 2006")
}

// clipBytes keeps text within limit bytes, the bound the message tool
// checks, without cutting a character in two.
func clipBytes(text string, limit int) string {
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
