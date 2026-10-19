package agentports

import (
	"slices"
	"strings"

	"github.com/emoss08/trenova/shared/stringutils"
)

type ToolNamespace struct {
	Name    string `json:"name"`
	Title   string `json:"title"`
	Summary string `json:"summary"`
	// Qualifiers are the words that read before the namespace when they open
	// a name's object: settlement_get_driver reads get_driver_settlement, the
	// kind of record it reads.
	Qualifiers []string `json:"qualifiers,omitempty"`
	// Records are the kinds of record in the namespace that keep their own
	// name: an object that opens with one reads without the namespace, so
	// receivable_get_bank_receipt reads get_bank_receipt, the record it reads.
	Records []string `json:"records,omitempty"`
}

const (
	toolNamespaceSeparator = "_"
	listVerb               = "list"
)

var toolNamespaces = []ToolNamespace{
	{
		Name:    "accounting",
		Title:   "Accounting sync",
		Summary: "The connection to the accounting system: sync records, mappings, drift and backfills.",
	},
	{
		Name:    "agent",
		Title:   "Agent",
		Summary: "The agent's own work: memories, waits, runs, profiles and finding things in Trenova.",
	},
	{
		Name:    "billing",
		Title:   "Billing queue",
		Summary: "The billing queue: transferring shipments into it and reviewing, assigning, holding and approving its items.",
	},
	{
		Name:    "carrier",
		Title:   "Carriers",
		Summary: "Outside carriers: their records, vetting, monitoring, capacity and intelligence events.",
	},
	{
		Name:    "customer",
		Title:   "Customers",
		Summary: "Customer records, their status, preferences and the updates sent to them.",
	},
	{
		Name:    "detention",
		Title:   "Detention",
		Summary: "Detention at a stop: occurrences, notices, approvals, waivers and disputes.",
	},
	{
		Name:    "dispatch",
		Title:   "Dispatch",
		Summary: "Covering moves: the board, assignments, tenders, check calls and stop actuals.",
	},
	{
		Name:    "document",
		Title:   "Documents",
		Summary: "Stored documents: search, summaries, attachments, versions and document types.",
	},
	{
		Name:    "edi",
		Title:   "EDI",
		Summary: "Trading partners and their EDI: tenders, transfers, messages, files and connections.",
	},
	{
		Name:    "equipment",
		Title:   "Equipment",
		Summary: "Tractors, trailers, equipment types and where each truck is.",
	},
	{
		Name:    "exception",
		Title:   "Exceptions",
		Summary: "The exception queue: watchtower items, service failures and their owners and status.",
	},
	{
		Name:    "factoring",
		Title:   "Factoring",
		Summary: "Selling invoices to a factor: agreements, notices, submissions, remittances and reserves.",
	},
	{
		Name:    "formula",
		Title:   "Formulas",
		Summary: "Rating formulas: templates, their schema and testing an expression.",
	},
	{
		Name:    "fuel",
		Title:   "Fuel and IFTA",
		Summary: "Fuel purchases, fuel cards, IFTA mileage and returns.",
	},
	{
		Name:    "home",
		Title:   "Home page",
		Summary: "The person's own home page: its widgets and their layout.",
	},
	{
		Name:    "inbox",
		Title:   "Inbox",
		Summary: "Inbound messages and captures: triage, replies, filing, senders and drafted loads.",
	},
	{
		Name:    "insight",
		Title:   "Insights",
		Summary: "Insights, the daily briefing and weather alerts.",
	},
	{
		Name:    "intake",
		Title:   "Shipment intake",
		Summary: "The shipment draft open on the page: accepting, correcting and placing its fields and stops.",
	},
	{
		Name:    "invoice",
		Title:   "Invoices",
		Summary: "Invoices: drafting, posting and sending them, invoice runs, statement billing, adjustments and disputes.",
	},
	{
		Name:    "ledger",
		Title:   "General ledger",
		Summary: "Journal entries, manual journals, reversals, GL accounts and fiscal periods.",
	},
	{
		Name:    "location",
		Title:   "Locations",
		Summary: "Customer and facility locations and their categories.",
	},
	{
		Name:    "masterdata",
		Title:   "Reference data",
		Summary: "Commodities, hazardous materials, hold reasons and the other lookup lists.",
	},
	{
		Name:    "order",
		Title:   "Orders",
		Summary: "Customer orders, the shipments under them and their charges.",
	},
	{
		Name:    "rate",
		Title:   "Rates",
		Summary: "Rate agreements, quotes, rating, rate imports and fuel index prices.",
	},
	{
		Name:    "receivable",
		Title:   "Receivables",
		Summary: "What customers owe: payments, bank receipts, credit memos, statements and collections.",
		Records: []string{"bank_receipt", "customer_payment"},
	},
	{
		Name:    "report",
		Title:   "Reports",
		Summary: "Reports, their runs and schedules, dashboards, table views and change alerts.",
	},
	{
		Name:       "settlement",
		Title:      "Settlements and pay",
		Summary:    "Driver and carrier settlements, pay events, advances, escrow, deductions and payroll.",
		Qualifiers: []string{"driver", "carrier"},
	},
	{
		Name:    "shipment",
		Title:   "Shipments",
		Summary: "Loads: finding, reading, tracking, creating and changing them, with their holds, comments and charges.",
	},
	{
		Name:    "web",
		Title:   "Web and portals",
		Summary: "The public web and the carrier and customer portals an organization set up.",
	},
	{
		Name:    "worker",
		Title:   "Workers",
		Summary: "Drivers and staff: records, hours, credentials, time off, training, safety and reviews.",
	},
}

