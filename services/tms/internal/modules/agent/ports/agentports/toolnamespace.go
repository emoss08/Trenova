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
		Title:   "Billing and invoices",
		Summary: "The billing queue, invoices, invoice runs, adjustments, disputes and sending invoices.",
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
	},
	{
		Name:    "report",
		Title:   "Reports",
		Summary: "Reports, their runs and schedules, dashboards, table views and change alerts.",
	},
	{
		Name:    "settlement",
		Title:   "Settlements and pay",
		Summary: "Driver and carrier settlements, pay events, advances, escrow, deductions and payroll.",
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
	prefix, _, found := strings.Cut(toolName, toolNamespaceSeparator)
	if !found {
		return ""
	}
	for idx := range toolNamespaces {
		if toolNamespaces[idx].Name == prefix {
			return prefix
		}
	}

	return ""
}

func ReadingOrder(toolName string) string {
	namespace := NamespaceOf(toolName)
	if namespace == "" {
		return toolName
	}
	rest := strings.TrimPrefix(toolName, namespace+toolNamespaceSeparator)
	verb, object, found := strings.Cut(rest, toolNamespaceSeparator)
	switch {
	case !found && verb == listVerb:
		return verb + toolNamespaceSeparator + stringutils.PluralNoun(namespace)
	case !found:
		return verb + toolNamespaceSeparator + namespace
	}

	return verb + toolNamespaceSeparator + namespace + toolNamespaceSeparator + object
}
