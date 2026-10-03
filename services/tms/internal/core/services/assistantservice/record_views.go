package assistantservice

import (
	"strconv"
	"strings"

	"github.com/emoss08/trenova/shared/typeutils"
)

/*
A record card's view: the few things a person reads a record of this kind
for, laid out for it, rather than every field the tool returned. A shipment
is its route and who is moving it; an invoice is what it bills and whether
it is paid; a billing queue item is what it would bill and what stands in
the way.

The view is built from the tool's JSON, as the model reads it, so it depends
on the tool's contract and not its Go types. A field the result does not
carry is left out, never guessed. Facts are keyed, not labelled: the reader
names them in their own language.
*/

const (
	payloadView = "view"

	viewShipment         = "shipment"
	viewInvoice          = "invoice"
	viewBillingQueueItem = "billing_queue_item"

	stopPickup   = "Pickup"
	stopDelivery = "Delivery"
)

// recordView is the card's view for a record of a kind that has one, nil
// for any other.
func recordView(entity string, result map[string]any) map[string]any {
	switch entity {
	case viewShipment:
		return shipmentView(result)
	case viewInvoice:
		return invoiceView(result)
	case viewBillingQueueItem:
		return billingQueueView(result)
	default:
		return nil
	}
}

type fact struct {
	key   string
	value any
}

// facts keeps the facts that hold a value, in the order given.
func facts(entries ...fact) []any {
	out := make([]any, 0, len(entries))
	for _, entry := range entries {
		switch value := entry.value.(type) {
		case nil:
			continue
		case string:
			if strings.TrimSpace(value) == "" {
				continue
			}
		case float64:
			if value == 0 {
				continue
			}
		case int64:
			if value == 0 {
				continue
			}
		}
		out = append(out, map[string]any{"key": entry.key, "value": entry.value})
	}

	return out
}

func objectOf(value any) map[string]any {
	object, _ := value.(map[string]any)
	return object
}

func listOf(value any) []map[string]any {
	items, _ := value.([]any)
	out := make([]map[string]any, 0, len(items))
	for _, item := range items {
		if object := objectOf(item); object != nil {
			out = append(out, object)
		}
	}

	return out
}

func textOf(object map[string]any, key string) string {
	return typeutils.StringOfTrimmed(object[key])
}

func numberOf(object map[string]any, key string) float64 {
	switch value := object[key].(type) {
	case float64:
		return value
	case int64:
		return float64(value)
	case int:
		return float64(value)
	case string:
		parsed, err := strconv.ParseFloat(value, 64)
		if err == nil {
			return parsed
		}
	}

	return 0
}

// shipmentView is the shipment as the design draws it: where it started and
// where it is going, how far along, and who and what is moving it.
func shipmentView(result map[string]any) map[string]any {
	var stops []map[string]any
	var assignment, carrier map[string]any
	var miles float64
	for _, move := range listOf(result["moves"]) {
		stops = append(stops, listOf(move["stops"])...)
		if assignment == nil {
			assignment = objectOf(move["assignment"])
		}
		if carrier == nil {
			carrier = objectOf(move["carrier"])
		}
		if loaded, _ := move["loaded"].(bool); loaded {
			miles += numberOf(move, "distance")
		}
	}

	view := map[string]any{
		"type":     viewShipment,
		"status":   textOf(result, "status"),
		"subtitle": textOf(objectOf(result["customer"]), "name"),
	}

	if len(stops) > 0 {
		first, last := stops[0], stops[len(stops)-1]
		for _, stop := range stops {
			if textOf(stop, "type") == stopPickup {
				first = stop
				break
			}
		}
		for idx := len(stops) - 1; idx >= 0; idx-- {
			if textOf(stops[idx], "type") == stopDelivery {
				last = stops[idx]
				break
			}
		}
		view["from"] = stopView(first, true)
		view["to"] = stopView(last, false)

		done := 0
		for _, stop := range stops {
			if textOf(stop, "actualDeparture") != "" || textOf(stop, "status") == "Completed" {
				done++
			}
		}
		progress := float64(done) / float64(len(stops))
		if textOf(result, "status") == "Completed" {
			progress = 1
		}
		view["progress"] = progress
	}

	var weight int64
	for _, commodity := range listOf(result["commodities"]) {
		weight += int64(numberOf(commodity, "weight"))
	}
	rating := objectOf(result["rating"])

	view["facts"] = facts(
		fact{"driver", textOf(assignment, "primaryWorker")},
		fact{"tractor", textOf(assignment, "tractor")},
		fact{"trailer", textOf(assignment, "trailer")},
		fact{"carrier", textOf(carrier, "carrier")},
		fact{"weight", weight},
		fact{"miles", miles},
		fact{"rate", textOf(rating, "totalChargeAmount")},
		fact{"bol", textOf(result, "bol")},
		fact{"serviceType", textOf(result, "serviceType")},
		fact{"shipmentType", textOf(result, "shipmentType")},
	)

	return view
}

