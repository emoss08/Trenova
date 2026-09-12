package invoicerunservice

import (
	"fmt"
	"strings"

	"github.com/emoss08/trenova/internal/core/domain/customer"
	"github.com/emoss08/trenova/internal/core/ports/repositories"
)

const (
	// keySeparator joins the customer id to the discriminator. A unit separator
	// cannot appear in a PO or BOL, so "PO-1" + "|2" can never collide with
	// "PO-1|2" the way a printable separator would allow.
	keySeparator = "\x1f"

	// missingValue stands in for a blank discriminator. It is not a legal user
	// string, so a shipment with no PO can never land in the same group as one
	// whose PO literally reads "No PO".
	missingValue = "\x00none"

	// standaloneShipment prefixes the key of a shipment grouped on its own because
	// it has no order. The NUL byte keeps it out of the space an order id can
	// occupy, so a shipment id can never collide with an order's key.
	standaloneShipment = "\x00shipment:"
)

// GroupKeyResult is the stable key a group is identified by plus the label an
// operator reads.
type GroupKeyResult struct {
	Key   string
	Label string
}

// GroupKeyFor is the group a candidate belongs to under a split key.
//
// Every key starts with the customer, because an invoice never spans customers.
// A blank discriminator gets its own bucket rather than being folded into the
// first real value: silently merging thirty unrelated loads that happen to have
// no PO onto one invoice is how a customer gets an invoice they refuse.
func GroupKeyFor(
	splitBy customer.InvoiceSplitKey,
	candidate *repositories.ConsolidationCandidate,
) GroupKeyResult {
	customerID := candidate.CustomerID.String()

	switch splitBy {
	case customer.InvoiceSplitKeyCustomerAndPONumber:
		return discriminated(customerID, candidate.OrderPONumber, "PO %s", "No PO")

	case customer.InvoiceSplitKeyCustomerAndShipmentBOL:
		return discriminated(customerID, candidate.ShipmentBOL, "BOL %s", "No BOL")

	case customer.InvoiceSplitKeyCustomerAndOrder:
		// A shipment booked without an order is its own commercial unit, so it gets
		// its own invoice. Unlike a missing PO, there is nothing these shipments
		// share: bucketing them together would merge every standalone job the
		// customer shipped onto one invoice, which is not what "one invoice per
		// order" means.
		if candidate.OrderID.IsNil() {
			label := candidate.ProNumber
			if label == "" {
				label = candidate.ShipmentID.String()
			}
			return GroupKeyResult{
				Key:   customerID + keySeparator + standaloneShipment + candidate.ShipmentID.String(),
				Label: label,
			}
		}
		label := candidate.OrderNumber
		if label == "" {
			label = candidate.OrderID.String()
		}
		return GroupKeyResult{
			Key:   customerID + keySeparator + candidate.OrderID.String(),
			Label: label,
		}

	case customer.InvoiceSplitKeyCustomerAndOrigin:
		return discriminated(customerID, candidate.OriginKey, "From %s", "Unknown origin")

	case customer.InvoiceSplitKeyCustomerAndDestination:
		return discriminated(customerID, candidate.DestinationKey, "To %s", "Unknown destination")

	case customer.InvoiceSplitKeyCustomerAndServiceType:
		return discriminated(
			customerID,
			candidate.ServiceTypeCode,
			"%s",
			"No service type",
		)

	default:
		label := candidate.CustomerName
		if label == "" {
			label = customerID
		}
		return GroupKeyResult{Key: customerID, Label: label}
	}
}

func discriminated(customerID, raw, labelFormat, missingLabel string) GroupKeyResult {
	value := strings.ToUpper(strings.TrimSpace(raw))
	if value == "" {
		return GroupKeyResult{
			Key:   customerID + keySeparator + missingValue,
			Label: missingLabel,
		}
	}

	return GroupKeyResult{
		Key:   customerID + keySeparator + value,
		Label: fmt.Sprintf(labelFormat, value),
	}
}

// CandidateGroup is one proposed invoice: the members that would sit on it and
// the settings that shape it.
type CandidateGroup struct {
	Key     string
	Label   string
	Part    int
	Parts   int
	First   *repositories.ConsolidationCandidate
	Members []*repositories.ConsolidationCandidate
}

// GroupCandidates is the whole split decision: which invoices a set of eligible
// queue items becomes, under each customer's own split key and cap.
//
// Shared by the run builder and the open-statements view on purpose. A biller
// watching a statement accumulate must see the same invoices the run will
// actually produce; two implementations of "how does this split" would drift,
// and the drift would only surface as a surprise on billing day.
//
// Insertion order is preserved so members keep the query's ordering — customer,
// then service date, then PRO — and the same input always yields the same
// proposal.
func GroupCandidates(candidates []*repositories.ConsolidationCandidate) []CandidateGroup {
	ordered := make([]string, 0, len(candidates))
	byKey := make(map[string][]*repositories.ConsolidationCandidate, len(candidates))
	labels := make(map[string]string, len(candidates))
	settings := make(map[string]*repositories.ConsolidationCandidate, len(candidates))

	for _, candidate := range candidates {
		result := GroupKeyFor(candidate.SplitBy, candidate)
		if _, seen := byKey[result.Key]; !seen {
			ordered = append(ordered, result.Key)
			labels[result.Key] = result.Label
			settings[result.Key] = candidate
		}
		byKey[result.Key] = append(byKey[result.Key], candidate)
	}

	groups := make([]CandidateGroup, 0, len(ordered))
	for _, key := range ordered {
		first := settings[key]
		parts := SplitOversized(byKey[key], int(first.MaxShipmentsPerInvoice))
		for i, part := range parts {
			groups = append(groups, CandidateGroup{
				Key:     PartKey(key, i+1, len(parts)),
				Label:   PartLabel(labels[key], i+1, len(parts)),
				Part:    i + 1,
				Parts:   len(parts),
				First:   first,
				Members: part,
			})
		}
	}

	return groups
}

// SplitOversized breaks a group that exceeds the customer's cap into numbered
// parts, preserving order.
//
// Splitting happens after the primary grouping rather than during it, so the
// boundaries fall in the same places on a re-preview of the same input. A cap of
// zero means unbounded.
func SplitOversized[T any](members []T, maxPerInvoice int) [][]T {
	if maxPerInvoice <= 0 || len(members) <= maxPerInvoice {
		return [][]T{members}
	}

	parts := make([][]T, 0, (len(members)+maxPerInvoice-1)/maxPerInvoice)
	for start := 0; start < len(members); start += maxPerInvoice {
		end := min(start+maxPerInvoice, len(members))
		parts = append(parts, members[start:end])
	}

	return parts
}

// PartLabel names one part of a split group, so an operator can tell which of
// three invoices for the same PO they are looking at.
func PartLabel(label string, part, total int) string {
	if total <= 1 {
		return label
	}

	return fmt.Sprintf("%s (%d of %d)", label, part, total)
}

// PartKey keeps each part of a split group unique against the group key index.
func PartKey(key string, part, total int) string {
	if total <= 1 {
		return key
	}

	return fmt.Sprintf("%s%s#%d", key, keySeparator, part)
}
