package api

import (
	"github.com/emoss08/trenova/internal/core/ports/repositories"
	"github.com/emoss08/trenova/internal/core/ports/services"
	"github.com/emoss08/trenova/internal/core/services/briefingservice/briefingfacts"
	"go.uber.org/fx"
)

// BriefingSourceParams collects what the morning's figures are drawn from.
// It is a separate constructor rather than an fx.In on the facts builder so
// the builder stays a plain struct that a test can fill with two sources
// and leave the rest nil.
type BriefingSourceParams struct {
	fx.In

	Watchtower  repositories.WatchtowerRepository
	Decisions   services.AgentDecisionQueueService
	Board       repositories.DispatchConsoleRepository
	Credentials repositories.WorkerCredentialRepository
	Billing     repositories.BillingQueueRepository
	Payments    repositories.CustomerPaymentRepository
	Detention   repositories.DetentionOccurrenceRepository
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
