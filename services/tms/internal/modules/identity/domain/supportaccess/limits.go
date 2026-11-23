package supportaccess

import "slices"

type Limit int

const (
	LimitPeople Limit = iota + 1
	LimitSingleSignOn
	LimitAPIKeys
	LimitSupportAccess
	LimitCredentials
	LimitDriverPersonal
	LimitDevices
	LimitSecurityReview
	LimitPlan
	LimitAgents
	LimitFactoring
	LimitDocumentTemplates
	LimitEDI
	LimitPaymentDestinations
	LimitExport
	LimitOutbound
	LimitDeskConversations
)

var limitLabels = map[Limit]string{ //nolint:gosec // customer-facing sentences that name credentials, holding none
	LimitPeople:              "Change users, roles or how people sign in",
	LimitSingleSignOn:        "See or change single sign-on or user provisioning",
	LimitAPIKeys:             "See or create API keys",
	LimitSupportAccess:       "See or change Trenova support access",
	LimitCredentials:         "See or change the stored credentials of integrations, accounting, email, mailboxes, AI providers or carrier portals",
	LimitDriverPersonal:      "See drivers' protected personal details",
	LimitDevices:             "Pair devices or invite drivers to the driver portal",
	LimitSecurityReview:      "Act on security alerts, reviews or legal holds",
	LimitPlan:                "Change what your plan includes",
	LimitAgents:              "Change agents, their extensions or their spending limits",
	LimitFactoring:           "Change factoring companies, agreements or remittances",
	LimitDocumentTemplates:   "Change document templates",
	LimitEDI:                 "Change EDI trading partners, their keys or their endpoints",
	LimitPaymentDestinations: "Change where money is sent: remittance instructions, remit-to addresses, the factor an invoice or customer pays, or your company address printed on invoices",
	LimitExport:              "Export data in bulk",
	LimitOutbound:            "Send email, share links or scheduled reports to anyone outside your organization, or change what your customers read on an invoice or in an email",
	LimitDeskConversations:   "Read the Desk conversations of people in your organization",
}

func (l Limit) Label() string {
	return limitLabels[l]
}

func Limits() []Limit {
	seen := make(map[Limit]struct{}, len(limitLabels))
	out := make([]Limit, 0, len(limitLabels))
	add := func(l Limit) {
		if _, ok := seen[l]; ok {
			return
		}
		seen[l] = struct{}{}
		out = append(out, l)
	}
	for _, r := range denied {
		add(r.limit)
	}
	for _, d := range paymentDestinations {
		add(d.Limit)
	}
	add(LimitExport)
	add(LimitOutbound)
	slices.Sort(out)
	return out
}