func ToolNamespaces() []ToolNamespace {
	return slices.Clone(toolNamespaces)
}

func NamespaceOf(toolName string) string {
	if namespace, ok := namespaceOf(toolName); ok {
		return namespace.Name
	}

	return ""
}

func namespaceOf(toolName string) (*ToolNamespace, bool) {
	prefix, _, found := strings.Cut(toolName, toolNamespaceSeparator)
	if !found {
		return nil, false
	}
	for idx := range toolNamespaces {
		if toolNamespaces[idx].Name == prefix {
			return &toolNamespaces[idx], true
		}
	}

	return nil, false
}

// ReadingOrder is a tool's name with its verb first, the way the words are
// read: shipment_place_hold is place_shipment_hold, a bare shipment_list is
// list_shipments, and a qualifier reads before its namespace, so
// settlement_add_driver_adjustment is add_driver_settlement_adjustment. A
// name outside a namespace is returned as it is.
func ReadingOrder(toolName string) string {
	namespace, ok := namespaceOf(toolName)
	if !ok {
		return toolName
	}
	rest := strings.TrimPrefix(toolName, namespace.Name+toolNamespaceSeparator)
	verb, object, found := strings.Cut(rest, toolNamespaceSeparator)
	if found && namespace.opensWithRecord(object) {
		return verb + toolNamespaceSeparator + object
	}
	noun := namespace.Name
	if found {
		qualifier, after, _ := strings.Cut(object, toolNamespaceSeparator)
		if slices.Contains(namespace.Qualifiers, qualifier) {
			noun = qualifier + toolNamespaceSeparator + namespace.Name
			object, found = after, after != ""
		}
	}
	if !found {
		if verb == listVerb {
			return verb + toolNamespaceSeparator + stringutils.PluralNoun(noun)
		}

		return verb + toolNamespaceSeparator + noun
	}

	return verb + toolNamespaceSeparator + noun + toolNamespaceSeparator + object
}

// ToolNameFor is the namespaced name a name in reading order stands for:
// get_driver_settlement is settlement_get_driver and list_invoices is
// invoice_list. A qualified noun is the more specific reading, so
// list_carrier_settlements is a settlement list, not a carrier one. It reads words, not the registry, so a caller looking a tool
// up falls back to the name as given while its module is outside a namespace.
// A name no namespace reads is returned as it is.
func ToolNameFor(readingOrder string) string {
	verb, rest, found := strings.Cut(readingOrder, toolNamespaceSeparator)
	if !found {
		return readingOrder
	}
	for idx := range toolNamespaces {
		if toolNamespaces[idx].opensWithRecord(rest) {
			return toolNamespaces[idx].Name + toolNamespaceSeparator + readingOrder
		}
	}
	for idx := range toolNamespaces {
		for _, qualifier := range toolNamespaces[idx].Qualifiers {
			name, ok := namespacedName(toolNamespaces[idx].Name, verb, rest, qualifier)
			if ok {
				return name
			}
		}
	}
	for idx := range toolNamespaces {
		if name, ok := namespacedName(toolNamespaces[idx].Name, verb, rest, ""); ok {
			return name
		}
	}

	return readingOrder
}

func namespacedName(namespace, verb, rest, qualifier string) (string, bool) {
	noun := namespace
	parts := []string{namespace, verb}
	if qualifier != "" {
		noun = qualifier + toolNamespaceSeparator + namespace
		parts = append(parts, qualifier)
	}
	switch {
	case rest == noun, verb == listVerb && rest == stringutils.PluralNoun(noun):
	case strings.HasPrefix(rest, noun+toolNamespaceSeparator):
		parts = append(parts, strings.TrimPrefix(rest, noun+toolNamespaceSeparator))
	default:
		return "", false
	}

	return strings.Join(parts, toolNamespaceSeparator), true
}

func (n *ToolNamespace) opensWithRecord(object string) bool {
	for _, record := range n.Records {
		for _, form := range [...]string{record, stringutils.PluralNoun(record)} {
			if object == form || strings.HasPrefix(object, form+toolNamespaceSeparator) {
				return true
			}
		}
	}

	return false
}