// stopView is one end of the route: the place, the city and the time that
// matters there, which is when it happened if it has, and the window if not.
func stopView(stop map[string]any, origin bool) map[string]any {
	city := textOf(stop, "city")
	if state := textOf(stop, "state"); state != "" {
		if city != "" {
			city += ", " + state
		} else {
			city = state
		}
	}

	view := map[string]any{"city": city, "place": textOf(stop, "location")}
	switch {
	case origin && textOf(stop, "actualDeparture") != "":
		view["when"], view["at"] = "departed", textOf(stop, "actualDeparture")
	case textOf(stop, "actualArrival") != "":
		view["when"], view["at"] = "arrived", textOf(stop, "actualArrival")
	case textOf(stop, "scheduledWindowStart") != "":
		view["when"], view["at"] = "scheduled", textOf(stop, "scheduledWindowStart")
	}

	return view
}

// invoiceView is what an invoice bills, to whom, and whether it is paid.
func invoiceView(result map[string]any) map[string]any {
	return map[string]any{
		"type":     viewInvoice,
		"status":   textOf(result, "status"),
		"subtitle": textOf(result, "billToName"),
		"amount": map[string]any{
			"total":    textOf(result, "totalAmount"),
			"balance":  textOf(result, "balanceDue"),
			"currency": textOf(result, "currency"),
		},
		"facts": facts(
			fact{"settlement", textOf(result, "settlementStatus")},
			fact{"dueDate", result["dueDate"]},
			fact{"invoiceDate", result["invoiceDate"]},
			fact{"paymentTerm", textOf(result, "paymentTerm")},
			fact{"proNumber", textOf(result, "proNumber")},
			fact{"bol", textOf(result, "bol")},
			fact{"sent", textOf(result, "sendStatus")},
			fact{"lines", numberOf(result, "lineCount")},
		),
	}
}

// billingQueueView is what an item would bill, who is on it, and what stands
// between it and an invoice.
func billingQueueView(result map[string]any) map[string]any {
	view := map[string]any{
		"type":     viewBillingQueueItem,
		"status":   textOf(result, "status"),
		"subtitle": textOf(result, "billTo"),
		"amount":   map[string]any{"total": textOf(result, "amount")},
		"facts": facts(
			fact{"proNumber", textOf(result, "proNumber")},
			fact{"bol", textOf(result, "bol")},
			fact{"biller", textOf(result, "assignedBiller")},
			fact{"billType", textOf(result, "billType")},
			fact{"ageDays", numberOf(result, "ageDays")},
			fact{"exception", textOf(result, "exceptionReason")},
		),
	}
	if _, ok := result["canApprove"]; ok {
		readiness := objectOf(result["readiness"])
		blockers := len(listOf(result["detentionHolds"]))
		for _, key := range []string{"missingDocuments", "validationFailures"} {
			if items, ok := readiness[key].([]any); ok {
				blockers += len(items)
			}
		}
		view["ready"] = map[string]any{
			"canApprove": result["canApprove"],
			"blockedBy":  textOf(result, "approvalBlockedBy"),
			"blockers":   blockers,
		}
	}

	return view
}
