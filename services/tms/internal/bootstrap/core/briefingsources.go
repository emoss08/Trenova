package core

import (
	"github.com/emoss08/trenova/internal/modules/agent/app/briefingservice/briefingfacts"
	"github.com/emoss08/trenova/internal/modules/agent/ports/agentdecisionqueueports"
	"github.com/emoss08/trenova/internal/modules/agent/ports/watchtowerports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/billingqueueports"
	"github.com/emoss08/trenova/internal/modules/billing/ports/customerpaymentports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/detentionports"
	"github.com/emoss08/trenova/internal/modules/operations/ports/dispatchboardports"
	"github.com/emoss08/trenova/internal/modules/workforce/ports/workercredentialports"
	"go.uber.org/fx"
)

// BriefingSourceParams collects what the morning's figures are drawn from.
// It is a separate constructor rather than an fx.In on the facts builder so
// the builder stays a plain struct that a test can fill with two sources
// and leave the rest nil.
type BriefingSourceParams struct {
	fx.In

	Watchtower  watchtowerports.WatchtowerRepository
	Decisions   agentdecisionqueueports.AgentDecisionQueueService
	Board       dispatchboardports.DispatchConsoleRepository
	Credentials workercredentialports.WorkerCredentialRepository
	Billing     billingqueueports.BillingQueueRepository
	Payments    customerpaymentports.CustomerPaymentRepository
	Detention   detentionports.DetentionOccurrenceRepository
}

func newBriefingSources(p BriefingSourceParams) briefingfacts.Sources {
	return briefingfacts.Sources{
		Watchtower:  p.Watchtower,
		Decisions:   p.Decisions,
		Board:       p.Board,
		Credentials: p.Credentials,
		Billing:     p.Billing,
		Payments:    p.Payments,
		Detention:   p.Detention,
	}
}
