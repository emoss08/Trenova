package podchasejobs

import (
	"fmt"
	"strings"
	"time"

	"github.com/emoss08/trenova/internal/modules/agent/ports/podchaseports"
	"github.com/emoss08/trenova/shared/timeutils"
)

const (
	toolDriverMessage     = "worker_send_message"
	toolDocumentRequest   = "shipment_request_documents"
	maxDriverMessageBytes = 500
	requestedDocument     = "Signed proof of delivery"
)

// chaseCall is the one request a shipment's proof of delivery is asked for
// with, and why, in the words its proposal carries.
type chaseCall struct {
	tool      string
	arguments map[string]any
	rationale string
}

// planCall picks who is asked: the driver who delivered it, when Dash reaches
// them; otherwise the carrier that hauled it, by email from the billing
// sender. Nil when nobody can be asked, which is the agent's to judge.
func planCall(fact *podchaseports.Facts, sender, timezone string) *chaseCall {
	delivered := deliveredOn(fact.DeliveredAt, timezone)
	shipment := shipmentLabel(fact)

	if driver := fact.Driver; driver != nil && driver.OnDash {
		message := fmt.Sprintf(
			"Hi %s, please send the signed proof of delivery for %s, delivered %s%s. "+
				"A clear photo of the signed paperwork in Dash is fine.",
			firstName(driver.Name), shipment, delivered, forCustomer(fact.CustomerName),
		)
		return &chaseCall{
			tool: toolDriverMessage,
			arguments: map[string]any{
				"workerId":   driver.WorkerID.String(),
				"title":      "Signed POD needed: " + proLabel(fact),
				"message":    clipBytes(message, maxDriverMessageBytes),
				"priority":   "medium",
				"shipmentId": fact.ShipmentID.String(),
			},
			rationale: fmt.Sprintf(
				"No signed proof of delivery is on file for %s, delivered %s%s. %s drove the "+
					"delivery and is on Dash, so the request goes to them there.",
				shipment, delivered, forCustomer(fact.CustomerName), driver.Name,
			),
		}
	}

	if carrier := fact.Carrier; carrier != nil && carrier.Email != "" && sender != "" {
		return &chaseCall{
			tool: toolDocumentRequest,
			arguments: map[string]any{
				"profileId": sender,
				"to":        []any{carrier.Email},
				"subject":   "Signed proof of delivery for " + proLabel(fact),
				"body": fmt.Sprintf(
					"We still need the signed proof of delivery for %s, delivered %s%s. "+
						"Please reply with a copy so we can bill the load.",
					shipment, delivered, forCustomer(fact.CustomerName),
				),
				"customerName":       carrier.Name,
				"shipmentProNumber":  fact.ProNumber,
				"requestedDocuments": []any{requestedDocument},
			},
			rationale: fmt.Sprintf(
				"No signed proof of delivery is on file for %s, delivered %s%s. %s hauled the "+
					"delivery, so the request goes to them by email.",
				shipment, delivered, forCustomer(fact.CustomerName), carrier.Name,
			),
		}
	}

	return nil
}

func shipmentLabel(fact *podchaseports.Facts) string {
	label := proLabel(fact)
	if bol := strings.TrimSpace(fact.BOL); bol != "" {
		label += " (BOL " + bol + ")"
	}

	return label
}

func proLabel(fact *podchaseports.Facts) string {
	if pro := strings.TrimSpace(fact.ProNumber); pro != "" {
		return "PRO " + pro
	}

	return "shipment " + fact.ShipmentID.String()
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
